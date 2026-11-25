package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/redis"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/validation"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("forbidden: insufficient permissions")
)

type ProjectService struct {
	projectRepo     *repository.ProjectRepository
	userRepo        *repository.UserRepository
	redisClient     *redis.Client
	hub             *websocket.Hub
	activityService *ActivityService
	notifService    *NotificationService
}

func NewProjectService(
	projectRepo *repository.ProjectRepository,
	userRepo *repository.UserRepository,
	redisClient *redis.Client,
	hub *websocket.Hub,
	activityService *ActivityService,
	notifService *NotificationService,
) *ProjectService {
	return &ProjectService{
		projectRepo:     projectRepo,
		userRepo:        userRepo,
		redisClient:     redisClient,
		hub:             hub,
		activityService: activityService,
		notifService:    notifService,
	}
}

func (s *ProjectService) CreateProject(ctx context.Context, userID uuid.UUID, name, description string) (*models.Project, error) {
	name = strings.TrimSpace(name)
	if err := validation.ValidateProjectInput(name); err != nil {
		return nil, err
	}

	project, err := s.projectRepo.Create(ctx, name, description, userID)
	if err != nil {
		return nil, err
	}

	// Cache project
	if s.redisClient != nil {
		s.redisClient.SetProject(ctx, project, 5*time.Minute)
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, project.ID, userID, "project.created", "project", &project.ID, project.Name, nil, project.Name)
	}

	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	return s.projectRepo.ListForUser(ctx, userID)
}

func (s *ProjectService) GetProject(ctx context.Context, userID, projectID uuid.UUID) (*models.Project, error) {
	// Verify membership
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	// Check cache
	if s.redisClient != nil {
		if cached, err := s.redisClient.GetProject(ctx, projectID); err == nil && cached != nil {
			cached.Role = role
			return cached, nil
		}
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	project.Role = role

	// Update cache
	if s.redisClient != nil {
		s.redisClient.SetProject(ctx, project, 5*time.Minute)
	}

	return project, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, name, description string) (*models.Project, error) {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if role != models.RoleAdmin {
		return nil, ErrForbidden
	}

	if name != "" {
		if err := validation.ValidateProjectInput(name); err != nil {
			return nil, err
		}
	}

	project, err := s.projectRepo.Update(ctx, projectID, name, description)
	if err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.InvalidateProject(ctx, projectID)
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "project.updated", "project", &projectID, project.Name, nil, map[string]interface{}{
			"name":        project.Name,
			"description": project.Description,
		})
	}

	return project, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if role != models.RoleAdmin {
		return ErrForbidden
	}

	if err := s.projectRepo.Delete(ctx, projectID); err != nil {
		return err
	}

	if s.redisClient != nil {
		s.redisClient.InvalidateProject(ctx, projectID)
	}

	return nil
}

// Members

func (s *ProjectService) ListMembers(ctx context.Context, userID, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		if cached, err := s.redisClient.GetMembers(ctx, projectID); err == nil && cached != nil {
			return cached, nil
		}
	}

	members, err := s.projectRepo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.SetMembers(ctx, projectID, members, 5*time.Minute)
	}

	return members, nil
}

func (s *ProjectService) AddMember(ctx context.Context, actorID, projectID uuid.UUID, email, role string) (*models.ProjectMember, error) {
	actorRole, err := s.projectRepo.GetMemberRole(ctx, projectID, actorID)
	if err != nil {
		return nil, err
	}
	if actorRole != models.RoleAdmin {
		return nil, ErrForbidden
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if err := validation.ValidateEmail(email); err != nil {
		return nil, err
	}

	if err := validation.ValidateRole(role); err != nil {
		return nil, err
	}

	// Verify target user exists by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	member, err := s.projectRepo.AddMember(ctx, projectID, user.ID, role)
	if err != nil {
		return nil, err
	}
	member.UserName = user.Name
	member.UserEmail = user.Email

	// Invalidate cache
	if s.redisClient != nil {
		s.redisClient.InvalidateProject(ctx, projectID)
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, actorID, "member.added", "member", &user.ID, user.Name, nil, role)
	}

	if s.notifService != nil {
		projName := "a project"
		if proj, err := s.projectRepo.GetByID(ctx, projectID); err == nil {
			projName = proj.Name
		}
		s.notifService.NotifyUser(
			ctx,
			user.ID,
			&projectID,
			nil,
			&actorID,
			NotifMemberAdded,
			"Added to project",
			fmt.Sprintf("You were added to \"%s\" as %s", projName, role),
			nil,
		)
	}

	// Broadcast real-time event
	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventMemberAdded,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   member,
		})
	}

	return member, nil
}

func (s *ProjectService) UpdateMemberRole(ctx context.Context, actorID, projectID, targetUserID uuid.UUID, newRole string) (*models.ProjectMember, error) {
	actorRole, err := s.projectRepo.GetMemberRole(ctx, projectID, actorID)
	if err != nil {
		return nil, err
	}
	if actorRole != models.RoleAdmin {
		return nil, ErrForbidden
	}

	if err := validation.ValidateRole(newRole); err != nil {
		return nil, err
	}

	member, err := s.projectRepo.UpdateMemberRole(ctx, projectID, targetUserID, newRole)
	if err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.InvalidateProject(ctx, projectID)
	}

	if s.activityService != nil {
		targetName := targetUserID.String()
		if u, err := s.userRepo.GetByID(ctx, targetUserID); err == nil {
			targetName = u.Name
		}
		s.activityService.Record(ctx, projectID, actorID, "member.role_changed", "member", &targetUserID, targetName, nil, newRole)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventRoleChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   member,
		})
	}

	return member, nil
}

func (s *ProjectService) RemoveMember(ctx context.Context, actorID, projectID, targetUserID uuid.UUID) error {
	actorRole, err := s.projectRepo.GetMemberRole(ctx, projectID, actorID)
	if err != nil {
		return err
	}
	if actorRole != models.RoleAdmin {
		return ErrForbidden
	}

	if err := s.projectRepo.RemoveMember(ctx, projectID, targetUserID); err != nil {
		return err
	}

	if s.redisClient != nil {
		s.redisClient.InvalidateProject(ctx, projectID)
	}

	if s.activityService != nil {
		targetName := targetUserID.String()
		if u, err := s.userRepo.GetByID(ctx, targetUserID); err == nil {
			targetName = u.Name
		}
		s.activityService.Record(ctx, projectID, actorID, "member.removed", "member", &targetUserID, targetName, nil, nil)
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventMemberRemoved,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"user_id": targetUserID,
			},
		})
	}

	return nil
}
