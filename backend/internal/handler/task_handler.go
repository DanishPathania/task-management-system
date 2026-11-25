package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

type CreateTaskPayload struct {
	ParentTaskID *uuid.UUID `json:"parent_task_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	AssignedTo   *uuid.UUID `json:"assigned_to"`
	StartDate    *time.Time `json:"start_date"`
	DueDate      *time.Time `json:"due_date"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	var req CreateTaskPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), userID, projectID, service.CreateTaskInput{
		ParentTaskID: req.ParentTaskID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       req.Status,
		Priority:     req.Priority,
		AssignedTo:   req.AssignedTo,
		StartDate:    req.StartDate,
		DueDate:      req.DueDate,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrAssigneeNotMember) || errors.Is(err, repository.ErrInvalidParentTask) {
			RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) ListSubtasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	taskID, err := validation.ParseUUID(chi.URLParam(r, "taskId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid task id")
		return
	}

	subtasks, err := h.taskService.ListSubtasks(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "parent task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list subtasks")
		return
	}

	JSON(w, http.StatusOK, subtasks)
}

func (h *TaskHandler) CreateSubtask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	parentTaskID, err := validation.ParseUUID(chi.URLParam(r, "taskId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid task id")
		return
	}

	var req CreateTaskPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	subtask, err := h.taskService.CreateTask(r.Context(), userID, projectID, service.CreateTaskInput{
		ParentTaskID: &parentTaskID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       req.Status,
		Priority:     req.Priority,
		AssignedTo:   req.AssignedTo,
		DueDate:      req.DueDate,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrAssigneeNotMember) || errors.Is(err, repository.ErrInvalidParentTask) {
			RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, subtask)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	query := r.URL.Query()
	filter := repository.TaskFilter{
		Search:          query.Get("search"),
		Status:          query.Get("status"),
		Priority:        query.Get("priority"),
		Unassigned:      query.Get("unassigned") == "true",
		Overdue:         query.Get("overdue") == "true",
		IncludeSubtasks: query.Get("include_subtasks") == "true",
		SortBy:          query.Get("sort_by"),
		Page:            1,
		Limit:           50,
	}

	if p, err := strconv.Atoi(query.Get("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if l, err := strconv.Atoi(query.Get("limit")); err == nil && l > 0 {
		filter.Limit = l
	}
	if a := query.Get("assignee"); a != "" {
		if a == "UNASSIGNED" {
			filter.Unassigned = true
		} else if uid, err := validation.ParseUUID(a); err == nil {
			filter.AssigneeID = &uid
		}
	}
	if lbl := query.Get("label"); lbl != "" {
		if lid, err := validation.ParseUUID(lbl); err == nil {
			filter.LabelID = &lid
		}
	}
	if dbStr := query.Get("due_before"); dbStr != "" {
		if t, err := time.Parse(time.RFC3339, dbStr); err == nil {
			filter.DueBefore = &t
		} else if t, err := time.Parse("2006-01-02", dbStr); err == nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.DueBefore = &endOfDay
		}
	}
	if daStr := query.Get("due_after"); daStr != "" {
		if t, err := time.Parse(time.RFC3339, daStr); err == nil {
			filter.DueAfter = &t
		} else if t, err := time.Parse("2006-01-02", daStr); err == nil {
			filter.DueAfter = &t
		}
	}

	tasks, err := h.taskService.ListTasks(r.Context(), userID, projectID, filter)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list tasks")
		return
	}

	JSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	taskID, err := validation.ParseUUID(chi.URLParam(r, "taskId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid task id")
		return
	}

	task, err := h.taskService.GetTask(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to fetch task")
		return
	}

	JSON(w, http.StatusOK, task)
}

type UpdateTaskPayload struct {
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

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	taskID, err := validation.ParseUUID(chi.URLParam(r, "taskId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid task id")
		return
	}

	var req UpdateTaskPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	task, err := h.taskService.UpdateTask(r.Context(), userID, projectID, taskID, service.UpdateTaskInput{
		Title:          req.Title,
		Description:    req.Description,
		Status:         req.Status,
		Priority:       req.Priority,
		AssignedTo:     req.AssignedTo,
		Unassign:       req.Unassign,
		StartDate:      req.StartDate,
		ClearStartDate: req.ClearStartDate,
		DueDate:        req.DueDate,
		ClearDueDate:   req.ClearDueDate,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "task not found")
			return
		}
		if errors.Is(err, repository.ErrAssigneeNotMember) {
			RespondError(w, http.StatusBadRequest, "bad_request", "assigned user is not a member of this project")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	taskID, err := validation.ParseUUID(chi.URLParam(r, "taskId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid task id")
		return
	}

	err = h.taskService.DeleteTask(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins or task creators can delete this task")
			return
		}
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
