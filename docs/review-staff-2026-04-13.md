# TaskFlow Staff-Level Backend Review

Date: 2026-04-13
Reviewer mode: Staff/Principal Backend Engineer

## Scope and Method

This review covered:
- Architecture and layer boundaries (Handler -> Service -> Repository)
- Domain model and PATCH pointer/null semantics
- Postgres access patterns and migration setup
- Authentication and authorization correctness
- Operational reliability and production readiness

Verification approach:
- Deep static code review across handlers, services, repositories, middleware, migrations, and docs
- Attempted runtime execution via Docker Compose and build

## Runtime Verification Attempt (Blocked)

Executed:
- docker compose up -d --build
- DOCKER_BUILDKIT=0 docker compose up -d --build

Observed blocker:
- API build fails on unused import in backend/internal/handler/project_handler.go (github.com/google/uuid)

Impact:
- Could not run live curl/integration sequences on this exact submission without code changes.

## Critical Flaws (Blockers)

1) Build does not pass (immediate deployment blocker)
- Evidence: backend/internal/handler/project_handler.go imports github.com/google/uuid but does not use it.
- Why this matters: container cannot build, service cannot start, assignment cannot be executed end-to-end.

2) Object-level authorization gap (IDOR risk) on project detail and task surfaces
- Project detail has no access check:
  - Service path returns project directly: internal/service/project_service.go -> GetByID
  - Repository returns project and all tasks by id without user scoping: internal/repository/postgres/project_repo.go
- Task list by project has no access check:
  - internal/service/task_service.go -> ListByProject checks existence only.
- Task update has no access check:
  - internal/service/task_service.go -> Update validates enums/assignee only, then updates.
- Task stats has no access check:
  - internal/service/task_service.go -> GetStats checks existence only.
- Task create has no project membership/ownership check:
  - internal/service/task_service.go -> Create checks project existence only.

Security consequence:
- Any authenticated user with a guessed/discovered project_id or task_id can read/modify resources outside their allowed scope.
- This is a production-grade security issue and disqualifying for a mid-senior backend evaluation.

## High Priority Issues

1) Assignment contract mismatch for HTTP conflict payload
- Error mapping returns conflict: resource already exists rather than conflict.
- This is minor functionally but deviates from strict API contract in assignment examples.

2) Unsafe default secret in compose fallback
- docker-compose.yml includes JWT_SECRET default change-me-to-a-strong-random-secret-in-production.
- Operational risk if deployers forget to override.

## Medium Priority Issues

1) Unknown JSON fields are silently accepted
- Decoders do not call DisallowUnknownFields.
- Can hide client bugs and accidental contract drift.

2) AuthZ policy is encoded in comments but not enforced consistently
- Architecture doc claims service-layer authorization rules.
- Implementation only enforces strong authz for delete paths, not read/update/create on tasks or read project detail.

3) No automated tests in repository
- No *_test.go files found.
- For this assignment level, at least targeted integration tests should exist for authz and PATCH null semantics.

## Strong Points

1) Layering and separation are mostly clean
- Handlers avoid SQL.
- Services depend on repository interfaces.
- Error-to-HTTP mapping is centralized.

2) Good PATCH model for nullable fields
- json.RawMessage with explicit clear flags supports null clearing semantics.
- Repository update composes SQL SET clauses safely and updates updated_at in DB time.

3) SQL safety is generally good
- Parameterized queries used consistently.
- No direct string interpolation of user inputs into SQL values.

4) Operational basics are solid
- Graceful shutdown present.
- Structured slog logging present.
- Migration embedding and idempotent seed strategy are reasonable.
- Pagination clamp to limit <= 100 prevents runaway list reads.

## Requested Test Cases Status

1) Clear via Null boundary
- Static assessment: implementation appears logically correct for assignee_id null and due_date null.
- Runtime execution: blocked by compile failure.

2) Unauthorized access control (cross-user attacks)
- Static assessment: fails due missing object-level authorization checks in multiple task/project paths.
- Runtime execution: blocked by compile failure.

3) Pagination bounds (?page=-1&limit=99999)
- Static assessment: page coerced to >= 1 and limit clamped to 100.
- Runtime execution: blocked by compile failure.

4) Migration idempotency
- Static assessment: m.Up with ErrNoChange handling and idempotent seed (ON CONFLICT DO NOTHING) are good.
- Runtime execution: blocked by compile failure.

## Idiomatic Go Improvements

1) Enforce authorization as first-class service policy
- Add a reusable access check helper in service layer for project-level visibility and write rights.
- Apply consistently in GetByID, ListByProject, Create task, Update task, GetStats.

2) Tighten handler JSON decoding
- Use decoder.DisallowUnknownFields for all request payloads.
- This improves API contract strictness and debuggability.

3) Strengthen token validation posture
- Validate expected signing algorithm exactly and optionally issuer/audience policy.

4) Improve test strategy
- Add integration tests for:
  - cross-tenant read/write denial
  - PATCH null clear behavior
  - pagination coercion/clamp
  - migration no-change idempotency startup path

5) Improve error contract consistency
- Align all error payload strings exactly with the API contract.

## Final Verdict

Decision: No-Hire

Reason:
- There are two disqualifying production issues for a mid-senior backend bar:
  - the service does not currently build/run from the submitted code
  - multiple object-level authorization gaps create cross-tenant data exposure and unauthorized writes

With those fixed, this could move to Hire-With-Feedback because the foundational architecture and data-layer implementation quality are otherwise promising.
