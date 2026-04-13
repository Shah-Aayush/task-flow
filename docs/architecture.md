# Architecture Decisions

## Layered Architecture (Handler -\> Service -\> Repository)

The codebase uses a strict 3-layer architecture:

```
HTTP Request
     |
     v
 Handler        <- decode request, call service, write response
     |
     v
 Service        <- business rules, authorization, validation logic
     |
     v
 Repository     <- raw SQL, database I/O only
     |
     v
 PostgreSQL
```

No business logic leaks into handlers. No SQL leaks into services. Each layer depends on interfaces (defined in `repository/interfaces.go` and `service/interfaces.go`), not on concrete implementations. This is the Dependency Inversion principle applied directly.

## Frameworks and Libraries

### Why chi over gin

`chi` is 100% `net/http` compatible. Every handler is a plain `http.HandlerFunc`. There is no framework-specific handler signature to wrap. Adding chi is adding routing and middleware composition, not a new paradigm. `gin` is faster for some benchmark scenarios but introduces its own context type and handler signature, making handlers non-portable. For a mid-sized API like this, `chi` is the idiomatic Go choice.

### Why pgx over GORM

The specification explicitly requires raw SQL. Even without that constraint, GORM hides what SQL is actually being executed, makes debugging query performance opaque, and makes migrations magical. `pgx/v5` puts you in full control. Every query is visible, and every index decision is deliberate. `pgxpool` provides connection pooling without extra dependencies.

### slog over zap/logrus

`log/slog` is part of the Go 1.21+ standard library. For the scope of this assignment, it provides structured JSON logging with zero added dependencies. Slog is sufficient, limits external framework requirements, and provides an API surface familiar to any Go developer.

## Design Patterns

### PATCH with pointer fields

The `UpdateTaskFields` struct uses pointer types for all fields:

```go
type UpdateTaskFields struct {
    Title    *string
    Status   *TaskStatus
    // ...
}
```

This is the correct way to implement PATCH semantics. A nil pointer means "field not sent, do not update it." A non-nil pointer (even pointing to an empty string) means "field was sent, update it to this value." Without pointers, you cannot distinguish `{"title": ""}` from a request where `title` was omitted entirely.

For nullable fields (assignee_id, due_date), explicit JSON `null` is handled via `json.RawMessage`. If the raw bytes are `null`, we set a `ClearAssignee`/`ClearDueDate` boolean flag and emit `assignee_id = NULL` in the UPDATE statement.

### Sentinel errors pattern

Services return domain-layer sentinel errors (`ErrNotFound`, `ErrForbidden`, etc.). Handlers map them to HTTP status codes in a single central `Error(w, err)` function. This prevents `if err.Error() == "not found"` string matching throughout the codebase:

```go
// In service layer:
return domain.ErrForbidden

// In handler (central mapping):
case errors.Is(err, domain.ErrForbidden):
    JSON(w, 403, ...)
```

### creator_id on Task (deliberate spec extension)

The original specification does not include `creator_id` on Task. It was added because the specification also requires that the "project owner OR task creator can delete" a task. There is no way to know who created a task without storing this information. This field is documented in migrations and noted in the API reference.

### Migrations in the Go binary

Migrations run via the `golang-migrate` library called from `main.go` before the server starts. Migration SQL files are embedded into the binary via `//go:embed`.
Benefits:
1. Enables a minimal distroless Docker runtime (no shell needed).
2. Migrations are guaranteed to run before the server accepts requests.
3. Idempotent by design, handling `migrate.ErrNoChange` gracefully.
