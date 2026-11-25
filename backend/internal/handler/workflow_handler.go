package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type WorkflowHandler struct {
	workflowService *service.WorkflowService
}

func NewWorkflowHandler(workflowService *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{workflowService: workflowService}
}

type CreateStatusPayload struct {
	Key                string   `json:"key"`
	Name               string   `json:"name"`
	Color              string   `json:"color"`
	Position           int      `json:"position"`
	IsDone             bool     `json:"is_done"`
	AllowedTransitions []string `json:"allowed_transitions"`
}

func (h *WorkflowHandler) ListStatuses(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	items, err := h.workflowService.ListStatuses(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list statuses")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *WorkflowHandler) CreateStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	var req CreateStatusPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	st, err := h.workflowService.CreateStatus(r.Context(), userID, projectID, req.Key, req.Name, req.Color, req.Position, req.IsDone, req.AllowedTransitions)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can manage workflow statuses")
			return
		}
		if errors.Is(err, repository.ErrStatusExists) {
			RespondError(w, http.StatusConflict, "conflict", err.Error())
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	JSON(w, http.StatusCreated, st)
}

func (h *WorkflowHandler) DeleteStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	statusID, err := validation.ParseUUID(chi.URLParam(r, "statusId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid status id")
		return
	}
	if err := h.workflowService.DeleteStatus(r.Context(), userID, projectID, statusID); err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can delete workflow statuses")
			return
		}
		if errors.Is(err, repository.ErrStatusInUse) {
			RespondError(w, http.StatusConflict, "conflict", err.Error())
			return
		}
		if errors.Is(err, repository.ErrStatusNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete status")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type CreateLabelPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (h *WorkflowHandler) ListLabels(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	items, err := h.workflowService.ListLabels(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list labels")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *WorkflowHandler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	var req CreateLabelPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	l, err := h.workflowService.CreateLabel(r.Context(), userID, projectID, req.Name, req.Color)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrLabelExists) {
			RespondError(w, http.StatusConflict, "conflict", err.Error())
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	JSON(w, http.StatusCreated, l)
}

func (h *WorkflowHandler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	labelID, err := validation.ParseUUID(chi.URLParam(r, "labelId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid label id")
		return
	}
	if err := h.workflowService.DeleteLabel(r.Context(), userID, projectID, labelID); err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can delete labels")
			return
		}
		RespondError(w, http.StatusNotFound, "not_found", "label not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkflowHandler) ListTaskLabels(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.workflowService.ListTaskLabels(r.Context(), userID, projectID, taskID)
	if err != nil {
		RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *WorkflowHandler) AttachLabel(w http.ResponseWriter, r *http.Request) {
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
	labelID, err := validation.ParseUUID(chi.URLParam(r, "labelId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid label id")
		return
	}
	if err := h.workflowService.AttachLabel(r.Context(), userID, projectID, taskID, labelID); err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkflowHandler) DetachLabel(w http.ResponseWriter, r *http.Request) {
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
	labelID, err := validation.ParseUUID(chi.URLParam(r, "labelId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid label id")
		return
	}
	if err := h.workflowService.DetachLabel(r.Context(), userID, projectID, taskID, labelID); err != nil {
		RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
