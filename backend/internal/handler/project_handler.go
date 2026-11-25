package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/redis"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	projectService *service.ProjectService
	redisClient    *redis.Client
}

func NewProjectHandler(projectService *service.ProjectService, redisClient *redis.Client) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		redisClient:    redisClient,
	}
}

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	project, err := h.projectService.CreateProject(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	projects, err := h.projectService.ListProjects(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list projects")
		return
	}

	if projects == nil {
		projects = make([]*models.Project, 0)
	}

	JSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	project, err := h.projectService.GetProject(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrProjectNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to fetch project")
		return
	}

	JSON(w, http.StatusOK, project)
}

type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	project, err := h.projectService.UpdateProject(r.Context(), userID, projectID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can update project details")
			return
		}
		if errors.Is(err, repository.ErrProjectNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	err = h.projectService.DeleteProject(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can delete a project")
			return
		}
		if errors.Is(err, repository.ErrProjectNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Members

func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	members, err := h.projectService.ListMembers(r.Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list members")
		return
	}

	JSON(w, http.StatusOK, members)
}

type AddMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	actorID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	member, err := h.projectService.AddMember(r.Context(), actorID, projectID, req.Email, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can add members")
			return
		}
		if errors.Is(err, repository.ErrAlreadyMember) {
			RespondError(w, http.StatusConflict, "conflict", "user is already a member of this project")
			return
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "user with this email address was not found")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, member)
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (h *ProjectHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	targetUserID, err := validation.ParseUUID(chi.URLParam(r, "userId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	member, err := h.projectService.UpdateMemberRole(r.Context(), actorID, projectID, targetUserID, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can change member roles")
			return
		}
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusNotFound, "not_found", "member not found in project")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusOK, member)
}

func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}
	targetUserID, err := validation.ParseUUID(chi.URLParam(r, "userId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	err = h.projectService.RemoveMember(r.Context(), actorID, projectID, targetUserID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			RespondError(w, http.StatusForbidden, "forbidden", "only project admins can remove members")
			return
		}
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusNotFound, "not_found", "member not found in project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Presence
func (h *ProjectHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	projectID, err := validation.ParseUUID(chi.URLParam(r, "projectId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	// Verify membership
	_, err = h.projectService.GetProject(r.Context(), userID, projectID)
	if err != nil {
		RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
		return
	}

	var onlineUserIDs []uuid.UUID
	if h.redisClient != nil {
		onlineUserIDs, _ = h.redisClient.GetOnlineUsers(r.Context(), projectID, 45)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"online_user_ids": onlineUserIDs,
	})
}
