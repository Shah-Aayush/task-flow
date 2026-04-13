DROP INDEX IF EXISTS idx_tasks_creator_id;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_assignee_id;
DROP INDEX IF EXISTS idx_tasks_project_id;
DROP TABLE IF EXISTS tasks;
-- Drop enum types AFTER dropping the table that references them
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_status;
