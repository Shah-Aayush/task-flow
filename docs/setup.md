# Setup and Execution

## Running Locally

**Prerequisites:** Docker and Docker Compose.

```bash
# 1. Clone the repository
git clone https://github.com/Shah-Aayush/task-flow-zomato-takehome.git
cd taskflow-zomato-takehome

# 2. Configure environment
cp .env.example .env
# Edit .env to change DB credentials or JWT secret if needed

# 3. Start everything
docker compose up
```

The API will be available at `http://localhost:8080`.

Logs are structured JSON. For readability, you can pipe them through `jq`:
```bash
docker compose up | docker compose logs -f api | jq .
```

On startup, the API container will:
1. Wait for Postgres to pass its health check.
2. Run all pending migrations automatically.
3. Seed the database with a test user, project, and 3 tasks.
4. Start serving HTTP requests.

**Verify it is working:**
```bash
curl http://localhost:8080/health
# Expected: {"status":"ok"}
```

**To restart cleanly (idempotent migrations):**
```bash
docker compose down -v  # removes volumes
docker compose up       # fresh DB, migrations re-run cleanly
```

## Running Migrations

Migrations run **automatically on container startup**. Manual steps are not required.

The `golang-migrate` library is called from `main.go` before the HTTP server starts. Migration files are embedded in the binary, avoiding external runtime dependencies.

**To run migrations manually** (e.g., against a local Postgres instance):
```bash
# Install migrate CLI
brew install golang-migrate

# Run migrations
migrate \
  -path ./backend/migrations \
  -database "postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable" \
  up

# Roll back all migrations
migrate \
  -path ./backend/migrations \
  -database "postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable" \
  down
```

**To apply seed data manually:**
```bash
psql "postgres://taskflow:taskflow_secret@localhost:5432/taskflow" \
  -f ./backend/seeds/seed.sql
```

## Test Credentials

The database is seeded automatically on first startup.

User 1:
```text
Email:    test@example.com
Password: password123
```

User 2 (for testing assignee flows):
```text
Email:    jane@example.com
Password: password123
```

Seed data uses `ON CONFLICT DO NOTHING` and is safe to run multiple times.

**Verify password is bcrypt hashed:**
```bash
docker compose exec postgres psql -U taskflow -d taskflow \
  -c "SELECT email, password FROM users LIMIT 2;"
```
The password column will show a bcrypt hash (`$2a$12$...`) rather than plaintext.
