package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type AttachmentHandler struct {
	attService *service.AttachmentService
}

func NewAttachmentHandler(attService *service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{attService: attService}
}

func (h *AttachmentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.attService.List(r.Context(), userID, projectID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list attachments")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAttachmentSize+1024*1024)
	if err := r.ParseMultipartForm(service.MaxAttachmentSize); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid multipart form or file too large (max 10MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "missing file field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	att, err := h.attService.Upload(r.Context(), userID, projectID, taskID, header.Filename, contentType, header.Size, file)
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

	JSON(w, http.StatusCreated, att)
}

func (h *AttachmentHandler) Download(w http.ResponseWriter, r *http.Request) {
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
	attID, err := validation.ParseUUID(chi.URLParam(r, "attachmentId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid attachment id")
		return
	}

	att, path, err := h.attService.GetFileForDownload(r.Context(), userID, projectID, taskID, attID)
	if err != nil {
		if errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "not a member of this project")
			return
		}
		if errors.Is(err, repository.ErrAttachmentNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "attachment not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to download attachment")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.OriginalName))
	w.Header().Set("Content-Type", att.ContentType)
	http.ServeFile(w, r, path)
}

func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	attID, err := validation.ParseUUID(chi.URLParam(r, "attachmentId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid attachment id")
		return
	}

	if err := h.attService.Delete(r.Context(), userID, projectID, taskID, attID); err != nil {
		if errors.Is(err, service.ErrForbidden) || errors.Is(err, repository.ErrNotProjectMember) {
			RespondError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
			return
		}
		if errors.Is(err, repository.ErrAttachmentNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "attachment not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
