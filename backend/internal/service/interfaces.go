package service

import (
	"context"

	"github.com/Shah-Aayush/task-flow/backend/internal/domain"
	"github.com/Shah-Aayush/task-flow/backend/internal/repository"
	"github.com/google/uuid"
)

// AuthService defines authentication business operations.
type AuthService interface {
	Register(ctx context.Context, name, email, password string) (token string, user *domain.User, err error)
	Login(ctx context.Context, email, password string) (token string, user *domain.User, err error)
}

// ProjectService defines project business operations.
type ProjectService interface {
	List(ctx context.Context, userID uuid.UUID, p repository.Pagination) (*domain.ProjectListResponse, error)
	Create(ctx context.Context, userID uuid.UUID, name, description string) (*domain.Project, error)
	GetByID(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.Project, error)
	Update(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, name, description *string) (*domain.Project, error)
	Delete(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) error
}

// TaskService defines task business operations.
type TaskService interface {
	ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, filters repository.TaskFilters, p repository.Pagination) (*domain.TaskListResponse, error)
	Create(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, input CreateTaskInput) (*domain.Task, error)
	Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, fields domain.UpdateTaskFields) (*domain.Task, error)
	Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
	GetStats(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.TaskStats, error)
}

// CreateTaskInput carries validated fields for task creation.
type CreateTaskInput struct {
	Title       string
	Description string
	Priority    domain.TaskPriority
	AssigneeID  *uuid.UUID
	DueDate     *string // "YYYY-MM-DD" string, parsed in service
}
