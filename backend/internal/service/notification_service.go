package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

const (
	NotifTaskAssigned  = "task_assigned"
	NotifTaskDue       = "task_due"
	NotifTaskOverdue   = "task_overdue"
	NotifMention       = "mention"
	NotifComment       = "comment"
	NotifMemberAdded   = "member_added"
	NotifStatusChanged = "status_changed"
)

var mentionRegex = regexp.MustCompile(`@([A-Za-z0-9._+-]+(?:@[A-Za-z0-9.-]+\.[A-Za-z]{2,})?)`)

type NotificationService struct {
	notifRepo   *repository.NotificationRepository
	projectRepo *repository.ProjectRepository
	hub         *websocket.Hub
}

func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
) *NotificationService {
	return &NotificationService{
		notifRepo:   notifRepo,
		projectRepo: projectRepo,
		hub:         hub,
	}
}

func (s *NotificationService) NotifyUser(
	ctx context.Context,
	userID uuid.UUID,
	projectID *uuid.UUID,
	taskID *uuid.UUID,
	actorID *uuid.UUID,
	notifType string,
	title string,
	message string,
	dedupeKey *string,
) {
	if userID == uuid.Nil {
		return
	}
	if actorID != nil && *actorID == userID {
		return
	}

	// Enforce project membership before delivering project-related notifications
	if projectID != nil && s.projectRepo != nil {
		if _, err := s.projectRepo.GetMemberRole(ctx, *projectID, userID); err != nil {
			return
		}
	}

	// Respect user notification preferences
	prefs, err := s.notifRepo.GetPreferences(ctx, userID)
	if err == nil && prefs != nil {
		switch notifType {
		case NotifTaskAssigned:
			if !prefs.TaskAssigned {
				return
			}
		case NotifTaskDue, NotifTaskOverdue:
			if !prefs.TaskDue {
				return
			}
		case NotifMention:
			if !prefs.Mentions {
				return
			}
		case NotifComment:
			if !prefs.Comments {
				return
			}
		case NotifStatusChanged:
			if !prefs.StatusChanges {
				return
			}
		case NotifMemberAdded:
			if !prefs.MemberAdded {
				return
			}
		}
	}

	n := &models.Notification{
		UserID:    userID,
		ProjectID: projectID,
		TaskID:    taskID,
		ActorID:   actorID,
		Type:      notifType,
		Title:     title,
		Message:   message,
	}

	created, inserted, err := s.notifRepo.Create(ctx, n, dedupeKey)
	if err != nil {
		log.Printf("Failed to create notification: %v", err)
		return
	}
	if !inserted || created == nil {
		return
	}

	if s.hub != nil {
		pid := uuid.Nil
		if projectID != nil {
			pid = *projectID
		}
		target := userID
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:         "event",
			Event:        models.EventNotificationCreated,
			EventID:      uuid.New(),
			ProjectID:    pid,
			TargetUserID: &target,
			Timestamp:    time.Now().UTC(),
			Payload:      created,
		})
	}
}

// NotifyCommentAndMentions parses @mentions and notifies mentioned members + task assignee/creator.
func (s *NotificationService) NotifyCommentAndMentions(
	ctx context.Context,
	projectID uuid.UUID,
	task *models.Task,
	actorID uuid.UUID,
	actorName string,
	content string,
) {
	if s.projectRepo == nil || task == nil {
		return
	}

	members, err := s.projectRepo.ListMembers(ctx, projectID)
	if err != nil {
		return
	}

	mentionedUsers := make(map[uuid.UUID]bool)
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		token := strings.ToLower(strings.TrimSpace(m[1]))
		for _, member := range members {
			if member.UserID == actorID {
				continue
			}
			emailLower := strings.ToLower(member.UserEmail)
			nameSlug := strings.ToLower(strings.ReplaceAll(member.UserName, " ", ""))
			firstName := strings.ToLower(strings.Fields(member.UserName)[0])
			if token == emailLower || token == nameSlug || (firstName != "" && token == firstName) {
				if !mentionedUsers[member.UserID] {
					mentionedUsers[member.UserID] = true
					s.NotifyUser(
						ctx,
						member.UserID,
						&projectID,
						&task.ID,
						&actorID,
						NotifMention,
						"You were mentioned in a comment",
						fmt.Sprintf("%s mentioned you on \"%s\"", actorName, task.Title),
						nil,
					)
				}
			}
		}
	}

	// Notify assignee and creator (if not already notified by mention and not actor)
	candidates := []uuid.UUID{task.CreatedBy}
	if task.AssignedTo != nil {
		candidates = append(candidates, *task.AssignedTo)
	}
	notified := make(map[uuid.UUID]bool)
	for _, uid := range candidates {
		if uid == actorID || mentionedUsers[uid] || notified[uid] {
			continue
		}
		notified[uid] = true
		s.NotifyUser(
			ctx,
			uid,
			&projectID,
			&task.ID,
			&actorID,
			NotifComment,
			"New comment on task",
			fmt.Sprintf("%s commented on \"%s\"", actorName, task.Title),
			nil,
		)
	}
}

func (s *NotificationService) CheckDueAndOverdueTasks(ctx context.Context) {
	records, err := s.notifRepo.FindDueAndOverdueTasks(ctx)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	dateKey := now.Format("2006-01-02")

	for _, rec := range records {
		pid := rec.ProjectID
		tid := rec.TaskID
		if rec.DueDate.Before(now) {
			key := fmt.Sprintf("overdue:%s:%s", rec.TaskID.String(), dateKey)
			s.NotifyUser(
				ctx,
				rec.AssignedTo,
				&pid,
				&tid,
				nil,
				NotifTaskOverdue,
				"Task overdue",
				fmt.Sprintf("\"%s\" is overdue (was due %s)", rec.Title, rec.DueDate.Format("Jan 2")),
				&key,
			)
		} else {
			key := fmt.Sprintf("due:%s:%s", rec.TaskID.String(), dateKey)
			s.NotifyUser(
				ctx,
				rec.AssignedTo,
				&pid,
				&tid,
				nil,
				NotifTaskDue,
				"Task due soon",
				fmt.Sprintf("\"%s\" is due on %s", rec.Title, rec.DueDate.Format("Jan 2")),
				&key,
			)
		}
	}
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, limit int) (*repository.PaginatedNotifications, error) {
	return s.notifRepo.ListForUser(ctx, userID, unreadOnly, page, limit)
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return s.notifRepo.MarkRead(ctx, userID, notificationID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifRepo.MarkAllRead(ctx, userID)
}

func (s *NotificationService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	return s.notifRepo.GetPreferences(ctx, userID)
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, prefs *models.NotificationPreferences) (*models.NotificationPreferences, error) {
	return s.notifRepo.UpdatePreferences(ctx, prefs)
}
