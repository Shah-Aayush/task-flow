-- Postgres ENUM types for type-safe status and priority columns.
-- Using DB-level enums enforces valid values at the storage layer,
-- complementing application-level validation.
DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'done');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS tasks (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    title       VARCHAR(255)  NOT NULL,
    description TEXT,
    status      task_status   NOT NULL DEFAULT 'todo',
    priority    task_priority NOT NULL DEFAULT 'medium',
    project_id  UUID          NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- creator_id is NOT in the original spec but is required to implement
    -- "project owner OR task creator can delete" authorization logic.
    creator_id  UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assignee_id UUID          REFERENCES users(id) ON DELETE SET NULL,
    due_date    DATE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Performance indexes for the most common query patterns
CREATE INDEX IF NOT EXISTS idx_tasks_project_id  ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status      ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_creator_id  ON tasks(creator_id);
