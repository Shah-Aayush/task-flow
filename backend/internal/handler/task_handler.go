package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aayushshah/taskflow/internal/domain"
	"github.com/aayushshah/taskflow/internal/handler/middleware"
	"github.com/aayushshah/taskflow/internal/repository"
	"github.com/aayushshah/taskflow/internal/service"
	"github.com/aayushshah/taskflow/internal/validator"
	"github.com/google/uuid"
)

// TaskHandler handles HTTP requests for task endpoints.
type TaskHandler struct {
	taskService service.TaskService
	validator   *validator.Validator
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService service.TaskService, v *validator.Validator) *TaskHandler {
	return &TaskHandler{taskService: taskService, validator: v}
}

// createTaskRequest is the JSON body for POST /projects/:id/tasks.
type createTaskRequest struct {
	Title       string  `json:"title"       validate:"required,min=1,max=255"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"    validate:"omitempty,oneof=low medium high"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

// updateTaskRequest is the JSON body for PATCH /tasks/:id.
// All fields are optional (pointer types) — nil means "don't update this field".
// The `json.RawMessage` approach is used for assignee_id so we can detect
// explicit null (clear assignee) vs absent (don't change).
type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *string `json:"assignee_id"` // "null" string handled below
	DueDate     *string `json:"due_date"`
}

// rawUpdateTaskRequest is used for JSON decoding to detect explicit null values.
type rawUpdateTaskRequest struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	Status      *string          `json:"status"`
	Priority    *string          `json:"priority"`
	AssigneeID  *json.RawMessage `json:"assignee_id"`
	DueDate     *json.RawMessage `json:"due_date"`
}

// ListByProject handles GET /projects/:id/tasks with optional ?status= and ?assignee= filters.
func (h *TaskHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	// Parse optional filters from query params
	filters := repository.TaskFilters{}
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := domain.TaskStatus(statusStr)
		if !s.Valid() {
			JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status filter"})
			return
		}
		filters.Status = &s
	}
	if assigneeStr := r.URL.Query().Get("assignee"); assigneeStr != "" {
		assigneeID, err := uuid.Parse(assigneeStr)
		if err != nil {
			JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid assignee filter"})
			return
		}
		filters.AssigneeID = &assigneeID
	}

	p := parsePagination(r)
	resp, err := h.taskService.ListByProject(r.Context(), claims.UserID, projectID, filters, p)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// Create handles POST /projects/:id/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		Error(w, err)
		return
	}

	// Parse assignee_id as UUID if provided
	var assigneeID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		id, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid assignee_id format"})
			return
		}
		assigneeID = &id
	}

	input := service.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    domain.TaskPriority(req.Priority),
		AssigneeID:  assigneeID,
		DueDate:     req.DueDate,
	}

	task, err := h.taskService.Create(r.Context(), claims.UserID, projectID, input)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, task)
}

// Update handles PATCH /tasks/:id — partial update.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	taskID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	// Use raw message decoding to detect explicit null for assignee_id and due_date
	var raw rawUpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	fields := domain.UpdateTaskFields{}

	if raw.Title != nil {
		fields.Title = raw.Title
	}
	if raw.Description != nil {
		fields.Description = raw.Description
	}
	if raw.Status != nil {
		s := domain.TaskStatus(*raw.Status)
		fields.Status = &s
	}
	if raw.Priority != nil {
		p := domain.TaskPriority(*raw.Priority)
		fields.Priority = &p
	}

	// Handle assignee_id: explicit JSON null → clear, UUID string → set, absent → no change
	if raw.AssigneeID != nil {
		rawVal := string(*raw.AssigneeID)
		if rawVal == "null" {
			fields.ClearAssignee = true
		} else {
			// Strip surrounding quotes from JSON string
			stripped := rawVal
			if len(stripped) >= 2 && stripped[0] == '"' {
				stripped = stripped[1 : len(stripped)-1]
			}
			id, err := uuid.Parse(stripped)
			if err != nil {
				JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid assignee_id format"})
				return
			}
			fields.AssigneeID = &id
		}
	}

	// Handle due_date: explicit null → clear, date string → set
	if raw.DueDate != nil {
		rawVal := string(*raw.DueDate)
		if rawVal == "null" {
			fields.ClearDueDate = true
		} else {
			stripped := rawVal
			if len(stripped) >= 2 && stripped[0] == '"' {
				stripped = stripped[1 : len(stripped)-1]
			}
			t, err := time.Parse("2006-01-02", stripped)
			if err != nil {
				JSON(w, http.StatusBadRequest, map[string]string{"error": "due_date must be YYYY-MM-DD"})
				return
			}
			fields.DueDate = &t
		}
	}

	task, err := h.taskService.Update(r.Context(), claims.UserID, taskID, fields)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, task)
}

// Delete handles DELETE /tasks/:id — deletes task (project owner or task creator only).
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	taskID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	if err := h.taskService.Delete(r.Context(), claims.UserID, taskID); err != nil {
		Error(w, err)
		return
	}

	NoContent(w)
}

// GetStats handles GET /projects/:id/stats (bonus endpoint).
func (h *TaskHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	stats, err := h.taskService.GetStats(r.Context(), claims.UserID, projectID)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, stats)
}



