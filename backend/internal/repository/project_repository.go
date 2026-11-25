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
	ErrProjectNotFound = errors.New("project not found")
	ErrNotProjectMember = errors.New("user is not a member of this project")
	ErrAlreadyMember   = errors.New("user is already a member of this project")
)

type ProjectRepository struct {
	db *database.DB
}

func NewProjectRepository(db *database.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, name, description string, createdBy uuid.UUID) (*models.Project, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	project := &models.Project{}

	projectQuery := `
		INSERT INTO projects (name, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, created_by, created_at, updated_at
	`
	err = tx.QueryRow(ctx, projectQuery, name, description, createdBy, now, now).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedBy,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project: %w", err)
	}

	// Creator automatically becomes project Admin
	memberQuery := `
		INSERT INTO project_members (project_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, memberQuery, project.ID, createdBy, models.RoleAdmin, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator as admin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit project creation: %w", err)
	}

	project.Role = models.RoleAdmin
	return project, nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, projectID uuid.UUID) (*models.Project, error) {
	query := `
		SELECT id, name, description, created_by, created_at, updated_at
		FROM projects
		WHERE id = $1
	`
	project := &models.Project{}
	err := r.db.Pool.QueryRow(ctx, query, projectID).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedBy,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}
	return project, nil
}

func (r *ProjectRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_by, p.created_at, p.updated_at, pm.role
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1
		ORDER BY p.updated_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.CreatedBy,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.Role,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

func (r *ProjectRepository) Update(ctx context.Context, projectID uuid.UUID, name, description string) (*models.Project, error) {
	query := `
		UPDATE projects
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = $3,
		    updated_at = $4
		WHERE id = $1
		RETURNING id, name, description, created_by, created_at, updated_at
	`
	now := time.Now().UTC()
	project := &models.Project{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, name, description, now).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedBy,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update project: %w", err)
	}
	return project, nil
}

func (r *ProjectRepository) Delete(ctx context.Context, projectID uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`
	cmdTag, err := r.db.Pool.Exec(ctx, query, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// Members Management
func (r *ProjectRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	query := `SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`
	var role string
	err := r.db.Pool.QueryRow(ctx, query, projectID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotProjectMember
		}
		return "", fmt.Errorf("failed to check member role: %w", err)
	}
	return role, nil
}

func (r *ProjectRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	query := `
		SELECT pm.project_id, pm.user_id, pm.role, pm.joined_at, u.name, u.email
		FROM project_members pm
		INNER JOIN users u ON pm.user_id = u.id
		WHERE pm.project_id = $1
		ORDER BY pm.joined_at ASC
	`
	rows, err := r.db.Pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project members: %w", err)
	}
	defer rows.Close()

	var members []*models.ProjectMember
	for rows.Next() {
		m := &models.ProjectMember{}
		err := rows.Scan(
			&m.ProjectID,
			&m.UserID,
			&m.Role,
			&m.JoinedAt,
			&m.UserName,
			&m.UserEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *ProjectRepository) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) (*models.ProjectMember, error) {
	query := `
		INSERT INTO project_members (project_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)
		RETURNING project_id, user_id, role, joined_at
	`
	now := time.Now().UTC()
	m := &models.ProjectMember{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, userID, role, now).Scan(
		&m.ProjectID,
		&m.UserID,
		&m.Role,
		&m.JoinedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyMember
		}
		return nil, fmt.Errorf("failed to add member: %w", err)
	}
	return m, nil
}

func (r *ProjectRepository) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, newRole string) (*models.ProjectMember, error) {
	query := `
		UPDATE project_members
		SET role = $3
		WHERE project_id = $1 AND user_id = $2
		RETURNING project_id, user_id, role, joined_at
	`
	m := &models.ProjectMember{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, userID, newRole).Scan(
		&m.ProjectID,
		&m.UserID,
		&m.Role,
		&m.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotProjectMember
		}
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}
	return m, nil
}

func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	query := `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`
	cmdTag, err := r.db.Pool.Exec(ctx, query, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotProjectMember
	}
	return nil
}
