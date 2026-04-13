package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus is a typed string enum for task status values.
// Using a named type prevents raw strings from flowing through the system
// and allows compile-time type checking.
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// Valid returns true if the status is one of the known enum values.
func (s TaskStatus) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	}
	return false
}

// TaskPriority is a typed string enum for task priority values.
type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// Valid returns true if the priority is one of the known enum values.
func (p TaskPriority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// Task represents a task entity in the domain layer.
// AssigneeID and DueDate are nullable (pointer types) to distinguish
// "not set" from zero values.
//
// CreatorID is an extra field beyond the original spec — it is required
// to implement the "project owner OR task creator can delete" authorization rule.
type Task struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TaskStatus   `json:"status"`
	Priority    TaskPriority `json:"priority"`
	ProjectID   uuid.UUID    `json:"project_id"`
	CreatorID   uuid.UUID    `json:"creator_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// UpdateTaskFields carries the partial update fields for PATCH /tasks/:id.
// Pointer types are used so we can distinguish "field not sent" (nil)
// from "field sent as empty string or zero value".
// Only non-nil fields will be included in the SQL UPDATE statement.
type UpdateTaskFields struct {
	Title       *string
	Description *string
	Status      *TaskStatus
	Priority    *TaskPriority
	AssigneeID  *uuid.UUID // pass uuid.Nil to explicitly unset the assignee
	DueDate     *time.Time // pass zero time to explicitly unset the due date
	ClearAssignee bool     // true when the client explicitly sends null for assignee_id
	ClearDueDate  bool     // true when the client explicitly sends null for due_date
}

// TaskListResponse wraps a paginated list of tasks.
type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Total int    `json:"total"`
}

// TaskStats holds aggregated task statistics for a project.
type TaskStats struct {
	TotalTasks  int                  `json:"total_tasks"`
	ByStatus    map[string]int       `json:"by_status"`
	ByAssignee  []AssigneeTaskCount  `json:"by_assignee"`
}

// AssigneeTaskCount groups task count per assignee.
type AssigneeTaskCount struct {
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	AssigneeName *string    `json:"assignee_name"`
	Count        int        `json:"count"`
}
