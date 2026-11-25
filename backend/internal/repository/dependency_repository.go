package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
)

var (
	ErrDependencyExists   = errors.New("this dependency already exists")
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrCrossProjectDep    = errors.New("dependencies can only link tasks within the same project")
	ErrSelfDependency     = errors.New("a task cannot depend on itself")
	ErrDependencyNotFound = errors.New("dependency not found")
)

type DependencyRepository struct {
	db *database.DB
}

func NewDependencyRepository(db *database.DB) *DependencyRepository {
	return &DependencyRepository{db: db}
}

func (r *DependencyRepository) Add(ctx context.Context, projectID, taskID, dependsOnTaskID, actorID uuid.UUID) (*models.TaskDependency, error) {
	if taskID == dependsOnTaskID {
		return nil, ErrSelfDependency
	}

	// Verify both tasks belong to the same project
	var count int
	err := r.db.Pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND id IN ($2, $3)`,
		projectID, taskID, dependsOnTaskID,
	).Scan(&count)
	if err != nil || count < 2 {
		return nil, ErrCrossProjectDep
	}

	// Cycle detection: if dependsOnTaskID already transitively depends on taskID, adding taskID -> dependsOnTaskID creates a cycle
	hasCycle, err := r.wouldCreateCycle(ctx, projectID, taskID, dependsOnTaskID)
	if err != nil {
		return nil, err
	}
	if hasCycle {
		return nil, ErrCircularDependency
	}

	now := time.Now().UTC()
	dep := &models.TaskDependency{
		ProjectID:       projectID,
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
		DependencyType:  "FINISH_TO_START",
	}

	err = r.db.Pool.QueryRow(
		ctx,
		`INSERT INTO task_dependencies (project_id, task_id, depends_on_task_id, dependency_type, created_by, created_at)
		 VALUES ($1, $2, $3, 'FINISH_TO_START', $4, $5)
		 RETURNING id, created_at`,
		projectID, taskID, dependsOnTaskID, actorID, now,
	).Scan(&dep.ID, &dep.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDependencyExists
		}
		return nil, fmt.Errorf("failed to add dependency: %w", err)
	}

	_ = r.db.Pool.QueryRow(ctx, `SELECT title, status FROM tasks WHERE id = $1`, dependsOnTaskID).Scan(&dep.DependsOnTitle, &dep.DependsOnStatus)
	_ = r.db.Pool.QueryRow(ctx, `SELECT title, status FROM tasks WHERE id = $1`, taskID).Scan(&dep.TaskTitle, &dep.TaskStatus)

	return dep, nil
}

func (r *DependencyRepository) wouldCreateCycle(ctx context.Context, projectID, taskID, dependsOnTaskID uuid.UUID) (bool, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT task_id, depends_on_task_id FROM task_dependencies WHERE project_id = $1`, projectID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	adj := make(map[uuid.UUID][]uuid.UUID)
	for rows.Next() {
		var t, d uuid.UUID
		if err := rows.Scan(&t, &d); err == nil {
			adj[t] = append(adj[t], d)
		}
	}

	// BFS starting from dependsOnTaskID following existing dependencies; if we reach taskID, it's a cycle
	visited := map[uuid.UUID]bool{dependsOnTaskID: true}
	queue := []uuid.UUID{dependsOnTaskID}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if curr == taskID {
			return true, nil
		}
		for _, next := range adj[curr] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false, nil
}

func (r *DependencyRepository) ListForTask(ctx context.Context, projectID, taskID uuid.UUID) (*models.TaskDependenciesResponse, error) {
	resp := &models.TaskDependenciesResponse{
		Prerequisites: make([]*models.TaskDependency, 0),
		Dependents:    make([]*models.TaskDependency, 0),
	}

	prereqRows, err := r.db.Pool.Query(ctx, `
		SELECT d.id, d.project_id, d.task_id, d.depends_on_task_id, d.dependency_type,
		       t.title, t.status, d.created_at
		FROM task_dependencies d
		INNER JOIN tasks t ON d.depends_on_task_id = t.id
		WHERE d.project_id = $1 AND d.task_id = $2
		ORDER BY d.created_at ASC
	`, projectID, taskID)
	if err != nil {
		return nil, err
	}
	defer prereqRows.Close()

	for prereqRows.Next() {
		d := &models.TaskDependency{}
		if err := prereqRows.Scan(&d.ID, &d.ProjectID, &d.TaskID, &d.DependsOnTaskID, &d.DependencyType, &d.DependsOnTitle, &d.DependsOnStatus, &d.CreatedAt); err == nil {
			resp.Prerequisites = append(resp.Prerequisites, d)
		}
	}

	depRows, err := r.db.Pool.Query(ctx, `
		SELECT d.id, d.project_id, d.task_id, d.depends_on_task_id, d.dependency_type,
		       t.title, t.status, d.created_at
		FROM task_dependencies d
		INNER JOIN tasks t ON d.task_id = t.id
		WHERE d.project_id = $1 AND d.depends_on_task_id = $2
		ORDER BY d.created_at ASC
	`, projectID, taskID)
	if err != nil {
		return nil, err
	}
	defer depRows.Close()

	for depRows.Next() {
		d := &models.TaskDependency{}
		if err := depRows.Scan(&d.ID, &d.ProjectID, &d.TaskID, &d.DependsOnTaskID, &d.DependencyType, &d.TaskTitle, &d.TaskStatus, &d.CreatedAt); err == nil {
			resp.Dependents = append(resp.Dependents, d)
		}
	}

	return resp, nil
}

func (r *DependencyRepository) ListForProject(ctx context.Context, projectID uuid.UUID) ([]*models.TaskDependency, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT d.id, d.project_id, d.task_id, d.depends_on_task_id, d.dependency_type,
		       t1.title, t1.status, t2.title, t2.status, d.created_at
		FROM task_dependencies d
		INNER JOIN tasks t1 ON d.task_id = t1.id
		INNER JOIN tasks t2 ON d.depends_on_task_id = t2.id
		WHERE d.project_id = $1
		ORDER BY d.created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*models.TaskDependency, 0)
	for rows.Next() {
		d := &models.TaskDependency{}
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.TaskID, &d.DependsOnTaskID, &d.DependencyType,
			&d.TaskTitle, &d.TaskStatus, &d.DependsOnTitle, &d.DependsOnStatus, &d.CreatedAt,
		); err == nil {
			items = append(items, d)
		}
	}
	return items, nil
}

func (r *DependencyRepository) Remove(ctx context.Context, projectID, taskID, dependencyID uuid.UUID) error {
	cmdTag, err := r.db.Pool.Exec(
		ctx,
		`DELETE FROM task_dependencies WHERE id = $1 AND project_id = $2 AND task_id = $3`,
		dependencyID, projectID, taskID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrDependencyNotFound
	}
	return nil
}
