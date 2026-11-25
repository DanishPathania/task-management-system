package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

type AddCommentPayload struct {
	Content string `json:"content"`
}

func (h *CommentHandler) Add(w http.ResponseWriter, r *http.Request) {
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

	var req AddCommentPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	comment, err := h.commentService.AddComment(r.Context(), userID, projectID, taskID, req.Content)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "task not found")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, comment)
}

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	comments, err := h.commentService.ListComments(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrTaskNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list comments")
		return
	}

	if comments == nil {
		comments = make([]*models.Comment, 0)
	}

	JSON(w, http.StatusOK, comments)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	commentID, err := validation.ParseUUID(chi.URLParam(r, "commentId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid comment id")
		return
	}

	err = h.commentService.DeleteComment(r.Context(), userID, projectID, taskID, commentID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			RespondError(w, http.StatusForbidden, "forbidden", "only comment authors or project admins can delete comments")
			return
		}
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrCommentNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "comment not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
