-- Seed data for TaskFlow
-- This file is run automatically on first startup via main.go (idempotent).
-- bcrypt hash below is for password "password123" at cost=12
-- Generated with: bcrypt.GenerateFromPassword([]byte("password123"), 12)
--
-- Test credentials:
--   Email:    test@example.com
--   Password: password123

-- Seed user
INSERT INTO users (id, name, email, password, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Test User',
    'test@example.com',
    '$2a$12$XMfkrzyzbWWi82l/RGWlReSqxWGRq0RxulVOEftXuTudJTlzcuep.',
    NOW()
)
ON CONFLICT (email) DO NOTHING;

-- Seed second user (for testing assignee flows)
INSERT INTO users (id, name, email, password, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'Jane Smith',
    'jane@example.com',
    '$2a$12$XMfkrzyzbWWi82l/RGWlReSqxWGRq0RxulVOEftXuTudJTlzcuep.',
    NOW()
)
ON CONFLICT (email) DO NOTHING;

-- Seed project
INSERT INTO projects (id, name, description, owner_id, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    'TaskFlow Demo Project',
    'A sample project with tasks in various states for testing the API.',
    '00000000-0000-0000-0000-000000000001',
    NOW()
)
ON CONFLICT DO NOTHING;

-- Seed tasks (3 tasks with different statuses)
INSERT INTO tasks (id, title, description, status, priority, project_id, creator_id, assignee_id, due_date, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000020',
    'Set up project repository',
    'Initialize Git repo, configure CI pipeline, and set up branch protection rules.',
    'done',
    'high',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '2026-04-10',
    NOW(),
    NOW()
)
ON CONFLICT DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, creator_id, assignee_id, due_date, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000021',
    'Design database schema',
    'Design and document the PostgreSQL schema for users, projects, and tasks.',
    'in_progress',
    'high',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    '2026-04-20',
    NOW(),
    NOW()
)
ON CONFLICT DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, creator_id, assignee_id, due_date, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000022',
    'Implement authentication endpoints',
    'Build /auth/register and /auth/login with bcrypt and JWT.',
    'todo',
    'medium',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    '2026-04-30',
    NOW(),
    NOW()
)
ON CONFLICT DO NOTHING;
