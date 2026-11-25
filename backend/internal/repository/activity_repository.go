package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
)

type ActivityRepository struct {
	db *database.DB
}

func NewActivityRepository(db *database.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Create(ctx context.Context, entry *models.ActivityLog) (*models.ActivityLog, error) {
	var oldJSON, newJSON []byte
	var err error
	if entry.OldValue != nil {
		oldJSON, err = json.Marshal(entry.OldValue)
		if err != nil {
			oldJSON = nil
		}
	}
	if entry.NewValue != nil {
		newJSON, err = json.Marshal(entry.NewValue)
		if err != nil {
			newJSON = nil
		}
	}

	query := `
		INSERT INTO activity_logs (project_id, actor_id, actor_name, action, entity_type, entity_id, entity_title, old_value, new_value, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	now := time.Now().UTC()
	err = r.db.Pool.QueryRow(
		ctx,
		query,
		entry.ProjectID,
		entry.ActorID,
		entry.ActorName,
		entry.Action,
		entry.EntityType,
		entry.EntityID,
		entry.EntityTitle,
		oldJSON,
		newJSON,
		now,
	).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity log: %w", err)
	}
	return entry, nil
}

func (r *ActivityRepository) ListByProject(ctx context.Context, projectID uuid.UUID, page, limit int) (*models.PaginatedActivityResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	var totalItems int64
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM activity_logs WHERE project_id = $1`, projectID).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("failed to count activity logs: %w", err)
	}

	offset := (page - 1) * limit
	query := `
		SELECT a.id, a.project_id, a.actor_id,
		       COALESCE(NULLIF(u.name, ''), a.actor_name) as actor_name,
		       a.action, a.entity_type, a.entity_id, a.entity_title,
		       a.old_value, a.new_value, a.created_at
		FROM activity_logs a
		LEFT JOIN users u ON a.actor_id = u.id
		WHERE a.project_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity logs: %w", err)
	}
	defer rows.Close()

	activities := make([]*models.ActivityLog, 0)
	for rows.Next() {
		a := &models.ActivityLog{}
		var oldBytes, newBytes []byte
		if err := rows.Scan(
			&a.ID,
			&a.ProjectID,
			&a.ActorID,
			&a.ActorName,
			&a.Action,
			&a.EntityType,
			&a.EntityID,
			&a.EntityTitle,
			&oldBytes,
			&newBytes,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}
		if len(oldBytes) > 0 {
			var v interface{}
			if json.Unmarshal(oldBytes, &v) == nil {
				a.OldValue = v
			}
		}
		if len(newBytes) > 0 {
			var v interface{}
			if json.Unmarshal(newBytes, &v) == nil {
				a.NewValue = v
			}
		}
		activities = append(activities, a)
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.PaginatedActivityResponse{
		Activities: activities,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}
