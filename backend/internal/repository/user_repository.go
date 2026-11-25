package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, name, email, passwordHash string) (*models.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, email, password_hash, created_at, updated_at
	`
	now := time.Now().UTC()
	user := &models.User{}

	err := r.db.Pool.QueryRow(ctx, query, name, email, passwordHash, now, now).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		// Check for unique constraint violation (code 23505 in postgres)
		if isUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by ID: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`
	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}

	return user, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
