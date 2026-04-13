package service

import (
	"context"
	"time"

	"github.com/aayushshah/taskflow/internal/domain"
	"github.com/aayushshah/taskflow/internal/repository"
	"github.com/google/uuid"
)

// TaskServiceImpl implements TaskService with task business rules.
type TaskServiceImpl struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	userRepo    repository.UserRepository
}

// NewTaskService creates a new TaskServiceImpl.
func NewTaskService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
) *TaskServiceImpl {
	return &TaskServiceImpl{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}
}

// ListByProject returns paginated tasks for a project with optional filters.
// Verifies the project exists before returning tasks.
func (s *TaskServiceImpl) ListByProject(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	filters repository.TaskFilters,
	p repository.Pagination,
) (*domain.TaskListResponse, error) {
	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	tasks, total, err := s.taskRepo.ListByProject(ctx, projectID, filters, p)
	if err != nil {
		return nil, err
	}

	return &domain.TaskListResponse{
		Tasks: tasks,
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}, nil
}

// Create creates a new task in a project.
// Business rules:
//  1. Project must exist
//  2. If assignee_id is provided, the assignee must exist as a user
//  3. creator_id is always set to the authenticated user
//  4. Default status: todo, priority defaults come from the request
func (s *TaskServiceImpl) Create(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, input CreateTaskInput) (*domain.Task, error) {
	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	// Validate assignee exists if provided
	if input.AssigneeID != nil {
		exists, err := s.userRepo.Exists(ctx, *input.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domain.NewValidationError(map[string]string{
				"assignee_id": "user does not exist",
			})
		}
	}

	// Default priority to medium if not provided
	priority := input.Priority
	if priority == "" {
		priority = domain.PriorityMedium
	}

	now := time.Now().UTC()
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       input.Title,
		Description: input.Description,
		Status:      domain.StatusTodo, // new tasks always start as todo
		Priority:    priority,
		ProjectID:   projectID,
		CreatorID:   userID,
		AssigneeID:  input.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Parse due_date string if provided
	if input.DueDate != nil {
		t, err := time.Parse("2006-01-02", *input.DueDate)
		if err != nil {
			return nil, domain.NewValidationError(map[string]string{
				"due_date": "must be a valid date in YYYY-MM-DD format",
			})
		}
		task.DueDate = &t
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// Update performs a partial update of a task.
// Business rules:
//  1. Task must exist
//  2. If status is provided, it must be a valid enum value
//  3. If priority is provided, it must be a valid enum value
//  4. If assignee_id is provided (non-nil, non-clear), the assignee must exist
func (s *TaskServiceImpl) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, fields domain.UpdateTaskFields) (*domain.Task, error) {
	// Verify task exists
	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return nil, err
	}

	// Validate enum values
	if fields.Status != nil && !fields.Status.Valid() {
		return nil, domain.NewValidationError(map[string]string{
			"status": "must be one of: todo, in_progress, done",
		})
	}
	if fields.Priority != nil && !fields.Priority.Valid() {
		return nil, domain.NewValidationError(map[string]string{
			"priority": "must be one of: low, medium, high",
		})
	}

	// Validate assignee exists if being changed
	if !fields.ClearAssignee && fields.AssigneeID != nil {
		exists, err := s.userRepo.Exists(ctx, *fields.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domain.NewValidationError(map[string]string{
				"assignee_id": "user does not exist",
			})
		}
	}

	return s.taskRepo.Update(ctx, taskID, fields)
}

// Delete removes a task.
// Authorization: the authenticated user must be either:
//   - The project owner, OR
//   - The task creator
//
// This requires fetching both the task and the project, which is 2 DB reads.
// The alternative (adding a project_owner_id to tasks) would denormalize the schema.
// For this scale, 2 reads is acceptable and correct.
func (s *TaskServiceImpl) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Check if user is task creator
	if task.CreatorID == userID {
		return s.taskRepo.Delete(ctx, taskID)
	}

	// Check if user is project owner
	proj, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return err
	}
	if proj.OwnerID != userID {
		return domain.ErrForbidden
	}

	return s.taskRepo.Delete(ctx, taskID)
}

// GetStats returns aggregated task statistics for a project.
func (s *TaskServiceImpl) GetStats(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.TaskStats, error) {
	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.taskRepo.GetStats(ctx, projectID)
}


