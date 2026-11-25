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

var ErrAttachmentNotFound = errors.New("attachment not found")

type AttachmentRepository struct {
	db *database.DB
}

func NewAttachmentRepository(db *database.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func (r *AttachmentRepository) Create(ctx context.Context, att *models.TaskAttachment) (*models.TaskAttachment, error) {
	now := time.Now().UTC()
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO task_attachments (project_id, task_id, uploaded_by, original_name, storage_key, content_type, size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, att.ProjectID, att.TaskID, att.UploadedBy, att.OriginalName, att.StorageKey, att.ContentType, att.SizeBytes, now).Scan(&att.ID, &att.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert attachment: %w", err)
	}
	return att, nil
}

func (r *AttachmentRepository) ListByTask(ctx context.Context, projectID, taskID uuid.UUID) ([]*models.TaskAttachment, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, task_id, uploaded_by, original_name, storage_key, content_type, size_bytes, created_at
		FROM task_attachments
		WHERE project_id = $1 AND task_id = $2
		ORDER BY created_at DESC
	`, projectID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.TaskAttachment, 0)
	for rows.Next() {
		a := &models.TaskAttachment{}
		if rows.Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.UploadedBy, &a.OriginalName, &a.StorageKey, &a.ContentType, &a.SizeBytes, &a.CreatedAt) == nil {
			items = append(items, a)
		}
	}
	return items, nil
}

func (r *AttachmentRepository) GetByID(ctx context.Context, projectID, taskID, attachmentID uuid.UUID) (*models.TaskAttachment, error) {
	a := &models.TaskAttachment{}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, project_id, task_id, uploaded_by, original_name, storage_key, content_type, size_bytes, created_at
		FROM task_attachments
		WHERE id = $1 AND project_id = $2 AND task_id = $3
	`, attachmentID, projectID, taskID).Scan(
		&a.ID, &a.ProjectID, &a.TaskID, &a.UploadedBy, &a.OriginalName, &a.StorageKey, &a.ContentType, &a.SizeBytes, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttachmentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (r *AttachmentRepository) Delete(ctx context.Context, projectID, taskID, attachmentID uuid.UUID) error {
	cmdTag, err := r.db.Pool.Exec(ctx, `DELETE FROM task_attachments WHERE id = $1 AND project_id = $2 AND task_id = $3`, attachmentID, projectID, taskID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
