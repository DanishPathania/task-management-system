package service

import (
	"context"
	"log"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type ActivityService struct {
	activityRepo *repository.ActivityRepository
	projectRepo  *repository.ProjectRepository
	userRepo     *repository.UserRepository
	hub          *websocket.Hub
}

func NewActivityService(
	activityRepo *repository.ActivityRepository,
	projectRepo *repository.ProjectRepository,
	userRepo *repository.UserRepository,
	hub *websocket.Hub,
) *ActivityService {
	return &ActivityService{
		activityRepo: activityRepo,
		projectRepo:  projectRepo,
		userRepo:     userRepo,
		hub:          hub,
	}
}

func (s *ActivityService) Record(
	ctx context.Context,
	projectID uuid.UUID,
	actorID uuid.UUID,
	action string,
	entityType string,
	entityID *uuid.UUID,
	entityTitle string,
	oldValue interface{},
	newValue interface{},
) {
	actorName := "System"
	if actorID != uuid.Nil && s.userRepo != nil {
		if u, err := s.userRepo.GetByID(ctx, actorID); err == nil {
			actorName = u.Name
		}
	}

	var actorPtr *uuid.UUID
	if actorID != uuid.Nil {
		actorPtr = &actorID
	}

	entry, err := s.activityRepo.Create(ctx, &models.ActivityLog{
		ProjectID:   projectID,
		ActorID:     actorPtr,
		ActorName:   actorName,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityTitle: entityTitle,
		OldValue:    oldValue,
		NewValue:    newValue,
	})
	if err != nil {
		log.Printf("Failed to record activity (%s): %v", action, err)
		return
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventActivityCreated,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   entry,
		})
	}
}

func (s *ActivityService) ListProjectActivity(ctx context.Context, userID, projectID uuid.UUID, page, limit int) (*models.PaginatedActivityResponse, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.activityRepo.ListByProject(ctx, projectID, page, limit)
}
