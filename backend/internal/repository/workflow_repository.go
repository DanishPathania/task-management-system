package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrStatusInUse      = errors.New("cannot delete status that is currently assigned to tasks")
	ErrStatusExists     = errors.New("status key already exists in this project")
	ErrStatusNotFound   = errors.New("status not found")
	ErrLabelExists      = errors.New("label name already exists in this project")
	ErrLabelNotFound    = errors.New("label not found")
)

type WorkflowRepository struct {
	db *database.DB
}

func NewWorkflowRepository(db *database.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) EnsureDefaultStatuses(ctx context.Context, projectID uuid.UUID) error {
	var count int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM project_statuses WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		key, name, color string
		pos              int
		isDone           bool
		transitions      []string
	}{
		{"TODO", "To Do", "slate", 0, false, []string{"IN_PROGRESS", "BLOCKED", "ON_HOLD", "IN_REVIEW", "DONE"}},
		{"IN_PROGRESS", "In Progress", "blue", 1, false, []string{"TODO", "IN_REVIEW", "BLOCKED", "ON_HOLD", "DONE"}},
		{"DONE", "Done", "emerald", 2, true, []string{"TODO", "IN_PROGRESS"}},
	}

	for _, d := range defaults {
		_, _ = r.db.Pool.Exec(ctx, `
			INSERT INTO project_statuses (project_id, key, name, color, position, is_done, allowed_transitions)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (project_id, key) DO NOTHING
		`, projectID, d.key, d.name, d.color, d.pos, d.isDone, d.transitions)
	}
	return nil
}

func (r *WorkflowRepository) ListStatuses(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectStatus, error) {
	if err := r.EnsureDefaultStatuses(ctx, projectID); err != nil {
		return nil, err
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, key, name, color, position, is_done, allowed_transitions, created_at
		FROM project_statuses
		WHERE project_id = $1
		ORDER BY position ASC, created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.ProjectStatus, 0)
	for rows.Next() {
		s := &models.ProjectStatus{}
		if rows.Scan(&s.ID, &s.ProjectID, &s.Key, &s.Name, &s.Color, &s.Position, &s.IsDone, &s.AllowedTransitions, &s.CreatedAt) == nil {
			items = append(items, s)
		}
	}
	return items, nil
}

func (r *WorkflowRepository) CreateStatus(ctx context.Context, projectID uuid.UUID, key, name, color string, position int, isDone bool, transitions []string) (*models.ProjectStatus, error) {
	_ = r.EnsureDefaultStatuses(ctx, projectID)
	key = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(key), " ", "_"))
	if transitions == nil {
		transitions = []string{}
	}
	s := &models.ProjectStatus{
		ProjectID:          projectID,
		Key:                key,
		Name:               strings.TrimSpace(name),
		Color:              color,
		Position:           position,
		IsDone:             isDone,
		AllowedTransitions: transitions,
	}
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO project_statuses (project_id, key, name, color, position, is_done, allowed_transitions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, projectID, s.Key, s.Name, s.Color, s.Position, s.IsDone, s.AllowedTransitions).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrStatusExists
		}
		return nil, fmt.Errorf("failed to create status: %w", err)
	}
	return s, nil
}

func (r *WorkflowRepository) DeleteStatus(ctx context.Context, projectID, statusID uuid.UUID) error {
	var key string
	err := r.db.Pool.QueryRow(ctx, `SELECT key FROM project_statuses WHERE id = $1 AND project_id = $2`, statusID, projectID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStatusNotFound
		}
		return err
	}

	var inUse int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = $2`, projectID, key).Scan(&inUse); err != nil {
		return err
	}
	if inUse > 0 {
		return ErrStatusInUse
	}

	_, err = r.db.Pool.Exec(ctx, `DELETE FROM project_statuses WHERE id = $1 AND project_id = $2`, statusID, projectID)
	return err
}

func (r *WorkflowRepository) ValidateStatusAndTransition(ctx context.Context, projectID uuid.UUID, oldStatus, newStatus string) error {
	statuses, err := r.ListStatuses(ctx, projectID)
	if err != nil {
		return err
	}
	var targetFound bool
	var oldDef *models.ProjectStatus
	for _, s := range statuses {
		if s.Key == newStatus {
			targetFound = true
		}
		if s.Key == oldStatus {
			oldDef = s
		}
	}
	if !targetFound {
		return fmt.Errorf("invalid status %q for this project", newStatus)
	}
	if oldDef != nil && len(oldDef.AllowedTransitions) > 0 && oldStatus != newStatus {
		allowed := false
		for _, t := range oldDef.AllowedTransitions {
			if t == newStatus {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("transition from %s to %s is not allowed by project workflow", oldStatus, newStatus)
		}
	}
	return nil
}

// Labels

func (r *WorkflowRepository) ListLabels(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectLabel, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, name, color, created_at
		FROM project_labels
		WHERE project_id = $1
		ORDER BY name ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.ProjectLabel, 0)
	for rows.Next() {
		l := &models.ProjectLabel{}
		if rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.CreatedAt) == nil {
			items = append(items, l)
		}
	}
	return items, nil
}

func (r *WorkflowRepository) CreateLabel(ctx context.Context, projectID uuid.UUID, name, color string) (*models.ProjectLabel, error) {
	if color == "" {
		color = "blue"
	}
	l := &models.ProjectLabel{
		ProjectID: projectID,
		Name:      strings.TrimSpace(name),
		Color:     color,
		CreatedAt: time.Now().UTC(),
	}
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO project_labels (project_id, name, color, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, projectID, l.Name, l.Color, l.CreatedAt).Scan(&l.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLabelExists
		}
		return nil, err
	}
	return l, nil
}

func (r *WorkflowRepository) DeleteLabel(ctx context.Context, projectID, labelID uuid.UUID) error {
	cmdTag, err := r.db.Pool.Exec(ctx, `DELETE FROM project_labels WHERE id = $1 AND project_id = $2`, labelID, projectID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrLabelNotFound
	}
	return nil
}

func (r *WorkflowRepository) AttachLabelToTask(ctx context.Context, projectID, taskID, labelID uuid.UUID) error {
	var valid int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM tasks WHERE id = $1 AND project_id = $2) +
		       (SELECT COUNT(*) FROM project_labels WHERE id = $3 AND project_id = $2)
	`, taskID, projectID, labelID).Scan(&valid)
	if err != nil || valid < 2 {
		return ErrLabelNotFound
	}
	_, err = r.db.Pool.Exec(ctx, `INSERT INTO task_labels (task_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, taskID, labelID)
	return err
}

func (r *WorkflowRepository) DetachLabelFromTask(ctx context.Context, taskID, labelID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM task_labels WHERE task_id = $1 AND label_id = $2`, taskID, labelID)
	return err
}

func (r *WorkflowRepository) ListTaskLabels(ctx context.Context, taskID uuid.UUID) ([]*models.ProjectLabel, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT l.id, l.project_id, l.name, l.color, l.created_at
		FROM project_labels l
		INNER JOIN task_labels tl ON tl.label_id = l.id
		WHERE tl.task_id = $1
		ORDER BY l.name ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.ProjectLabel, 0)
	for rows.Next() {
		l := &models.ProjectLabel{}
		if rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.CreatedAt) == nil {
			items = append(items, l)
		}
	}
	return items, nil
}
