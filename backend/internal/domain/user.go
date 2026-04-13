package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents the user entity in the domain layer.
// The Password field holds the bcrypt hash — it is NEVER serialized to JSON
// in any API response. All response DTOs use a separate UserResponse type.
type User struct {
	ID        uuid.UUID `json:"-"`
	Name      string    `json:"-"`
	Email     string    `json:"-"`
	Password  string    `json:"-"` // always bcrypt hash
	CreatedAt time.Time `json:"-"`
}

// UserResponse is the safe, serializable view of a User for API responses.
// The hashed password is deliberately excluded.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a User to a safe UserResponse (no password).
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
