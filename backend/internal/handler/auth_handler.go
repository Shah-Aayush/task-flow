package handler

import (
	"net/http"

	"github.com/Shah-Aayush/task-flow/backend/internal/service"
	"github.com/Shah-Aayush/task-flow/backend/internal/validator"
)

// AuthHandler handles HTTP requests for authentication endpoints.
// It owns exactly one responsibility: decode request → call service → write response.
// Zero business logic lives here.
type AuthHandler struct {
	authService service.AuthService
	validator   *validator.Validator
}

// NewAuthHandler creates a new AuthHandler with injected dependencies.
func NewAuthHandler(authService service.AuthService, v *validator.Validator) *AuthHandler {
	return &AuthHandler{authService: authService, validator: v}
}

// registerRequest is the expected JSON body for POST /auth/register.
type registerRequest struct {
	Name     string `json:"name"     validate:"required,min=1,max=255"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// loginRequest is the expected JSON body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// authResponse is the response body for both register and login.
type authResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

// Register handles POST /auth/register.
// Returns 201 Created with {token, user} on success.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		Error(w, err)
		return
	}

	token, user, err := h.authService.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Login handles POST /auth/login.
// Returns 200 OK with {token, user} on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		Error(w, err)
		return
	}

	token, user, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}
