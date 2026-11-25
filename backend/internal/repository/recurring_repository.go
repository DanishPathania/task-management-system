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

var ErrRecurringNotFound = errors.New("recurring task not found")

type RecurringRepository struct {
	db *database.DB
}

func NewRecurringRepository(db *database.DB) *RecurringRepository {
	return &RecurringRepository{db: db}
}

func (r *RecurringRepository) Create(ctx context.Context, rt *models.RecurringTask) (*models.RecurringTask, error) {
	now := time.Now().UTC()
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO recurring_tasks (project_id, created_by, title, description, priority, assigned_to, frequency, interval_value, start_date, end_date, next_run_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`, rt.ProjectID, rt.CreatedBy, rt.Title, rt.Description, rt.Priority, rt.AssignedTo, rt.Frequency, rt.IntervalValue, rt.StartDate, rt.EndDate, rt.NextRunAt, rt.IsActive, now, now).Scan(&rt.ID, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create recurring task: %w", err)
	}
	return rt, nil
}

func (r *RecurringRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*models.RecurringTask, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, created_by, title, COALESCE(description, ''), priority, assigned_to, frequency, interval_value, start_date, end_date, next_run_at, last_run_at, is_active, created_at, updated_at
		FROM recurring_tasks
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.RecurringTask, 0)
	for rows.Next() {
		rt := &models.RecurringTask{}
		if rows.Scan(&rt.ID, &rt.ProjectID, &rt.CreatedBy, &rt.Title, &rt.Description, &rt.Priority, &rt.AssignedTo, &rt.Frequency, &rt.IntervalValue, &rt.StartDate, &rt.EndDate, &rt.NextRunAt, &rt.LastRunAt, &rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt) == nil {
			items = append(items, rt)
		}
	}
	return items, nil
}

func (r *RecurringRepository) ToggleActive(ctx context.Context, projectID, recurringID uuid.UUID, active bool) (*models.RecurringTask, error) {
	rt := &models.RecurringTask{}
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE recurring_tasks
		SET is_active = $3, updated_at = $4
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, created_by, title, COALESCE(description, ''), priority, assigned_to, frequency, interval_value, start_date, end_date, next_run_at, last_run_at, is_active, created_at, updated_at
	`, recurringID, projectID, active, time.Now().UTC()).Scan(
		&rt.ID, &rt.ProjectID, &rt.CreatedBy, &rt.Title, &rt.Description, &rt.Priority, &rt.AssignedTo, &rt.Frequency, &rt.IntervalValue, &rt.StartDate, &rt.EndDate, &rt.NextRunAt, &rt.LastRunAt, &rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecurringNotFound
		}
		return nil, err
	}
	return rt, nil
}

func (r *RecurringRepository) Delete(ctx context.Context, projectID, recurringID uuid.UUID) error {
	cmdTag, err := r.db.Pool.Exec(ctx, `DELETE FROM recurring_tasks WHERE id = $1 AND project_id = $2`, recurringID, projectID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrRecurringNotFound
	}
	return nil
}

// ProcessDue atomically generates tasks for due recurring series using FOR UPDATE SKIP LOCKED
// so restarts or multiple instances never create duplicate tasks.
func (r *RecurringRepository) ProcessDue(ctx context.Context) ([]*models.Task, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	rows, err := tx.Query(ctx, `
		SELECT id, project_id, created_by, title, COALESCE(description, ''), priority, assigned_to, frequency, interval_value, next_run_at, end_date
		FROM recurring_tasks
		WHERE is_active = TRUE AND next_run_at <= $1
		FOR UPDATE SKIP LOCKED
	`, now)
	if err != nil {
		return nil, err
	}

	type dueItem struct {
		rt *models.RecurringTask
	}
	var dueItems []dueItem
	for rows.Next() {
		rt := &models.RecurringTask{}
		if rows.Scan(&rt.ID, &rt.ProjectID, &rt.CreatedBy, &rt.Title, &rt.Description, &rt.Priority, &rt.AssignedTo, &rt.Frequency, &rt.IntervalValue, &rt.NextRunAt, &rt.EndDate) == nil {
			dueItems = append(dueItems, dueItem{rt: rt})
		}
	}
	rows.Close()

	var createdTasks []*models.Task
	for _, item := range dueItems {
		rt := item.rt
		if rt.EndDate != nil && now.After(*rt.EndDate) {
			_, _ = tx.Exec(ctx, `UPDATE recurring_tasks SET is_active = FALSE, updated_at = $2 WHERE id = $1`, rt.ID, now)
			continue
		}

		// Create task instance
		dueDate := rt.NextRunAt
		var taskID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO tasks (project_id, title, description, status, priority, created_by, assigned_to, due_date, recurring_task_id, created_at, updated_at)
			VALUES ($1, $2, $3, 'TODO', $4, $5, $6, $7, $8, $9, $9)
			RETURNING id
		`, rt.ProjectID, rt.Title, rt.Description, rt.Priority, rt.CreatedBy, rt.AssignedTo, dueDate, rt.ID, now).Scan(&taskID)
		if err != nil {
			continue
		}

		// Advance next_run_at
		nextRun := rt.NextRunAt
		for !nextRun.After(now) {
			switch rt.Frequency {
			case "DAILY":
				nextRun = nextRun.AddDate(0, 0, rt.IntervalValue)
			case "WEEKLY":
				nextRun = nextRun.AddDate(0, 0, 7*rt.IntervalValue)
			case "MONTHLY":
				nextRun = nextRun.AddDate(0, rt.IntervalValue, 0)
			default:
				nextRun = nextRun.AddDate(0, 0, 1)
			}
		}

		stillActive := true
		if rt.EndDate != nil && nextRun.After(*rt.EndDate) {
			stillActive = false
		}

		_, _ = tx.Exec(ctx, `
			UPDATE recurring_tasks
			SET next_run_at = $2, last_run_at = $3, is_active = $4, updated_at = $3
			WHERE id = $1
		`, rt.ID, nextRun, now, stillActive)

		createdTasks = append(createdTasks, &models.Task{
			ID:        taskID,
			ProjectID: rt.ProjectID,
			Title:     rt.Title,
			Status:    models.StatusTodo,
			Priority:  rt.Priority,
			CreatedBy: rt.CreatedBy,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return createdTasks, nil
}
