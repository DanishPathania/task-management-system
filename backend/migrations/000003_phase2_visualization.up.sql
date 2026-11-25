-- 000003_phase2_visualization.up.sql
-- Phase 2: Task Start Date (Timeline/Gantt) & Task Dependencies

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS start_date TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_tasks_start_date ON tasks(project_id, start_date);

CREATE TABLE IF NOT EXISTS task_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(30) NOT NULL DEFAULT 'FINISH_TO_START' CHECK (dependency_type IN ('FINISH_TO_START')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (task_id, depends_on_task_id),
    CHECK (task_id <> depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_deps_task ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_task_deps_depends_on ON task_dependencies(depends_on_task_id);
CREATE INDEX IF NOT EXISTS idx_task_deps_project ON task_dependencies(project_id);
