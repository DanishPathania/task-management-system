package service

import (
	"context"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/google/uuid"
)

type AnalyticsService struct {
	analyticsRepo *repository.AnalyticsRepository
	projectRepo   *repository.ProjectRepository
}

func NewAnalyticsService(analyticsRepo *repository.AnalyticsRepository, projectRepo *repository.ProjectRepository) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		projectRepo:   projectRepo,
	}
}

func (s *AnalyticsService) GetProjectAnalytics(ctx context.Context, userID, projectID uuid.UUID) (*models.ProjectAnalytics, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.analyticsRepo.GetProjectAnalytics(ctx, projectID)
}

func (s *AnalyticsService) GetDashboardAnalytics(ctx context.Context, userID uuid.UUID) (*models.DashboardAnalytics, error) {
	return s.analyticsRepo.GetDashboardAnalytics(ctx, userID)
}
