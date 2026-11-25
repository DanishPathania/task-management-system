package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/validation"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type RecurringService struct {
	recRepo     *repository.RecurringRepository
	projectRepo *repository.ProjectRepository
	hub         *websocket.Hub
}

func NewRecurringService(
	recRepo *repository.RecurringRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
) *RecurringService {
	return &RecurringService{
		recRepo:     recRepo,
		projectRepo: projectRepo,
		hub:         hub,
	}
}

func (s *RecurringService) Create(ctx context.Context, userID, projectID uuid.UUID, input *models.RecurringTask) (*models.RecurringTask, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, validation.ErrTitleRequired
	}
	if input.Priority == "" {
		input.Priority = models.PriorityMedium
	}
	if err := validation.ValidatePriorityOnly(input.Priority); err != nil {
		return nil, err
	}
	if input.Frequency != "DAILY" && input.Frequency != "WEEKLY" && input.Frequency != "MONTHLY" {
		return nil, errors.New("frequency must be DAILY, WEEKLY, or MONTHLY")
	}
	if input.IntervalValue < 1 {
		input.IntervalValue = 1
	}
	if input.StartDate.IsZero() {
		input.StartDate = time.Now().UTC()
	}
	if input.NextRunAt.IsZero() {
		input.NextRunAt = input.StartDate
	}
	input.ProjectID = projectID
	input.CreatedBy = userID
	input.IsActive = true

	return s.recRepo.Create(ctx, input)
}

func (s *RecurringService) List(ctx context.Context, userID, projectID uuid.UUID) ([]*models.RecurringTask, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.recRepo.ListByProject(ctx, projectID)
}

func (s *RecurringService) Toggle(ctx context.Context, userID, projectID, recurringID uuid.UUID, active bool) (*models.RecurringTask, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.recRepo.ToggleActive(ctx, projectID, recurringID, active)
}

func (s *RecurringService) Delete(ctx context.Context, userID, projectID, recurringID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if role != models.RoleAdmin {
		return ErrForbidden
	}
	return s.recRepo.Delete(ctx, projectID, recurringID)
}

func (s *RecurringService) ProcessDue(ctx context.Context) {
	created, err := s.recRepo.ProcessDue(ctx)
	if err != nil || len(created) == 0 {
		return
	}
	if s.hub != nil {
		for _, t := range created {
			s.hub.Broadcast(&models.RealtimeEvent{
				Type:      "event",
				Event:     models.EventTaskCreated,
				EventID:   uuid.New(),
				ProjectID: t.ProjectID,
				Timestamp: time.Now().UTC(),
				Payload:   t,
			})
		}
	}
}
