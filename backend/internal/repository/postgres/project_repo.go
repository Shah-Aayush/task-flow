package postgres

import (
	"context"
	"errors"

	"github.com/aayushshah/taskflow/internal/domain"
	"github.com/aayushshah/taskflow/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectRepository is the Postgres adapter for the ProjectRepository port.
type ProjectRepository struct {
	db *pgxpool.Pool
}

// NewProjectRepository creates a new Postgres-backed ProjectRepository.
func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// ListByUser returns paginated projects where the given user is the owner
// OR has at least one task assigned to them in the project.
//
// The LEFT JOIN + DISTINCT approach is used over UNION because it produces
// a single query execution plan. On small datasets the difference is negligible;
// on larger tables a covering index on tasks(project_id, assignee_id) would
// make this very efficient.
func (r *ProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID, p repository.Pagination) ([]domain.Project, int, error) {
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id AND t.assignee_id = $1
		WHERE p.owner_id = $1 OR t.id IS NOT NULL
	`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	if p.Limit <= 0 {
		p.Limit = 20
	}

	listQuery := `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id AND t.assignee_id = $1
		WHERE p.owner_id = $1 OR t.id IS NOT NULL
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, listQuery, userID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var proj domain.Project
		if err := rows.Scan(
			&proj.ID, &proj.Name, &proj.Description, &proj.OwnerID, &proj.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		projects = append(projects, proj)
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	return projects, total, rows.Err()
}

// Create inserts a new project record.
func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	query := `
		INSERT INTO projects (id, name, description, owner_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		project.ID, project.Name, project.Description, project.OwnerID, project.CreatedAt,
	)
	return err
}

// GetByID returns a project with its tasks populated (for GET /projects/:id).
// Uses a single JOIN query rather than two round-trips to minimize latency.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	projQuery := `
		SELECT id, name, description, owner_id, created_at
		FROM projects WHERE id = $1
	`
	var proj domain.Project
	err := r.db.QueryRow(ctx, projQuery, id).Scan(
		&proj.ID, &proj.Name, &proj.Description, &proj.OwnerID, &proj.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Fetch tasks for this project
	taskQuery := `
		SELECT id, title, description, status, priority, project_id, creator_id,
		       assignee_id, due_date, created_at, updated_at
		FROM tasks WHERE project_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, taskQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	proj.Tasks = []domain.Task{}
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
			&task.ProjectID, &task.CreatorID, &task.AssigneeID, &task.DueDate,
			&task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		proj.Tasks = append(proj.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &proj, nil
}

// Update persists changes to a project's name and/or description.
// Only the fields present in the Project struct are updated.
func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	query := `
		UPDATE projects
		SET name = $1, description = $2
		WHERE id = $3
	`
	result, err := r.db.Exec(ctx, query, project.Name, project.Description, project.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a project by ID. The cascade constraint in the schema
// automatically deletes all tasks belonging to this project.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
