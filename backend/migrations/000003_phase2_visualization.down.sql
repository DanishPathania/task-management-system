-- 000003_phase2_visualization.down.sql
DROP TABLE IF EXISTS task_dependencies;
DROP INDEX IF EXISTS idx_tasks_start_date;
ALTER TABLE tasks DROP COLUMN IF EXISTS start_date;
