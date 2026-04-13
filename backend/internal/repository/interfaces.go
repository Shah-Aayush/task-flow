package repository

import (
	"context"

	"github.com/Shah-Aayush/task-flow-zomato-takehome/backend/internal/domain"
	"github.com/google/uuid"
)

// Pagination holds common query pagination parameters.
type Pagination struct {
	Page  int
	Limit int
}

// Offset calculates the SQL OFFSET value from page and limit.
func (p Pagination) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.Limit
}

// TaskFilters holds optional filter parameters for task list queries.
type TaskFilters struct {
	Status     *domain.TaskStatus
	AssigneeID *uuid.UUID
}

// UserRepository defines the persistence interface for User entities.
// The Postgres implementation in postgres/user_repo.go satisfies this interface.
// Swap to any DB without touching the service layer.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// ProjectRepository defines the persistence interface for Project entities.
type ProjectRepository interface {
	// ListByUser returns projects where the user is owner OR has tasks assigned to them.
	ListByUser(ctx context.Context, userID uuid.UUID, p Pagination) ([]domain.Project, int, error)
	Create(ctx context.Context, project *domain.Project) error
	// Exists returns true when a project with the given ID exists.
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	// HasAccess returns true when user can access the project (owner or assignee in project tasks).
	HasAccess(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
	// GetByID returns the project with its tasks populated.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TaskRepository defines the persistence interface for Task entities.
type TaskRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID, filters TaskFilters, p Pagination) ([]domain.Task, int, error)
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	Update(ctx context.Context, id uuid.UUID, fields domain.UpdateTaskFields) (*domain.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetStats(ctx context.Context, projectID uuid.UUID) (*domain.TaskStats, error)
}
