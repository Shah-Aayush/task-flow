package service

import (
	"context"
	"errors"
	"time"

	jwtpkg "github.com/Shah-Aayush/task-flow/backend/internal/auth"
	"github.com/Shah-Aayush/task-flow/backend/internal/domain"
	"github.com/Shah-Aayush/task-flow/backend/internal/repository"
	"github.com/google/uuid"
)

const jwtExpiry = 24 * time.Hour

// AuthServiceImpl contains the business logic for authentication.
// It depends only on the UserRepository interface — not on the Postgres implementation.
type AuthServiceImpl struct {
	userRepo   repository.UserRepository
	jwtSecret  string
	bcryptCost int
}

// NewAuthService creates a new AuthServiceImpl with injected dependencies.
func NewAuthService(userRepo repository.UserRepository, jwtSecret string, bcryptCost int) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:   userRepo,
		jwtSecret:  jwtSecret,
		bcryptCost: bcryptCost,
	}
}

// Register creates a new user account.
// Business rules:
//  1. Email must be unique (ErrConflict if taken)
//  2. Password is hashed with bcrypt before storage
//  3. Returns a JWT token on success (user is logged in immediately after register)
func (s *AuthServiceImpl) Register(ctx context.Context, name, email, password string) (string, *domain.User, error) {
	hash, err := jwtpkg.HashPassword(password, s.bcryptCost)
	if err != nil {
		return "", nil, err
	}

	user := &domain.User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Password:  hash,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		// ErrConflict is propagated as-is — handler maps it to 409
		return "", nil, err
	}

	token, err := jwtpkg.GenerateToken(user.ID, user.Email, s.jwtSecret, jwtExpiry)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// Login authenticates a user and returns a JWT token.
// Business rules:
//  1. User must exist (ErrNotFound → map to generic "invalid credentials" to avoid email enumeration)
//  2. Password must match the stored bcrypt hash
//  3. Returns a JWT token on success
func (s *AuthServiceImpl) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Return ErrUnauthorized, not ErrNotFound — prevents email enumeration
			return "", nil, domain.ErrUnauthorized
		}
		return "", nil, err
	}

	if err := jwtpkg.ComparePassword(user.Password, password); err != nil {
		return "", nil, domain.ErrUnauthorized
	}

	token, err := jwtpkg.GenerateToken(user.ID, user.Email, s.jwtSecret, jwtExpiry)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
