-- 000004_phase3_management.down.sql
ALTER TABLE tasks DROP COLUMN IF EXISTS recurring_task_id;
DROP TABLE IF EXISTS recurring_tasks;
DROP TABLE IF EXISTS task_labels;
DROP TABLE IF EXISTS project_labels;
DROP TABLE IF EXISTS project_statuses;
DROP TABLE IF EXISTS task_attachments;
