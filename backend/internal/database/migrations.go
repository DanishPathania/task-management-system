package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultInitSQL = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects(created_by);

CREATE TABLE IF NOT EXISTS project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'TODO' CHECK (status IN ('TODO', 'IN_PROGRESS', 'DONE')),
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM' CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(project_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(project_id, priority);
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_to ON tasks(project_id, assigned_to);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(project_id, due_date);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_task_id ON comments(task_id, created_at ASC);
`

func (db *DB) RunMigrations(ctx context.Context, migrationsPath string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Ensure schema_migrations tracking table exists
	_, err := db.Pool.Exec(ctxTimeout, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Collect *.up.sql files sorted lexicographically
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		// Fallback to embedded default SQL if migrations directory is missing
		if _, execErr := db.Pool.Exec(ctxTimeout, defaultInitSQL); execErr != nil {
			return fmt.Errorf("failed to apply fallback migrations: %w", execErr)
		}
		log.Println("Database schema fallback migration applied successfully")
		return nil
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	for _, fileName := range upFiles {
		var alreadyApplied bool
		err := db.Pool.QueryRow(ctxTimeout, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, fileName).Scan(&alreadyApplied)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", fileName, err)
		}
		if alreadyApplied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsPath, fileName))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fileName, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			continue
		}

		tx, err := db.Pool.Begin(ctxTimeout)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(ctxTimeout, string(content)); err != nil {
			tx.Rollback(ctxTimeout)
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(ctxTimeout, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, fileName); err != nil {
			tx.Rollback(ctxTimeout)
			return fmt.Errorf("failed to record migration %s: %w", fileName, err)
		}

		if err := tx.Commit(ctxTimeout); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", fileName, err)
		}

		log.Printf("Applied database migration: %s\n", fileName)
	}

	log.Println("Database schema migrations verified and applied successfully")
	return nil
}
