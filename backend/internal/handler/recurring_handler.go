package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RecurringHandler struct {
	recService *service.RecurringService
}

func NewRecurringHandler(recService *service.RecurringService) *RecurringHandler {
	return &RecurringHandler{recService: recService}
}

type CreateRecurringPayload struct {
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Priority      string     `json:"priority"`
	AssignedTo    *uuid.UUID `json:"assigned_to"`
	Frequency     string     `json:"frequency"`
	IntervalValue int        `json:"interval_value"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
}

func (h *RecurringHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	items, err := h.recService.List(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list recurring tasks")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *RecurringHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	var req CreateRecurringPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	rt, err := h.recService.Create(r.Context(), userID, projectID, &models.RecurringTask{
		Title:         req.Title,
		Description:   req.Description,
		Priority:      req.Priority,
		AssignedTo:    req.AssignedTo,
		Frequency:     req.Frequency,
		IntervalValue: req.IntervalValue,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	JSON(w, http.StatusCreated, rt)
}

func (h *RecurringHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	recID, err := validation.ParseUUID(chi.URLParam(r, "recurringId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid recurring id")
		return
	}
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	rt, err := h.recService.Toggle(r.Context(), userID, projectID, recID, req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrRecurringNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update recurring task")
		return
	}
	JSON(w, http.StatusOK, rt)
}

func (h *RecurringHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	recID, err := validation.ParseUUID(chi.URLParam(r, "recurringId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid recurring id")
		return
	}
	if err := h.recService.Delete(r.Context(), userID, projectID, recID); err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can delete recurring tasks")
			return
		}
		RespondError(w, http.StatusNotFound, "not_found", "recurring task not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
