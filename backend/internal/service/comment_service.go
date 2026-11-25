package service

import (
	"context"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/validation"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type CommentService struct {
	commentRepo     *repository.CommentRepository
	taskRepo        *repository.TaskRepository
	projectRepo     *repository.ProjectRepository
	hub             *websocket.Hub
	activityService *ActivityService
	notifService    *NotificationService
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
	activityService *ActivityService,
	notifService *NotificationService,
) *CommentService {
	return &CommentService{
		commentRepo:     commentRepo,
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		hub:             hub,
		activityService: activityService,
		notifService:    notifService,
	}
}

func (s *CommentService) AddComment(ctx context.Context, userID, projectID, taskID uuid.UUID, content string) (*models.Comment, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	// Verify task belongs to project
	task, err := s.taskRepo.GetByID(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if err := validation.ValidateCommentInput(content); err != nil {
		return nil, err
	}

	comment, err := s.commentRepo.Create(ctx, taskID, userID, content)
	if err != nil {
		return nil, err
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "comment.created", "comment", &task.ID, task.Title, nil, comment.Content)
	}

	if s.notifService != nil {
		s.notifService.NotifyCommentAndMentions(ctx, projectID, task, userID, comment.UserName, comment.Content)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventCommentCreated,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   comment,
		})
	}

	return comment, nil
}

func (s *CommentService) ListComments(ctx context.Context, userID, projectID, taskID uuid.UUID) ([]*models.Comment, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	_, err = s.taskRepo.GetByID(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}

	return s.commentRepo.ListByTask(ctx, taskID)
}

func (s *CommentService) DeleteComment(ctx context.Context, userID, projectID, taskID, commentID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}

	task, err := s.taskRepo.GetByID(ctx, projectID, taskID)
	if err != nil {
		return err
	}

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	// Author or Admin can delete
	if role != models.RoleAdmin && comment.UserID != userID {
		return ErrForbidden
	}

	if err := s.commentRepo.Delete(ctx, commentID); err != nil {
		return err
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "comment.deleted", "comment", &task.ID, task.Title, comment.Content, nil)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventCommentDeleted,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"task_id":    taskID,
				"comment_id": commentID,
			},
		})
	}

	return nil
}
