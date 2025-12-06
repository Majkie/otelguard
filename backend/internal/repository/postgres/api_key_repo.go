package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

type APIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) error {
	query := `
		INSERT INTO api_keys (id, project_id, name, key_hash, key_prefix, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		apiKey.ID,
		apiKey.ProjectID,
		apiKey.Name,
		apiKey.KeyHash,
		apiKey.KeyPrefix,
		apiKey.Scopes,
		apiKey.ExpiresAt,
		apiKey.CreatedAt,
	)
	return err
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	var key domain.APIKey
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &key, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &key, err
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var key domain.APIKey
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE key_hash = $1
	`
	err := pgxscan.Get(ctx, r.db, &key, query, keyHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &key, err
}

func (r *APIKeyRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.APIKey, error) {
	var keys []*domain.APIKey
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &keys, query, projectID)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}
