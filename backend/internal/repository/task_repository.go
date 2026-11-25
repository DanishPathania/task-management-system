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
	ErrTaskNotFound      = errors.New("task not found")
	ErrAssigneeNotMember = errors.New("assigned user is not a member of this project")
	ErrInvalidParentTask = errors.New("invalid parent task: cannot nest subtasks or reference a task in another project")
)

type TaskRepository struct {
	db *database.DB
}

func NewTaskRepository(db *database.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) (*models.Task, error) {
	// If assignee is specified, verify they belong to the project
	if task.AssignedTo != nil {
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`
		err := r.db.Pool.QueryRow(ctx, checkQuery, task.ProjectID, *task.AssignedTo).Scan(&exists)
		if err != nil || !exists {
			return nil, ErrAssigneeNotMember
		}
	}

	// If parent task is specified, verify it belongs to the same project and is not itself a subtask
	if task.ParentTaskID != nil {
		var parentProjectID uuid.UUID
		var grandParentID *uuid.UUID
		err := r.db.Pool.QueryRow(
			ctx,
			`SELECT project_id, parent_task_id FROM tasks WHERE id = $1`,
			*task.ParentTaskID,
		).Scan(&parentProjectID, &grandParentID)
		if err != nil || parentProjectID != task.ProjectID || grandParentID != nil {
			return nil, ErrInvalidParentTask
		}
	}

	query := `
		INSERT INTO tasks (project_id, parent_task_id, title, description, status, priority, created_by, assigned_to, start_date, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, project_id, parent_task_id, title, description, status, priority, created_by, assigned_to, start_date, due_date, created_at, updated_at
	`
	now := time.Now().UTC()
	created := &models.Task{}

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		task.ProjectID,
		task.ParentTaskID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.CreatedBy,
		task.AssignedTo,
		task.StartDate,
		task.DueDate,
		now,
		now,
	).Scan(
		&created.ID,
		&created.ProjectID,
		&created.ParentTaskID,
		&created.Title,
		&created.Description,
		&created.Status,
		&created.Priority,
		&created.CreatedBy,
		&created.AssignedTo,
		&created.StartDate,
		&created.DueDate,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	return r.GetByID(ctx, created.ProjectID, created.ID)
}

func (r *TaskRepository) GetByID(ctx context.Context, projectID, taskID uuid.UUID) (*models.Task, error) {
	query := `
		SELECT t.id, t.project_id, t.parent_task_id, t.title, t.description, t.status, t.priority,
		       t.created_by, uc.name as creator_name,
		       t.assigned_to, ua.name as assignee_name,
		       t.start_date, t.due_date,
		       EXISTS(
		           SELECT 1 FROM task_dependencies td
		           INNER JOIN tasks dep ON td.depends_on_task_id = dep.id
		           WHERE td.task_id = t.id AND dep.status <> 'DONE'
		       ) as is_blocked,
		       (SELECT COUNT(*) FROM tasks st WHERE st.parent_task_id = t.id) as subtask_count,
		       (SELECT COUNT(*) FROM tasks st WHERE st.parent_task_id = t.id AND st.status = 'DONE') as completed_subtask_count,
		       t.created_at, t.updated_at
		FROM tasks t
		LEFT JOIN users uc ON t.created_by = uc.id
		LEFT JOIN users ua ON t.assigned_to = ua.id
		WHERE t.project_id = $1 AND t.id = $2
	`
	t := &models.Task{}
	var creatorName, assigneeName *string

	err := r.db.Pool.QueryRow(ctx, query, projectID, taskID).Scan(
		&t.ID,
		&t.ProjectID,
		&t.ParentTaskID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.Priority,
		&t.CreatedBy,
		&creatorName,
		&t.AssignedTo,
		&assigneeName,
		&t.StartDate,
		&t.DueDate,
		&t.IsBlocked,
		&t.SubtaskCount,
		&t.CompletedSubtaskCount,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	if creatorName != nil {
		t.CreatorName = *creatorName
	}
	if assigneeName != nil {
		t.AssigneeName = *assigneeName
	}
	if t.SubtaskCount > 0 {
		t.CompletionPercent = (t.CompletedSubtaskCount * 100) / t.SubtaskCount
	}

	return t, nil
}

type TaskFilter struct {
	Search          string
	Status          string
	Priority        string
	AssigneeID      *uuid.UUID
	LabelID         *uuid.UUID
	Unassigned      bool
	Overdue         bool
	DueBefore       *time.Time
	DueAfter        *time.Time
	ParentTaskID    *uuid.UUID
	IncludeSubtasks bool
	SortBy          string
	Page            int
	Limit           int
}

func (r *TaskRepository) List(ctx context.Context, projectID uuid.UUID, filter TaskFilter) (*models.PaginatedTasksResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}

	var whereClauses []string
	var args []interface{}
	argIndex := 1

	whereClauses = append(whereClauses, fmt.Sprintf("t.project_id = $%d", argIndex))
	args = append(args, projectID)
	argIndex++

	if filter.ParentTaskID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.parent_task_id = $%d", argIndex))
		args = append(args, *filter.ParentTaskID)
		argIndex++
	} else if !filter.IncludeSubtasks {
		whereClauses = append(whereClauses, "t.parent_task_id IS NULL")
	}

	if strings.TrimSpace(filter.Search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(t.title ILIKE $%d OR COALESCE(t.description, '') ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		argIndex++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Priority != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.priority = $%d", argIndex))
		args = append(args, filter.Priority)
		argIndex++
	}

	if filter.Unassigned {
		whereClauses = append(whereClauses, "t.assigned_to IS NULL")
	} else if filter.AssigneeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.assigned_to = $%d", argIndex))
		args = append(args, *filter.AssigneeID)
		argIndex++
	}

	if filter.LabelID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS(SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id = $%d)", argIndex))
		args = append(args, *filter.LabelID)
		argIndex++
	}

	if filter.Overdue {
		whereClauses = append(whereClauses, "t.due_date IS NOT NULL AND t.due_date < NOW() AND t.status <> 'DONE'")
	}

	if filter.DueAfter != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.due_date >= $%d", argIndex))
		args = append(args, *filter.DueAfter)
		argIndex++
	}

	if filter.DueBefore != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.due_date <= $%d", argIndex))
		args = append(args, *filter.DueBefore)
		argIndex++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	orderSQL := "t.created_at DESC"
	switch filter.SortBy {
	case "oldest":
		orderSQL = "t.created_at ASC"
	case "priority":
		orderSQL = "CASE t.priority WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END ASC, t.created_at DESC"
	case "due_date":
		orderSQL = "t.due_date ASC NULLS LAST, t.created_at DESC"
	}

	// Count total items
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks t WHERE %s", whereSQL)
	var totalItems int64
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Select items with pagination
	offset := (filter.Page - 1) * filter.Limit
	selectQuery := fmt.Sprintf(`
		SELECT t.id, t.project_id, t.parent_task_id, t.title, t.description, t.status, t.priority,
		       t.created_by, uc.name as creator_name,
		       t.assigned_to, ua.name as assignee_name,
		       t.start_date, t.due_date,
		       EXISTS(
		           SELECT 1 FROM task_dependencies td
		           INNER JOIN tasks dep ON td.depends_on_task_id = dep.id
		           WHERE td.task_id = t.id AND dep.status <> 'DONE'
		       ) as is_blocked,
		       (SELECT COUNT(*) FROM tasks st WHERE st.parent_task_id = t.id) as subtask_count,
		       (SELECT COUNT(*) FROM tasks st WHERE st.parent_task_id = t.id AND st.status = 'DONE') as completed_subtask_count,
		       t.created_at, t.updated_at
		FROM tasks t
		LEFT JOIN users uc ON t.created_by = uc.id
		LEFT JOIN users ua ON t.assigned_to = ua.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderSQL, argIndex, argIndex+1)

	args = append(args, filter.Limit, offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*models.Task, 0)
	for rows.Next() {
		t := &models.Task{}
		var creatorName, assigneeName *string
		err := rows.Scan(
			&t.ID,
			&t.ProjectID,
			&t.ParentTaskID,
			&t.Title,
			&t.Description,
			&t.Status,
			&t.Priority,
			&t.CreatedBy,
			&creatorName,
			&t.AssignedTo,
			&assigneeName,
			&t.StartDate,
			&t.DueDate,
			&t.IsBlocked,
			&t.SubtaskCount,
			&t.CompletedSubtaskCount,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		if creatorName != nil {
			t.CreatorName = *creatorName
		}
		if assigneeName != nil {
			t.AssigneeName = *assigneeName
		}
		if t.SubtaskCount > 0 {
			t.CompletionPercent = (t.CompletedSubtaskCount * 100) / t.SubtaskCount
		}
		tasks = append(tasks, t)
	}

	totalPages := int((totalItems + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.PaginatedTasksResponse{
		Tasks: tasks,
		Pagination: models.Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

func (r *TaskRepository) ListSubtasks(ctx context.Context, projectID, parentTaskID uuid.UUID) ([]*models.Task, error) {
	resp, err := r.List(ctx, projectID, TaskFilter{
		ParentTaskID: &parentTaskID,
		SortBy:       "oldest",
		Page:         1,
		Limit:        100,
	})
	if err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

type UpdateTaskParams struct {
	Title          *string
	Description    *string
	Status         *string
	Priority       *string
	AssignedTo     *uuid.UUID
	Unassign       bool
	StartDate      *time.Time
	ClearStartDate bool
	DueDate        *time.Time
	ClearDueDate   bool
}

func (r *TaskRepository) Update(ctx context.Context, projectID, taskID uuid.UUID, params UpdateTaskParams) (*models.Task, error) {
	// Verify assignee belongs to project if assigning
	if params.AssignedTo != nil {
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`
		err := r.db.Pool.QueryRow(ctx, checkQuery, projectID, *params.AssignedTo).Scan(&exists)
		if err != nil || !exists {
			return nil, ErrAssigneeNotMember
		}
	}

	query := `
		UPDATE tasks
		SET title = COALESCE($3, title),
		    description = COALESCE($4, description),
		    status = COALESCE($5, status),
		    priority = COALESCE($6, priority),
		    assigned_to = CASE WHEN $7 = TRUE THEN NULL WHEN $8::uuid IS NOT NULL THEN $8::uuid ELSE assigned_to END,
		    due_date = CASE WHEN $9 = TRUE THEN NULL WHEN $10::timestamptz IS NOT NULL THEN $10::timestamptz ELSE due_date END,
		    start_date = CASE WHEN $12 = TRUE THEN NULL WHEN $13::timestamptz IS NOT NULL THEN $13::timestamptz ELSE start_date END,
		    updated_at = $11
		WHERE project_id = $1 AND id = $2
	`
	now := time.Now().UTC()
	cmdTag, err := r.db.Pool.Exec(
		ctx,
		query,
		projectID,
		taskID,
		params.Title,
		params.Description,
		params.Status,
		params.Priority,
		params.Unassign,
		params.AssignedTo,
		params.ClearDueDate,
		params.DueDate,
		now,
		params.ClearStartDate,
		params.StartDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, ErrTaskNotFound
	}

	return r.GetByID(ctx, projectID, taskID)
}

func (r *TaskRepository) Delete(ctx context.Context, projectID, taskID uuid.UUID) error {
	query := `DELETE FROM tasks WHERE project_id = $1 AND id = $2`
	cmdTag, err := r.db.Pool.Exec(ctx, query, projectID, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}
