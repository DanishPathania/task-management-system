package service

import (
	"context"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type DependencyService struct {
	depRepo         *repository.DependencyRepository
	projectRepo     *repository.ProjectRepository
	hub             *websocket.Hub
	activityService *ActivityService
}

func NewDependencyService(
	depRepo *repository.DependencyRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
	activityService *ActivityService,
) *DependencyService {
	return &DependencyService{
		depRepo:         depRepo,
		projectRepo:     projectRepo,
		hub:             hub,
		activityService: activityService,
	}
}

func (s *DependencyService) AddDependency(ctx context.Context, userID, projectID, taskID, dependsOnTaskID uuid.UUID) (*models.TaskDependency, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}

	dep, err := s.depRepo.Add(ctx, projectID, taskID, dependsOnTaskID, userID)
	if err != nil {
		return nil, err
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "dependency.added", "task", &taskID, dep.TaskTitle, nil, dep.DependsOnTitle)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventDependencyChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   dep,
		})
	}

	return dep, nil
}

func (s *DependencyService) ListTaskDependencies(ctx context.Context, userID, projectID, taskID uuid.UUID) (*models.TaskDependenciesResponse, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.depRepo.ListForTask(ctx, projectID, taskID)
}

func (s *DependencyService) ListProjectDependencies(ctx context.Context, userID, projectID uuid.UUID) ([]*models.TaskDependency, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.depRepo.ListForProject(ctx, projectID)
}

func (s *DependencyService) RemoveDependency(ctx context.Context, userID, projectID, taskID, dependencyID uuid.UUID) error {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return err
	}

	if err := s.depRepo.Remove(ctx, projectID, taskID, dependencyID); err != nil {
		return err
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "dependency.removed", "task", &taskID, "", nil, nil)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventDependencyChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"task_id":       taskID,
				"dependency_id": dependencyID,
			},
		})
	}

	return nil
}
