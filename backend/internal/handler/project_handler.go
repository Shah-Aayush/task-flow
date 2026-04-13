package handler

import (
	"net/http"

	"github.com/Shah-Aayush/task-flow/backend/internal/handler/middleware"
	"github.com/Shah-Aayush/task-flow/backend/internal/service"
	"github.com/Shah-Aayush/task-flow/backend/internal/validator"
)

// ProjectHandler handles HTTP requests for project endpoints.
type ProjectHandler struct {
	projectService service.ProjectService
	validator      *validator.Validator
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projectService service.ProjectService, v *validator.Validator) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, validator: v}
}

// createProjectRequest is the JSON body for POST /projects.
type createProjectRequest struct {
	Name        string `json:"name"        validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

// updateProjectRequest is the JSON body for PATCH /projects/:id.
// Pointer fields allow distinguishing "not sent" from "sent as empty".
type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// List handles GET /projects — returns paginated projects accessible by the user.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	p := parsePagination(r)
	resp, err := h.projectService.List(r.Context(), claims.UserID, p)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// Create handles POST /projects — creates a new project owned by the current user.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	var req createProjectRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		Error(w, err)
		return
	}

	project, err := h.projectService.Create(r.Context(), claims.UserID, req.Name, req.Description)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, project)
}

// GetByID handles GET /projects/:id — returns project details with its tasks.
func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	project, err := h.projectService.GetByID(r.Context(), claims.UserID, projectID)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, project)
}

// Update handles PATCH /projects/:id — partial update (owner only).
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	var req updateProjectRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	project, err := h.projectService.Update(r.Context(), claims.UserID, projectID, req.Name, req.Description)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, project)
}

// Delete handles DELETE /projects/:id — deletes project + tasks (owner only).
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireAuth(w, r)
	if !ok {
		return
	}

	projectID, err := parseUUID(w, r, "id")
	if err != nil {
		return
	}

	if err := h.projectService.Delete(r.Context(), claims.UserID, projectID); err != nil {
		Error(w, err)
		return
	}

	NoContent(w)
}
