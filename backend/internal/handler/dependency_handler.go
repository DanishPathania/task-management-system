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

type DependencyHandler struct {
	depService *service.DependencyService
}

func NewDependencyHandler(depService *service.DependencyService) *DependencyHandler {
	return &DependencyHandler{depService: depService}
}

type AddDependencyPayload struct {
	DependsOnTaskID string `json:"depends_on_task_id"`
}

func (h *DependencyHandler) ListForProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	deps, err := h.depService.ListProjectDependencies(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list project dependencies")
		return
	}

	JSON(w, http.StatusOK, deps)
}

func (h *DependencyHandler) ListForTask(w http.ResponseWriter, r *http.Request) {
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

	deps, err := h.depService.ListTaskDependencies(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list task dependencies")
		return
	}

	JSON(w, http.StatusOK, deps)
}

func (h *DependencyHandler) Add(w http.ResponseWriter, r *http.Request) {
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

	var req AddDependencyPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	dependsOnID, err := validation.ParseUUID(req.DependsOnTaskID)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid depends_on_task_id")
		return
	}

	dep, err := h.depService.AddDependency(r.Context(), userID, projectID, taskID, dependsOnID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrDependencyExists) {
			RespondError(w, http.StatusConflict, "conflict", err.Error())
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, dep)
}

func (h *DependencyHandler) Remove(w http.ResponseWriter, r *http.Request) {
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
	depID, err := validation.ParseUUID(chi.URLParam(r, "dependencyId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid dependency id")
		return
	}

	if err := h.depService.RemoveDependency(r.Context(), userID, projectID, taskID, depID); err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrDependencyNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "dependency not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to remove dependency")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
