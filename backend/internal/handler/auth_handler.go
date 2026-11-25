package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/example/taskmgr/internal/config"
	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_request", "malformed request payload")
		return
	}

	result, err := h.authService.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			RespondError(w, http.StatusConflict, "conflict", "a user with this email address already exists")
			return
		}
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	JSON(w, http.StatusCreated, result)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_request", "malformed request payload")
		return
	}

	result, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			RespondError(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
			return
		}
		RespondError(w, http.StatusInternalServerError, "internal_error", "failed to authenticate")
		return
	}

	h.setAuthCookie(w, result.Token)
	JSON(w, http.StatusOK, result)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "task_jwt",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	if h.cfg.CookieDomain != "" {
		cookie.Domain = h.cfg.CookieDomain
	}
	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}

	user, err := h.authService.GetUserProfile(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "not_found", "user profile not found")
		return
	}

	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) setAuthCookie(w http.ResponseWriter, token string) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.CookieSameSite == "none" {
		sameSite = http.SameSiteNoneMode
	} else if h.cfg.CookieSameSite == "strict" {
		sameSite = http.SameSiteStrictMode
	}

	cookie := &http.Cookie{
		Name:     "task_jwt",
		Value:    token,
		Path:     "/",
		MaxAge:   h.cfg.JWTExpirationHours * 3600,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: sameSite,
	}
	if h.cfg.CookieDomain != "" {
		cookie.Domain = h.cfg.CookieDomain
	}
	http.SetCookie(w, cookie)
}
