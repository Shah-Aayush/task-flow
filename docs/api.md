# API Reference

Import `postman/taskflow.json` into Postman. Set the `BASE_URL` environment variable to `http://localhost:8080`. The `TOKEN` variable is set automatically by the collection's test scripts after login or register.

## Auth

| Method | Endpoint | Auth | Status | Description |
|---|---|---|---|---|
| POST | `/auth/register` | No | 201 | Register new user, returns `{token, user}` |
| POST | `/auth/login` | No | 200 | Login, returns `{token, user}` |

**Register example:**
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@test.com", "password": "mypassword123"}'

# Response 201:
# {"token":"eyJ...", "user":{"id":"uuid","name":"Alice","email":"alice@test.com","created_at":"..."}}
```

## Projects

| Method | Endpoint | Auth | Status | Description |
|---|---|---|---|---|
| GET | `/projects` | Yes | 200 | List projects (owner or assignee). Paginated. |
| POST | `/projects` | Yes | 201 | Create project (owner = current user) |
| GET | `/projects/:id` | Yes | 200 | Get project + its tasks inline |
| PATCH | `/projects/:id` | Yes (Owner) | 200 | Partial update name/description |
| DELETE | `/projects/:id` | Yes (Owner) | 204 | Delete project + all tasks |
| GET | `/projects/:id/stats` | Yes | 200 | task counts by status + assignee |

**List projects (paginated):**
```bash
curl "http://localhost:8080/projects?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"

# Response 200:
# {"projects":[...],"page":1,"limit":20,"total":3}
```

## Tasks

| Method | Endpoint | Auth | Status | Description |
|---|---|---|---|---|
| GET | `/projects/:id/tasks` | Yes | 200 | List tasks. Optional `?status=` and `?assignee=` filters. Paginated. |
| POST | `/projects/:id/tasks` | Yes | 201 | Create task in project |
| PATCH | `/tasks/:id` | Yes | 200 | Partial update (all fields optional) |
| DELETE | `/tasks/:id` | Yes | 204 | Delete task (Owner or Creator) |

**Create task:**
```bash
curl -X POST http://localhost:8080/projects/PROJECT_ID/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Implement login endpoint",
    "priority": "high",
    "assignee_id": "USER_UUID",
    "due_date": "2026-05-01"
  }'
```

**Partial update (PATCH):**
```bash
# Update only status, other fields unchanged
curl -X PATCH http://localhost:8080/tasks/TASK_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "done"}'

# Unset assignee (explicit JSON null)
curl -X PATCH http://localhost:8080/tasks/TASK_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assignee_id": null}'
```

## Error Responses

All error responses follow a consistent format:

```json
// 400 Validation error
{"error": "validation failed", "fields": {"email": "is required"}}

// 401 Unauthenticated
{"error": "unauthorized"}

// 403 Forbidden (wrong owner)
{"error": "forbidden"}

// 404 Not found
{"error": "not found"}

// 409 Conflict (duplicate email)
{"error": "conflict: resource already exists"}
```
