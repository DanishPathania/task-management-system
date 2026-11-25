-- 000004_phase3_management.up.sql
-- Phase 3: File Attachments, Custom Workflows & Labels, Recurring Tasks

-- 1. File Attachments
CREATE TABLE IF NOT EXISTS task_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    original_name VARCHAR(255) NOT NULL,
    storage_key VARCHAR(255) NOT NULL UNIQUE,
    content_type VARCHAR(150) NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_task_attachments_task ON task_attachments(task_id, created_at DESC);

-- 2. Custom Workflow Statuses (drop hardcoded status check safely)
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;

CREATE TABLE IF NOT EXISTS project_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key VARCHAR(40) NOT NULL,
    name VARCHAR(60) NOT NULL,
    color VARCHAR(30) NOT NULL DEFAULT 'slate',
    position INT NOT NULL DEFAULT 0,
    is_done BOOLEAN NOT NULL DEFAULT FALSE,
    allowed_transitions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, key)
);
CREATE INDEX IF NOT EXISTS idx_project_statuses_project ON project_statuses(project_id, position ASC);

-- 3. Task Labels
CREATE TABLE IF NOT EXISTS project_labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(30) NOT NULL DEFAULT 'blue',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS task_labels (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES project_labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);
CREATE INDEX IF NOT EXISTS idx_task_labels_label ON task_labels(label_id);

-- 4. Recurring Tasks
CREATE TABLE IF NOT EXISTS recurring_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    frequency VARCHAR(20) NOT NULL CHECK (frequency IN ('DAILY', 'WEEKLY', 'MONTHLY')),
    interval_value INT NOT NULL DEFAULT 1 CHECK (interval_value >= 1),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_run_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_recurring_tasks_due ON recurring_tasks(is_active, next_run_at) WHERE is_active = TRUE;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS recurring_task_id UUID REFERENCES recurring_tasks(id) ON DELETE SET NULL;
