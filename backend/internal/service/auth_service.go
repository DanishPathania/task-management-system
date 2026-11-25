package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/auth"
	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/validation"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AuthService struct {
	userRepo       *repository.UserRepository
	jwtSecret      string
	tokenDuration  time.Duration
}

type AuthResult struct {
	Token string               `json:"token"`
	User  *models.UserResponse `json:"user"`
}

type RegisterResult struct {
	Message string               `json:"message"`
	User    *models.UserResponse `json:"user"`
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, expirationHours int) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		tokenDuration: time.Duration(expirationHours) * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*RegisterResult, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if err := validation.ValidateRegisterInput(name, email, password); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to process password: %w", err)
	}

	user, err := s.userRepo.Create(ctx, name, email, hash)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		Message: "Registration successful! Please log in with your new account.",
		User:    user.ToResponse(),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user.ID, user.Email, s.jwtSecret, s.tokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &AuthResult{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}
