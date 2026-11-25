-- 000002_phase1_productivity.up.sql
-- Phase 1: Subtasks, Activity Logs, Notifications & Preferences, Search Indexes

-- 1. Subtasks support on tasks table
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project_created ON tasks(project_id, created_at DESC);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_tasks_not_self_parent'
    ) THEN
        ALTER TABLE tasks ADD CONSTRAINT chk_tasks_not_self_parent CHECK (parent_task_id IS NULL OR parent_task_id <> id);
    END IF;
END $$;

-- 2. Activity Logs (Audit History)
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_name VARCHAR(255) NOT NULL DEFAULT '',
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    entity_title VARCHAR(255) NOT NULL DEFAULT '',
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_project_created ON activity_logs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_logs_entity ON activity_logs(project_id, entity_type, entity_id);

-- 3. In-App Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type VARCHAR(60) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    dedupe_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_dedupe ON notifications(user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;

-- 4. Notification Preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    task_assigned BOOLEAN NOT NULL DEFAULT TRUE,
    task_due BOOLEAN NOT NULL DEFAULT TRUE,
    mentions BOOLEAN NOT NULL DEFAULT TRUE,
    comments BOOLEAN NOT NULL DEFAULT TRUE,
    status_changes BOOLEAN NOT NULL DEFAULT TRUE,
    member_added BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
