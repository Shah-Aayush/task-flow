package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Shah-Aayush/task-flow/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// decodeJSONStrict decodes exactly one JSON object and rejects unknown fields.
func decodeJSONStrict(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return err
	}
	return nil
}

// parseUUID extracts and parses a UUID path parameter from the chi router context.
// On failure, it writes a 400 response and returns a non-nil error so the caller
// can immediately return without writing a second response.
func parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, error) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + param + " format"})
		return uuid.UUID{}, err
	}
	return id, nil
}

// parsePagination extracts pagination parameters from the query string.
// Applies sensible defaults and enforces a maximum page size to prevent
// runaway queries from clients sending limit=99999.
func parsePagination(r *http.Request) repository.Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return repository.Pagination{Page: page, Limit: limit}
}
