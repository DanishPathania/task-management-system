package service

import (
	"context"
	"fmt"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/validation"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo        *repository.TaskRepository
	projectRepo     *repository.ProjectRepository
	workflowRepo    *repository.WorkflowRepository
	hub             *websocket.Hub
	activityService *ActivityService
	notifService    *NotificationService
}

func NewTaskService(
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	workflowRepo *repository.WorkflowRepository,
	hub *websocket.Hub,
	activityService *ActivityService,
	notifService *NotificationService,
) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		workflowRepo:    workflowRepo,
		hub:             hub,
		activityService: activityService,
		notifService:    notifService,
	}
}

type CreateTaskInput struct {
	ParentTaskID *uuid.UUID `json:"parent_task_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	AssignedTo   *uuid.UUID `json:"assigned_to"`
	StartDate    *time.Time `json:"start_date"`
	DueDate      *time.Time `json:"due_date"`
}

func (s *TaskService) CreateTask(ctx context.Context, userID, projectID uuid.UUID, input CreateTaskInput) (*models.Task, error) {
	// Verify user is a member of the project
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	if input.Status == "" {
		input.Status = models.StatusTodo
	}
	if input.Priority == "" {
		input.Priority = models.PriorityMedium
	}

	if err := validation.ValidateTaskInput(input.Title, input.Status, input.Priority, input.DueDate); err != nil {
		return nil, err
	}

	task := &models.Task{
		ProjectID:    projectID,
		ParentTaskID: input.ParentTaskID,
		Title:        input.Title,
		Description:  input.Description,
		Status:       input.Status,
		Priority:     input.Priority,
		CreatedBy:    userID,
		AssignedTo:   input.AssignedTo,
		StartDate:    input.StartDate,
		DueDate:      input.DueDate,
	}

	created, err := s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	// Record activity
	if s.activityService != nil {
		action := "task.created"
		entityType := "task"
		if created.ParentTaskID != nil {
			action = "subtask.created"
			entityType = "subtask"
		}
		s.activityService.Record(ctx, projectID, userID, action, entityType, &created.ID, created.Title, nil, map[string]interface{}{
			"status":   created.Status,
			"priority": created.Priority,
		})
	}

	// Notify assignee
	if s.notifService != nil && created.AssignedTo != nil {
		s.notifService.NotifyUser(
			ctx,
			*created.AssignedTo,
			&projectID,
			&created.ID,
			&userID,
			NotifTaskAssigned,
			"New task assigned to you",
			fmt.Sprintf("You were assigned to \"%s\"", created.Title),
			nil,
		)
	}

	// Broadcast real-time events
	if s.hub != nil {
		eventName := models.EventTaskCreated
		if created.ParentTaskID != nil {
			eventName = models.EventSubtaskCreated
			// Also broadcast updated parent task stats
			if parent, err := s.taskRepo.GetByID(ctx, projectID, *created.ParentTaskID); err == nil {
				s.hub.Broadcast(&models.RealtimeEvent{
					Type:      "event",
					Event:     models.EventTaskUpdated,
					EventID:   uuid.New(),
					ProjectID: projectID,
					Timestamp: time.Now().UTC(),
					Payload:   parent,
				})
			}
		}
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     eventName,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   created,
		})
	}

	return created, nil
}

func (s *TaskService) ListSubtasks(ctx context.Context, userID, projectID, parentTaskID uuid.UUID) ([]*models.Task, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	if _, err := s.taskRepo.GetByID(ctx, projectID, parentTaskID); err != nil {
		return nil, err
	}
	return s.taskRepo.ListSubtasks(ctx, projectID, parentTaskID)
}

func (s *TaskService) GetTask(ctx context.Context, userID, projectID, taskID uuid.UUID) (*models.Task, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	return s.taskRepo.GetByID(ctx, projectID, taskID)
}

func (s *TaskService) ListTasks(ctx context.Context, userID, projectID uuid.UUID, filter repository.TaskFilter) (*models.PaginatedTasksResponse, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	return s.taskRepo.List(ctx, projectID, filter)
}

type UpdateTaskInput struct {
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	Status         *string    `json:"status"`
	Priority       *string    `json:"priority"`
	AssignedTo     *uuid.UUID `json:"assigned_to"`
	Unassign       bool       `json:"unassign"`
	StartDate      *time.Time `json:"start_date"`
	ClearStartDate bool       `json:"clear_start_date"`
	DueDate        *time.Time `json:"due_date"`
	ClearDueDate   bool       `json:"clear_due_date"`
}

func (s *TaskService) UpdateTask(ctx context.Context, userID, projectID, taskID uuid.UUID, input UpdateTaskInput) (*models.Task, error) {
	_, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	existingTask, err := s.taskRepo.GetByID(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil && *input.Title != "" {
		if err := validation.ValidateTitleOnly(*input.Title); err != nil {
			return nil, err
		}
	}
	if input.Status != nil && *input.Status != "" {
		if s.workflowRepo != nil {
			if err := s.workflowRepo.ValidateStatusAndTransition(ctx, projectID, existingTask.Status, *input.Status); err != nil {
				return nil, err
			}
		} else if err := validation.ValidateStatusOnly(*input.Status); err != nil {
			return nil, err
		}
	}
	if input.Priority != nil && *input.Priority != "" {
		if err := validation.ValidatePriorityOnly(*input.Priority); err != nil {
			return nil, err
		}
	}

	updated, err := s.taskRepo.Update(ctx, projectID, taskID, repository.UpdateTaskParams{
		Title:          input.Title,
		Description:    input.Description,
		Status:         input.Status,
		Priority:       input.Priority,
		AssignedTo:     input.AssignedTo,
		Unassign:       input.Unassign,
		StartDate:      input.StartDate,
		ClearStartDate: input.ClearStartDate,
		DueDate:        input.DueDate,
		ClearDueDate:   input.ClearDueDate,
	})
	if err != nil {
		return nil, err
	}

	// Record granular activity logs & notifications
	if s.activityService != nil {
		if input.Title != nil && *input.Title != existingTask.Title {
			s.activityService.Record(ctx, projectID, userID, "task.title_changed", "task", &taskID, updated.Title, existingTask.Title, updated.Title)
		}
		if input.Description != nil && *input.Description != existingTask.Description {
			s.activityService.Record(ctx, projectID, userID, "task.description_changed", "task", &taskID, updated.Title, existingTask.Description, updated.Description)
		}
		if input.Status != nil && *input.Status != existingTask.Status {
			s.activityService.Record(ctx, projectID, userID, "task.status_changed", "task", &taskID, updated.Title, existingTask.Status, updated.Status)
		}
		if input.Priority != nil && *input.Priority != existingTask.Priority {
			s.activityService.Record(ctx, projectID, userID, "task.priority_changed", "task", &taskID, updated.Title, existingTask.Priority, updated.Priority)
		}
		if input.Unassign || (input.AssignedTo != nil && (existingTask.AssignedTo == nil || *input.AssignedTo != *existingTask.AssignedTo)) {
			s.activityService.Record(ctx, projectID, userID, "task.assignee_changed", "task", &taskID, updated.Title, existingTask.AssigneeName, updated.AssigneeName)
		}
		if input.ClearDueDate || (input.DueDate != nil && (existingTask.DueDate == nil || !input.DueDate.Equal(*existingTask.DueDate))) {
			s.activityService.Record(ctx, projectID, userID, "task.due_date_changed", "task", &taskID, updated.Title, existingTask.DueDate, updated.DueDate)
		}
	}

	if s.notifService != nil {
		if input.AssignedTo != nil && (existingTask.AssignedTo == nil || *input.AssignedTo != *existingTask.AssignedTo) {
			s.notifService.NotifyUser(
				ctx,
				*input.AssignedTo,
				&projectID,
				&taskID,
				&userID,
				NotifTaskAssigned,
				"Task assigned to you",
				fmt.Sprintf("You were assigned to \"%s\"", updated.Title),
				nil,
			)
		}
		if input.Status != nil && *input.Status != existingTask.Status {
			targets := []uuid.UUID{existingTask.CreatedBy}
			if updated.AssignedTo != nil {
				targets = append(targets, *updated.AssignedTo)
			}
			seen := make(map[uuid.UUID]bool)
			for _, uid := range targets {
				if uid == userID || seen[uid] {
					continue
				}
				seen[uid] = true
				s.notifService.NotifyUser(
					ctx,
					uid,
					&projectID,
					&taskID,
					&userID,
					NotifStatusChanged,
					"Task status updated",
					fmt.Sprintf("\"%s\" moved from %s to %s", updated.Title, existingTask.Status, *input.Status),
					nil,
				)
			}
		}
	}

	// Publish granular real-time events
	if s.hub != nil {
		eventName := models.EventTaskUpdated
		if updated.ParentTaskID != nil {
			eventName = models.EventSubtaskUpdated
			if parent, err := s.taskRepo.GetByID(ctx, projectID, *updated.ParentTaskID); err == nil {
				s.hub.Broadcast(&models.RealtimeEvent{
					Type:      "event",
					Event:     models.EventTaskUpdated,
					EventID:   uuid.New(),
					ProjectID: projectID,
					Timestamp: time.Now().UTC(),
					Payload:   parent,
				})
			}
		}
		// General updated event
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     eventName,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   updated,
		})

		// Status changed event if status changed
		if input.Status != nil && *input.Status != existingTask.Status {
			s.hub.Broadcast(&models.RealtimeEvent{
				Type:      "event",
				Event:     models.EventTaskStatusChanged,
				EventID:   uuid.New(),
				ProjectID: projectID,
				Timestamp: time.Now().UTC(),
				Payload: map[string]interface{}{
					"task_id":    taskID,
					"old_status": existingTask.Status,
					"new_status": *input.Status,
					"task":       updated,
				},
			})
		}

		// Assigned event if assignment changed
		if input.Unassign || (input.AssignedTo != nil && (existingTask.AssignedTo == nil || *input.AssignedTo != *existingTask.AssignedTo)) {
			s.hub.Broadcast(&models.RealtimeEvent{
				Type:      "event",
				Event:     models.EventTaskAssigned,
				EventID:   uuid.New(),
				ProjectID: projectID,
				Timestamp: time.Now().UTC(),
				Payload: map[string]interface{}{
					"task_id":     taskID,
					"assigned_to": updated.AssignedTo,
					"task":        updated,
				},
			})
		}
	}

	return updated, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, userID, projectID, taskID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}

	task, err := s.taskRepo.GetByID(ctx, projectID, taskID)
	if err != nil {
		return err
	}

	// Only creator or project Admin can delete a task
	if role != models.RoleAdmin && task.CreatedBy != userID {
		return ErrForbidden
	}

	if err := s.taskRepo.Delete(ctx, projectID, taskID); err != nil {
		return err
	}

	if s.activityService != nil {
		s.activityService.Record(ctx, projectID, userID, "task.deleted", "task", &taskID, task.Title, nil, nil)
	}

	if s.hub != nil {
		if task.ParentTaskID != nil {
			s.hub.Broadcast(&models.RealtimeEvent{
				Type:      "event",
				Event:     models.EventSubtaskDeleted,
				EventID:   uuid.New(),
				ProjectID: projectID,
				Timestamp: time.Now().UTC(),
				Payload: map[string]interface{}{
					"task_id":        taskID,
					"parent_task_id": *task.ParentTaskID,
				},
			})
			if parent, err := s.taskRepo.GetByID(ctx, projectID, *task.ParentTaskID); err == nil {
				s.hub.Broadcast(&models.RealtimeEvent{
					Type:      "event",
					Event:     models.EventTaskUpdated,
					EventID:   uuid.New(),
					ProjectID: projectID,
					Timestamp: time.Now().UTC(),
					Payload:   parent,
				})
			}
		}
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventTaskDeleted,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"task_id": taskID,
			},
		})
	}

	return nil
}
