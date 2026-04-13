package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Shah-Aayush/task-flow/backend/internal/domain"
)

// errorResponse matches the spec format for error responses.
type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// JSON writes a JSON response with the given status code and body.
// Sets Content-Type: application/json on all responses.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Error maps a domain error to an HTTP status code and writes the error response.
// This is the ONLY place in the codebase that maps errors to HTTP codes —
// handlers call Error(w, err), never set status codes manually for error cases.
func Error(w http.ResponseWriter, err error) {
	var valErr *domain.ValidationError
	if errors.As(err, &valErr) {
		JSON(w, http.StatusBadRequest, errorResponse{
			Error:  "validation failed",
			Fields: valErr.Fields,
		})
		return
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		JSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	case errors.Is(err, domain.ErrForbidden):
		JSON(w, http.StatusForbidden, errorResponse{Error: "forbidden"})
	case errors.Is(err, domain.ErrUnauthorized):
		JSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
	case errors.Is(err, domain.ErrConflict):
		JSON(w, http.StatusConflict, errorResponse{Error: "conflict"})
	default:
		JSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

// NoContent writes a 204 No Content response (used for DELETE endpoints).
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
