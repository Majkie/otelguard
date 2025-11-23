package postgres

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// ListOptions contains options for listing resources
type ListOptions struct {
	Limit  int
	Offset int
}

// PromptRepository handles prompt data access
type PromptRepository struct {
	db *pgxpool.Pool
}

// NewPromptRepository creates a new prompt repository
func NewPromptRepository(db *pgxpool.Pool) *PromptRepository {
	return &PromptRepository{db: db}
}

// Create creates a new prompt
func (r *PromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	query := `
		INSERT INTO prompts (id, project_id, name, description, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		prompt.ID,
		prompt.ProjectID,
		prompt.Name,
		prompt.Description,
		prompt.Tags,
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
	err := pgxscan.Get(ctx, r.db, &prompt, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prompt, nil
}

// List returns prompts for a project with pagination
func (r *PromptRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Prompt, int, error) {
	var prompts []*domain.Prompt
	var total int

	// Use default values if opts is nil
	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	// Ensure reasonable limits
	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM prompts WHERE project_id = $1 AND deleted_at IS NULL`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
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
	if err := pgxscan.Select(ctx, r.db, &prompts, listQuery, projectID, limit, offset); err != nil {
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
	_, err := r.db.Exec(ctx, query,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		prompt.Tags,
		prompt.UpdatedAt,
	)
	return err
}

// Delete soft-deletes a prompt
func (r *PromptRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE prompts SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// CreateVersion creates a new prompt version
func (r *PromptRepository) CreateVersion(ctx context.Context, version *domain.PromptVersion) error {
	query := `
		INSERT INTO prompt_versions (id, prompt_id, version, content, config, labels, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		version.ID,
		version.PromptID,
		version.Version,
		version.Content,
		version.Config,
		version.Labels,
		version.CreatedBy,
		version.CreatedAt,
	)
	return err
}

// GetLatestVersion retrieves the latest version number for a prompt
func (r *PromptRepository) GetLatestVersion(ctx context.Context, promptID string) (int, error) {
	var version int
	query := `SELECT COALESCE(MAX(version), 0) FROM prompt_versions WHERE prompt_id = $1`
	err := pgxscan.Get(ctx, r.db, &version, query, promptID)
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
	err := pgxscan.Get(ctx, r.db, &pv, query, promptID, version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &pv, nil
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
	err := pgxscan.Select(ctx, r.db, &versions, query, promptID)
	return versions, err
}

// UpdateVersionLabels updates the labels for a specific version
func (r *PromptRepository) UpdateVersionLabels(ctx context.Context, promptID string, version int, labels []string) error {
	query := `
		UPDATE prompt_versions
		SET labels = $3
		WHERE prompt_id = $1 AND version = $2
	`
	result, err := r.db.Exec(ctx, query, promptID, version, labels)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetPromptWithLatestVersion retrieves a prompt along with its latest version content
func (r *PromptRepository) GetPromptWithLatestVersion(ctx context.Context, id string) (*domain.Prompt, *domain.PromptVersion, error) {
	prompt, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	latestVersion, err := r.GetLatestVersion(ctx, id)
	if err != nil {
		return prompt, nil, nil
	}

	if latestVersion == 0 {
		return prompt, nil, nil
	}

	version, err := r.GetVersion(ctx, id, latestVersion)
	if err != nil {
		return prompt, nil, nil
	}

	return prompt, version, nil
}
