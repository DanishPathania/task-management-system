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
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

type CommentRepository struct {
	db *database.DB
}

func NewCommentRepository(db *database.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, taskID, userID uuid.UUID, content string) (*models.Comment, error) {
	query := `
		INSERT INTO comments (task_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, task_id, user_id, content, created_at, updated_at
	`
	now := time.Now().UTC()
	c := &models.Comment{}

	err := r.db.Pool.QueryRow(ctx, query, taskID, userID, content, now, now).Scan(
		&c.ID,
		&c.TaskID,
		&c.UserID,
		&c.Content,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	return r.GetByID(ctx, c.ID)
}

func (r *CommentRepository) GetByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
	query := `
		SELECT c.id, c.task_id, c.user_id, c.content, c.created_at, c.updated_at, u.name, u.email
		FROM comments c
		INNER JOIN users u ON c.user_id = u.id
		WHERE c.id = $1
	`
	c := &models.Comment{}
	err := r.db.Pool.QueryRow(ctx, query, commentID).Scan(
		&c.ID,
		&c.TaskID,
		&c.UserID,
		&c.Content,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.UserName,
		&c.UserEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to query comment: %w", err)
	}
	return c, nil
}

func (r *CommentRepository) ListByTask(ctx context.Context, taskID uuid.UUID) ([]*models.Comment, error) {
	query := `
		SELECT c.id, c.task_id, c.user_id, c.content, c.created_at, c.updated_at, u.name, u.email
		FROM comments c
		INNER JOIN users u ON c.user_id = u.id
		WHERE c.task_id = $1
		ORDER BY c.created_at ASC
	`
	rows, err := r.db.Pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		err := rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.UserID,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.UserName,
			&c.UserEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *CommentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`
	cmdTag, err := r.db.Pool.Exec(ctx, query, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}
