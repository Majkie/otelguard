package postgres

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// GuardrailRepository handles guardrail policy data access
type GuardrailRepository struct {
	db *pgxpool.Pool
}

// NewGuardrailRepository creates a new guardrail repository
func NewGuardrailRepository(db *pgxpool.Pool) *GuardrailRepository {
	return &GuardrailRepository{db: db}
}

// Create creates a new guardrail policy
func (r *GuardrailRepository) Create(ctx context.Context, policy *domain.GuardrailPolicy) error {
	query := `
		INSERT INTO guardrail_policies (id, project_id, name, description, enabled, priority, triggers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		policy.ID,
		policy.ProjectID,
		policy.Name,
		policy.Description,
		policy.Enabled,
		policy.Priority,
		policy.Triggers,
		policy.CreatedAt,
		policy.UpdatedAt,
	)
	return err
}

// GetByID retrieves a guardrail policy by ID
func (r *GuardrailRepository) GetByID(ctx context.Context, id string) (*domain.GuardrailPolicy, error) {
	var policy domain.GuardrailPolicy
	query := `
		SELECT id, project_id, name, description, enabled, priority, triggers, created_at, updated_at
		FROM guardrail_policies
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &policy, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &policy, nil
}

// GetEnabledPolicies returns all enabled policies for a project
func (r *GuardrailRepository) GetEnabledPolicies(ctx context.Context, projectID uuid.UUID) ([]*domain.GuardrailPolicy, error) {
	var policies []*domain.GuardrailPolicy
	query := `
		SELECT id, project_id, name, description, enabled, priority, triggers, created_at, updated_at
		FROM guardrail_policies
		WHERE project_id = $1 AND enabled = true
		ORDER BY priority DESC
	`
	err := pgxscan.Select(ctx, r.db, &policies, query, projectID)
	return policies, err
}

// List returns guardrail policies for a project
func (r *GuardrailRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.GuardrailPolicy, int, error) {
	var policies []*domain.GuardrailPolicy
	var total int

	countQuery := `SELECT COUNT(*) FROM guardrail_policies WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT id, project_id, name, description, enabled, priority, triggers, created_at, updated_at
		FROM guardrail_policies
		WHERE project_id = $1
		ORDER BY priority DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &policies, listQuery, projectID, opts.Limit, opts.Offset); err != nil {
		return nil, 0, err
	}

	return policies, total, nil
}

// Update updates a guardrail policy
func (r *GuardrailRepository) Update(ctx context.Context, policy *domain.GuardrailPolicy) error {
	query := `
		UPDATE guardrail_policies
		SET name = $2, description = $3, enabled = $4, priority = $5, triggers = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Enabled,
		policy.Priority,
		policy.Triggers,
		policy.UpdatedAt,
	)
	return err
}

// Delete deletes a guardrail policy
func (r *GuardrailRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM guardrail_policies WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// AddRule adds a rule to a policy
func (r *GuardrailRepository) AddRule(ctx context.Context, rule *domain.GuardrailRule) error {
	query := `
		INSERT INTO guardrail_rules (id, policy_id, type, config, action, action_config, order_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.PolicyID,
		rule.Type,
		rule.Config,
		rule.Action,
		rule.ActionConfig,
		rule.OrderIndex,
		rule.CreatedAt,
	)
	return err
}

// GetRules returns all rules for a policy
func (r *GuardrailRepository) GetRules(ctx context.Context, policyID uuid.UUID) ([]*domain.GuardrailRule, error) {
	var rules []*domain.GuardrailRule
	query := `
		SELECT id, policy_id, type, config, action, action_config, order_index, created_at
		FROM guardrail_rules
		WHERE policy_id = $1
		ORDER BY order_index ASC
	`
	err := pgxscan.Select(ctx, r.db, &rules, query, policyID)
	return rules, err
}

// UpdateRule updates a rule
func (r *GuardrailRepository) UpdateRule(ctx context.Context, rule *domain.GuardrailRule) error {
	query := `
		UPDATE guardrail_rules
		SET type = $2, config = $3, action = $4, action_config = $5, order_index = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.Type,
		rule.Config,
		rule.Action,
		rule.ActionConfig,
		rule.OrderIndex,
	)
	return err
}

// DeleteRule deletes a rule
func (r *GuardrailRepository) DeleteRule(ctx context.Context, id string) error {
	query := `DELETE FROM guardrail_rules WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
