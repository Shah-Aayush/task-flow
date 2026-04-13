# TaskFlow

A production-quality task management REST API built in Go for the Zomato backend take-home assignment.

**Stack:** Go 1.23, chi, pgx/v5, golang-migrate, Docker (distroless)

## Overview

TaskFlow is a REST API for managing collaborative projects and tasks with JWT-based authentication. Users can register, log in, create projects, add tasks to those projects, and assign tasks to other users.

For assignment review convenience, Docker Compose is configured so a fresh clone can run with a single `docker compose up` command even without a `.env` file. A local fallback `JWT_SECRET` is provided for this bootstrap path; production deployments must override it with a strong, unique secret.

This is a backend-only codebase. A Postman collection is included for manual testing of all endpoints.

### Features
- **Auth:** Register, Login, JWT (24h expiry, HS256)
- **Projects:** Full CRUD operations (list, create, get with tasks inline, update, delete)
- **Tasks:** List with filters, create, partial update (PATCH), delete
- **Analytics:** Access task counts by status and assignee through the stats endpoint
- **Performance:** Pagination on all list endpoints

## Documentation

For detailed information, please refer to the specific documentation sections:

- [Setup & Execution Guide](docs/setup.md)
  Instructions for running the project locally with Docker Compose, handling manual migrations, and using test credentials.
- [API Reference](docs/api.md)
  Comprehensive list of endpoints, expected request formats, payload examples, and responses.
- [Architecture & Design Decisions](docs/architecture.md)
  In-depth explanation of the layered architecture, pointer semantics for partial updates, error handling patterns, and library choices.

## Future Improvements

Given more time, the system could be enhanced with:

**Security**
- Refresh tokens for stateless access token management and invalidation.
- Rate limiting on authentication routes to prevent brute-force attacks.
- Request ID tracing propagated through log lines and error responses.

**Features**
- Soft deletes on tasks allowing for audit history and recovery.
- Task ordering with fractional indexing for drag-and-drop interfaces.
- WebSockets or Server-Sent Events (SSE) for real-time state updates.

**Observability**
- OpenTelemetry spans for distributed tracing of complex operations.
- Prometheus metrics for latency and database connection pool monitoring.
- OpenAPI specification generated from struct annotations.
