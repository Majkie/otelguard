package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/otelguard/otelguard/internal/domain"
)

// UserRepository handles user data access
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.AvatarURL,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &user, err
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &user, query, email)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &user, err
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET name = $2, avatar_url = $3, password_hash = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Name,
		user.AvatarURL,
		user.PasswordHash,
		user.UpdatedAt,
	)
	return err
}

// Delete soft-deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
