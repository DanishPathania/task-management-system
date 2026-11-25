package handler

import (
	"errors"
	"net/http"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) ProjectAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	stats, err := h.analyticsService.GetProjectAnalytics(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to compute project analytics")
		return
	}

	JSON(w, http.StatusOK, stats)
}

func (h *AnalyticsHandler) DashboardAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	stats, err := h.analyticsService.GetDashboardAnalytics(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to compute dashboard analytics")
		return
	}
	JSON(w, http.StatusOK, stats)
}
