package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/validation"
	"github.com/go-chi/chi/v5"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	q := r.URL.Query()
	unreadOnly := q.Get("unread_only") == "true"
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	resp, err := h.notifService.List(r.Context(), userID, unreadOnly, page, limit)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to fetch notifications")
		return
	}
	JSON(w, http.StatusOK, resp)
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	notifID, err := validation.ParseUUID(chi.URLParam(r, "notificationId"))
	if err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid notification id")
		return
	}

	if err := h.notifService.MarkRead(r.Context(), userID, notifID); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			RespondError(w, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to mark notification read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	if err := h.notifService.MarkAllRead(r.Context(), userID); err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to mark all notifications read")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	prefs, err := h.notifService.GetPreferences(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to load notification preferences")
		return
	}
	JSON(w, http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var req models.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	req.UserID = userID
	prefs, err := h.notifService.UpdatePreferences(r.Context(), &req)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update notification preferences")
		return
	}
	JSON(w, http.StatusOK, prefs)
}
