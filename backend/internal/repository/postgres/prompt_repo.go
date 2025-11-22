package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/otelguard/otelguard/internal/domain"
)

// ListOptions contains options for listing resources
type ListOptions struct {
	Limit  int
	Offset int
}

// PromptRepository handles prompt data access
type PromptRepository struct {
	db *sqlx.DB
}

// NewPromptRepository creates a new prompt repository
func NewPromptRepository(db *sqlx.DB) *PromptRepository {
	return &PromptRepository{db: db}
}

// Create creates a new prompt
func (r *PromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	query := `
		INSERT INTO prompts (id, project_id, name, description, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		prompt.ID,
		prompt.ProjectID,
		prompt.Name,
		prompt.Description,
		pq.Array(prompt.Tags),
		prompt.CreatedAt,
		prompt.UpdatedAt,
	)
	return err
}

// GetByID retrieves a prompt by ID
func (r *PromptRepository) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
	var prompt domain.Prompt
	query := `
		SELECT id, project_id, name, description, tags, created_at, updated_at
		FROM prompts
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &prompt, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &prompt, err
}

// List returns prompts for a project with pagination
func (r *PromptRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Prompt, int, error) {
	var prompts []*domain.Prompt
	var total int

	// Count query
	countQuery := `SELECT COUNT(*) FROM prompts WHERE project_id = $1 AND deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, project_id, name, description, tags, created_at, updated_at
		FROM prompts
		WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &prompts, listQuery, projectID, opts.Limit, opts.Offset); err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}

// Update updates a prompt
func (r *PromptRepository) Update(ctx context.Context, prompt *domain.Prompt) error {
	query := `
		UPDATE prompts
		SET name = $2, description = $3, tags = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		pq.Array(prompt.Tags),
		prompt.UpdatedAt,
	)
	return err
}

// Delete soft-deletes a prompt
func (r *PromptRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE prompts SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// CreateVersion creates a new prompt version
func (r *PromptRepository) CreateVersion(ctx context.Context, version *domain.PromptVersion) error {
	query := `
		INSERT INTO prompt_versions (id, prompt_id, version, content, config, labels, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		version.ID,
		version.PromptID,
		version.Version,
		version.Content,
		version.Config,
		pq.Array(version.Labels),
		version.CreatedBy,
		version.CreatedAt,
	)
	return err
}

// GetLatestVersion retrieves the latest version number for a prompt
func (r *PromptRepository) GetLatestVersion(ctx context.Context, promptID string) (int, error) {
	var version int
	query := `SELECT COALESCE(MAX(version), 0) FROM prompt_versions WHERE prompt_id = $1`
	err := r.db.GetContext(ctx, &version, query, promptID)
	return version, err
}

// GetVersion retrieves a specific version of a prompt
func (r *PromptRepository) GetVersion(ctx context.Context, promptID string, version int) (*domain.PromptVersion, error) {
	var pv domain.PromptVersion
	query := `
		SELECT id, prompt_id, version, content, config, labels, created_by, created_at
		FROM prompt_versions
		WHERE prompt_id = $1 AND version = $2
	`
	err := r.db.GetContext(ctx, &pv, query, promptID, version)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &pv, err
}

// ListVersions returns all versions of a prompt
func (r *PromptRepository) ListVersions(ctx context.Context, promptID string) ([]*domain.PromptVersion, error) {
	var versions []*domain.PromptVersion
	query := `
		SELECT id, prompt_id, version, content, config, labels, created_by, created_at
		FROM prompt_versions
		WHERE prompt_id = $1
		ORDER BY version DESC
	`
	err := r.db.SelectContext(ctx, &versions, query, promptID)
	return versions, err
}
