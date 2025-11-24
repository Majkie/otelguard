package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// FeedbackScoreMappingRepository handles feedback score mapping data access
type FeedbackScoreMappingRepository struct {
	db *pgxpool.Pool
}

// NewFeedbackScoreMappingRepository creates a new feedback score mapping repository
func NewFeedbackScoreMappingRepository(db *pgxpool.Pool) *FeedbackScoreMappingRepository {
	return &FeedbackScoreMappingRepository{db: db}
}

// Create creates a new feedback score mapping
func (r *FeedbackScoreMappingRepository) Create(ctx context.Context, mapping *domain.FeedbackScoreMapping) error {
	configJSON, err := json.Marshal(mapping.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO feedback_score_mappings (
			id, project_id, name, description, item_type, enabled, config, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err = r.db.Exec(ctx, query,
		mapping.ID, mapping.ProjectID, mapping.Name, mapping.Description,
		mapping.ItemType, mapping.Enabled, configJSON,
		mapping.CreatedAt, mapping.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create feedback score mapping: %w", err)
	}

	return nil
}

// GetByID retrieves a feedback score mapping by ID
func (r *FeedbackScoreMappingRepository) GetByID(ctx context.Context, id string) (*domain.FeedbackScoreMapping, error) {
	query := `
		SELECT id, project_id, name, description, item_type, enabled, config, created_at, updated_at
		FROM feedback_score_mappings
		WHERE id = $1`

	var mapping domain.FeedbackScoreMapping
	err := pgxscan.Get(ctx, r.db, &mapping, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback score mapping: %w", err)
	}

	return &mapping, nil
}

// Update updates an existing feedback score mapping
func (r *FeedbackScoreMappingRepository) Update(ctx context.Context, mapping *domain.FeedbackScoreMapping) error {
	configJSON, err := json.Marshal(mapping.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE feedback_score_mappings
		SET name = $1, description = $2, enabled = $3, config = $4, updated_at = $5
		WHERE id = $6`

	_, err = r.db.Exec(ctx, query,
		mapping.Name, mapping.Description, mapping.Enabled, configJSON, mapping.UpdatedAt, mapping.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update feedback score mapping: %w", err)
	}

	return nil
}

// Delete deletes a feedback score mapping
func (r *FeedbackScoreMappingRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM feedback_score_mappings WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feedback score mapping: %w", err)
	}

	return nil
}

// List retrieves feedback score mappings with filtering
func (r *FeedbackScoreMappingRepository) List(ctx context.Context, projectID string, itemType string, enabled *bool) ([]*domain.FeedbackScoreMapping, error) {
	whereClause := "WHERE project_id = $1"
	args := []interface{}{projectID}
	argCount := 1

	if itemType != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND item_type = $%d", argCount)
		args = append(args, itemType)
	}

	if enabled != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND enabled = $%d", argCount)
		args = append(args, *enabled)
	}

	query := fmt.Sprintf(`
		SELECT id, project_id, name, description, item_type, enabled, config, created_at, updated_at
		FROM feedback_score_mappings %s
		ORDER BY created_at DESC`, whereClause)

	var mappings []*domain.FeedbackScoreMapping
	err := pgxscan.Select(ctx, r.db, &mappings, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list feedback score mappings: %w", err)
	}

	return mappings, nil
}

// GetEnabledForItemType retrieves enabled mappings for a specific item type
func (r *FeedbackScoreMappingRepository) GetEnabledForItemType(ctx context.Context, projectID string, itemType string) ([]*domain.FeedbackScoreMapping, error) {
	enabled := true
	return r.List(ctx, projectID, itemType, &enabled)
}
