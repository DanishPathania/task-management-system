-- 000002_phase1_productivity.down.sql
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS activity_logs;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS chk_tasks_not_self_parent;
DROP INDEX IF EXISTS idx_tasks_parent_task_id;
DROP INDEX IF EXISTS idx_tasks_project_created;
ALTER TABLE tasks DROP COLUMN IF EXISTS parent_task_id;
