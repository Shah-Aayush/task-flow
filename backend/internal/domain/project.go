package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project entity in the domain layer.
// Tasks is only populated on GetByID — list queries return projects without tasks
// to avoid N+1 query problems.
type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	Tasks       []Task    `json:"tasks,omitempty"` // populated on GetByID only
}

// ProjectListResponse wraps a paginated list of projects.
type ProjectListResponse struct {
	Projects []Project `json:"projects"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	Total    int       `json:"total"`
}
