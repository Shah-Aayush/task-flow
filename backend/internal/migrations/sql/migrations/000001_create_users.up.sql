-- Enable UUID generation (idempotent — safe to run on existing databases)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,   -- always bcrypt hash, never plaintext
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Unique index on email for fast lookups and uniqueness enforcement
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
