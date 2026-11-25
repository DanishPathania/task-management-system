package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type WorkflowService struct {
	workflowRepo *repository.WorkflowRepository
	projectRepo  *repository.ProjectRepository
	hub          *websocket.Hub
}

func NewWorkflowService(
	workflowRepo *repository.WorkflowRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo: workflowRepo,
		projectRepo:  projectRepo,
		hub:          hub,
	}
}

func (s *WorkflowService) ListStatuses(ctx context.Context, userID, projectID uuid.UUID) ([]*models.ProjectStatus, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.workflowRepo.ListStatuses(ctx, projectID)
}

func (s *WorkflowService) CreateStatus(ctx context.Context, userID, projectID uuid.UUID, key, name, color string, position int, isDone bool, transitions []string) (*models.ProjectStatus, error) {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if role != models.RoleAdmin {
		return nil, ErrForbidden
	}
	if strings.TrimSpace(key) == "" || strings.TrimSpace(name) == "" {
		return nil, errors.New("status key and name are required")
	}
	st, err := s.workflowRepo.CreateStatus(ctx, projectID, key, name, color, position, isDone, transitions)
	if err != nil {
		return nil, err
	}
	s.broadcastWorkflow(projectID)
	return st, nil
}

func (s *WorkflowService) DeleteStatus(ctx context.Context, userID, projectID, statusID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if role != models.RoleAdmin {
		return ErrForbidden
	}
	if err := s.workflowRepo.DeleteStatus(ctx, projectID, statusID); err != nil {
		return err
	}
	s.broadcastWorkflow(projectID)
	return nil
}

func (s *WorkflowService) ListLabels(ctx context.Context, userID, projectID uuid.UUID) ([]*models.ProjectLabel, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.workflowRepo.ListLabels(ctx, projectID)
}

func (s *WorkflowService) CreateLabel(ctx context.Context, userID, projectID uuid.UUID, name, color string) (*models.ProjectLabel, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("label name is required")
	}
	l, err := s.workflowRepo.CreateLabel(ctx, projectID, name, color)
	if err != nil {
		return nil, err
	}
	s.broadcastWorkflow(projectID)
	return l, nil
}

func (s *WorkflowService) DeleteLabel(ctx context.Context, userID, projectID, labelID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if role != models.RoleAdmin {
		return ErrForbidden
	}
	if err := s.workflowRepo.DeleteLabel(ctx, projectID, labelID); err != nil {
		return err
	}
	s.broadcastWorkflow(projectID)
	return nil
}

func (s *WorkflowService) ListTaskLabels(ctx context.Context, userID, projectID, taskID uuid.UUID) ([]*models.ProjectLabel, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.workflowRepo.ListTaskLabels(ctx, taskID)
}

func (s *WorkflowService) AttachLabel(ctx context.Context, userID, projectID, taskID, labelID uuid.UUID) error {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return err
	}
	return s.workflowRepo.AttachLabelToTask(ctx, projectID, taskID, labelID)
}

func (s *WorkflowService) DetachLabel(ctx context.Context, userID, projectID, taskID, labelID uuid.UUID) error {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return err
	}
	return s.workflowRepo.DetachLabelFromTask(ctx, taskID, labelID)
}

func (s *WorkflowService) broadcastWorkflow(projectID uuid.UUID) {
	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventWorkflowChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   map[string]interface{}{"project_id": projectID},
		})
	}
}
