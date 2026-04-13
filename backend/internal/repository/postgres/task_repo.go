package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Shah-Aayush/task-flow/backend/internal/domain"
	"github.com/Shah-Aayush/task-flow/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskRepository is the Postgres adapter for the TaskRepository port.
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository creates a new Postgres-backed TaskRepository.
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// scanTask is a helper that scans a task row into a domain.Task struct.
// Centralizing the scan avoids duplication across query methods.
func scanTask(row pgx.Row) (*domain.Task, error) {
	var task domain.Task
	err := row.Scan(
		&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
		&task.ProjectID, &task.CreatorID, &task.AssigneeID, &task.DueDate,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

// ListByProject returns paginated tasks for a project with optional status/assignee filters.
// The WHERE clause is built dynamically based on which filters are provided.
func (r *TaskRepository) ListByProject(ctx context.Context, projectID uuid.UUID, filters repository.TaskFilters, p repository.Pagination) ([]domain.Task, int, error) {
	// Build WHERE conditions dynamically
	conditions := []string{"project_id = $1"}
	args := []interface{}{projectID}
	argIdx := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*filters.Status))
		argIdx++
	}
	if filters.AssigneeID != nil {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *filters.AssigneeID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count query (without pagination)
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM tasks %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if p.Limit <= 0 {
		p.Limit = 20
	}

	// Data query with pagination
	listQuery := fmt.Sprintf(`
		SELECT id, title, description, status, priority, project_id, creator_id,
		       assignee_id, due_date, created_at, updated_at
		FROM tasks %s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())
	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
			&task.ProjectID, &task.CreatorID, &task.AssigneeID, &task.DueDate,
			&task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

// Create inserts a new task record.
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
		INSERT INTO tasks (
			id, title, description, status, priority, project_id, creator_id,
			assignee_id, due_date, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.db.Exec(ctx, query,
		task.ID, task.Title, task.Description, string(task.Status), string(task.Priority),
		task.ProjectID, task.CreatorID, task.AssigneeID, task.DueDate,
		task.CreatedAt, task.UpdatedAt,
	)
	return err
}

// GetByID retrieves a single task by its UUID.
// Returns domain.ErrNotFound if the task does not exist.
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, creator_id,
		       assignee_id, due_date, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	return scanTask(r.db.QueryRow(ctx, query, id))
}

// Update performs a partial update of task fields.
//
// Only non-nil pointer fields in UpdateTaskFields are included in the SET clause.
// This is the correct implementation of PATCH semantics — distinguishing "field not
// sent" (nil pointer) from "field sent as empty/zero" (non-nil pointer to zero value).
//
// updated_at is always set to NOW() via SQL, not from Go's time.Now(), ensuring
// the timestamp reflects the actual DB commit time and avoids clock skew issues.
func (r *TaskRepository) Update(ctx context.Context, id uuid.UUID, fields domain.UpdateTaskFields) (*domain.Task, error) {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if fields.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *fields.Title)
		argIdx++
	}
	if fields.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *fields.Description)
		argIdx++
	}
	if fields.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*fields.Status))
		argIdx++
	}
	if fields.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*fields.Priority))
		argIdx++
	}
	if fields.ClearAssignee {
		setClauses = append(setClauses, "assignee_id = NULL")
	} else if fields.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *fields.AssigneeID)
		argIdx++
	}
	if fields.ClearDueDate {
		setClauses = append(setClauses, "due_date = NULL")
	} else if fields.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *fields.DueDate)
		argIdx++
	}

	if len(setClauses) == 0 {
		// No fields to update — return current state
		return r.GetByID(ctx, id)
	}

	// Always update updated_at via SQL NOW() — not time.Now() from Go
	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE tasks SET %s WHERE id = $%d
		RETURNING id, title, description, status, priority, project_id, creator_id,
		          assignee_id, due_date, created_at, updated_at
	`, strings.Join(setClauses, ", "), argIdx)

	return scanTask(r.db.QueryRow(ctx, query, args...))
}

// Delete removes a task by ID.
// Returns domain.ErrNotFound if no row was deleted.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetStats returns aggregated task counts by status and by assignee for a project.
//
// Single SQL with GROUP BY — avoids multiple round-trips.
// We compute totals and breakdowns in application code from the grouped rows.
func (r *TaskRepository) GetStats(ctx context.Context, projectID uuid.UUID) (*domain.TaskStats, error) {
	// Status counts
	statusQuery := `
		SELECT status, COUNT(*) FROM tasks
		WHERE project_id = $1
		GROUP BY status
	`
	statusRows, err := r.db.Query(ctx, statusQuery, projectID)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()

	byStatus := map[string]int{
		"todo":        0,
		"in_progress": 0,
		"done":        0,
	}
	total := 0
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, err
		}
		byStatus[status] = count
		total += count
	}
	if err := statusRows.Err(); err != nil {
		return nil, err
	}

	// Assignee counts (single query with LEFT JOIN for name lookup)
	assigneeQuery := `
		SELECT t.assignee_id, u.name, COUNT(*) as count
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = $1
		GROUP BY t.assignee_id, u.name
		ORDER BY count DESC
	`
	assigneeRows, err := r.db.Query(ctx, assigneeQuery, projectID)
	if err != nil {
		return nil, err
	}
	defer assigneeRows.Close()

	var byAssignee []domain.AssigneeTaskCount
	for assigneeRows.Next() {
		var ac domain.AssigneeTaskCount
		if err := assigneeRows.Scan(&ac.AssigneeID, &ac.AssigneeName, &ac.Count); err != nil {
			return nil, err
		}
		byAssignee = append(byAssignee, ac)
	}
	if byAssignee == nil {
		byAssignee = []domain.AssigneeTaskCount{}
	}

	return &domain.TaskStats{
		TotalTasks: total,
		ByStatus:   byStatus,
		ByAssignee: byAssignee,
	}, assigneeRows.Err()
}
