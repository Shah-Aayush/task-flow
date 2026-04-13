package service

import (
	"context"
	"errors"
	"time"

	"github.com/Shah-Aayush/task-flow/backend/internal/domain"
	"github.com/Shah-Aayush/task-flow/backend/internal/repository"
	"github.com/google/uuid"
)

// ProjectServiceImpl implements ProjectService with project business rules.
type ProjectServiceImpl struct {
	projectRepo repository.ProjectRepository
}

// NewProjectService creates a new ProjectServiceImpl.
func NewProjectService(projectRepo repository.ProjectRepository) *ProjectServiceImpl {
	return &ProjectServiceImpl{projectRepo: projectRepo}
}

// List returns a paginated list of projects accessible by the user.
func (s *ProjectServiceImpl) List(ctx context.Context, userID uuid.UUID, p repository.Pagination) (*domain.ProjectListResponse, error) {
	projects, total, err := s.projectRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return nil, err
	}
	return &domain.ProjectListResponse{
		Projects: projects,
		Page:     p.Page,
		Limit:    p.Limit,
		Total:    total,
	}, nil
}

// Create creates a new project owned by the given user.
func (s *ProjectServiceImpl) Create(ctx context.Context, userID uuid.UUID, name, description string) (*domain.Project, error) {
	project := &domain.Project{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		OwnerID:     userID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

// GetByID returns a project with its tasks.
// Access is not restricted — any authenticated user can view a project they have
// access to (are owner or assignee). The list endpoint handles access filtering.
func (s *ProjectServiceImpl) GetByID(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.Project, error) {
	if err := s.ensureProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.projectRepo.GetByID(ctx, projectID)
}

// Update modifies a project's name and/or description.
// Only the project owner is allowed to update — returns ErrForbidden otherwise.
// Uses pointer args: nil means "don't update this field".
func (s *ProjectServiceImpl) Update(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, name, description *string) (*domain.Project, error) {
	proj, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Authorization check: must be owner
	if proj.OwnerID != userID {
		return nil, domain.ErrForbidden
	}

	// Apply only the provided fields
	if name != nil {
		proj.Name = *name
	}
	if description != nil {
		proj.Description = *description
	}

	if err := s.projectRepo.Update(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

// Delete removes a project and all its tasks (via DB cascade).
// Only the project owner is allowed to delete.
func (s *ProjectServiceImpl) Delete(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) error {
	proj, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return err
	}

	// Authorization check: return 403 (not 404) if the project exists but user is not owner.
	// Returning 404 for non-owners would leak existence information.
	if proj.OwnerID != userID {
		return domain.ErrForbidden
	}

	return s.projectRepo.Delete(ctx, projectID)
}

func (s *ProjectServiceImpl) ensureProjectAccess(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) error {
	exists, err := s.projectRepo.Exists(ctx, projectID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}

	hasAccess, err := s.projectRepo.HasAccess(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if !hasAccess {
		return domain.ErrForbidden
	}

	return nil
}
