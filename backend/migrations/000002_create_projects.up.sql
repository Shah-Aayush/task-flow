CREATE TABLE IF NOT EXISTS projects (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index for fast "projects owned by user" queries
CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
