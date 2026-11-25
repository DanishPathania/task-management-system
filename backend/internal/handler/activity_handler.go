package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type ActivityHandler struct {
	activityService *service.ActivityService
}

func NewActivityHandler(activityService *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	resp, err := h.activityService.ListProjectActivity(r.Context(), userID, projectID, page, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to fetch activity log")
		return
	}

	JSON(w, http.StatusOK, resp)
}
