package postgres

import (
	"context"
	"errors"

	"github.com/aayushshah/taskflow/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository is the Postgres adapter for the UserRepository port.
// All queries use parameterized placeholders ($1, $2) — no string formatting of SQL.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new Postgres-backed UserRepository.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user record. Returns domain.ErrConflict if the email is
// already registered (Postgres unique index violation code 23505).
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Name, user.Email, user.Password, user.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

// FindByEmail finds a user by their email address.
// Returns domain.ErrNotFound if no user matches.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password, created_at
		FROM users WHERE email = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by their UUID.
// Returns domain.ErrNotFound if no user matches.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password, created_at
		FROM users WHERE id = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Exists returns true if a user with the given ID exists.
// Used to validate assignee_id before accepting task assignments.
func (r *UserRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// isUniqueViolation checks if a pgx error is a Postgres unique constraint violation.
// Postgres SQLSTATE 23505 = unique_violation
func isUniqueViolation(err error) bool {
	// pgx wraps pgconn.PgError — check via error interface
	type pgError interface {
		SQLState() string
	}
	var pgErr pgError
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
