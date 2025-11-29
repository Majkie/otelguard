package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/otelguard/otelguard/internal/domain"
)

// EvaluatorRepository handles database operations for evaluators
type EvaluatorRepository struct {
	db *pgxpool.Pool
}

// NewEvaluatorRepository creates a new EvaluatorRepository
func NewEvaluatorRepository(db *pgxpool.Pool) *EvaluatorRepository {
	return &EvaluatorRepository{db: db}
}

// Create creates a new evaluator
func (r *EvaluatorRepository) Create(ctx context.Context, evaluator *domain.Evaluator) error {
	query := `
		INSERT INTO evaluators (
			id, project_id, name, description, type, provider, model,
			template, config, output_type, min_value, max_value, categories,
			enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	// Convert pointers to sql.NullFloat64 for database
	var minValue, maxValue sql.NullFloat64
	if evaluator.MinValue != nil {
		minValue = sql.NullFloat64{Float64: *evaluator.MinValue, Valid: true}
	}
	if evaluator.MaxValue != nil {
		maxValue = sql.NullFloat64{Float64: *evaluator.MaxValue, Valid: true}
	}

	_, err := r.db.Exec(ctx, query,
		evaluator.ID,
		evaluator.ProjectID,
		evaluator.Name,
		evaluator.Description,
		evaluator.Type,
		evaluator.Provider,
		evaluator.Model,
		evaluator.Template,
		evaluator.Config,
		evaluator.OutputType,
		minValue,
		maxValue,
		evaluator.Categories,
		evaluator.Enabled,
		evaluator.CreatedAt,
		evaluator.UpdatedAt,
	)
	return err
}

// GetByID retrieves an evaluator by ID
func (r *EvaluatorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Evaluator, error) {
	query := `
		SELECT id, project_id, name, description, type, provider, model,
			template, config, output_type, min_value, max_value, categories,
			enabled, created_at, updated_at
		FROM evaluators
		WHERE id = $1 AND deleted_at IS NULL
	`

	var evaluator domain.Evaluator
	var minValue, maxValue sql.NullFloat64
	err := r.db.QueryRow(ctx, query, id).Scan(
		&evaluator.ID,
		&evaluator.ProjectID,
		&evaluator.Name,
		&evaluator.Description,
		&evaluator.Type,
		&evaluator.Provider,
		&evaluator.Model,
		&evaluator.Template,
		&evaluator.Config,
		&evaluator.OutputType,
		&minValue,
		&maxValue,
		&evaluator.Categories,
		&evaluator.Enabled,
		&evaluator.CreatedAt,
		&evaluator.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Convert nullable fields to pointers
	if minValue.Valid {
		evaluator.MinValue = &minValue.Float64
	}
	if maxValue.Valid {
		evaluator.MaxValue = &maxValue.Float64
	}

	return &evaluator, nil
}

// Update updates an evaluator
func (r *EvaluatorRepository) Update(ctx context.Context, evaluator *domain.Evaluator) error {
	query := `
		UPDATE evaluators SET
			name = $2,
			description = $3,
			provider = $4,
			model = $5,
			template = $6,
			config = $7,
			output_type = $8,
			min_value = $9,
			max_value = $10,
			categories = $11,
			enabled = $12,
			updated_at = $13
		WHERE id = $1 AND deleted_at IS NULL
	`

	// Convert pointers to sql.NullFloat64 for database
	var minValue, maxValue sql.NullFloat64
	if evaluator.MinValue != nil {
		minValue = sql.NullFloat64{Float64: *evaluator.MinValue, Valid: true}
	}
	if evaluator.MaxValue != nil {
		maxValue = sql.NullFloat64{Float64: *evaluator.MaxValue, Valid: true}
	}

	result, err := r.db.Exec(ctx, query,
		evaluator.ID,
		evaluator.Name,
		evaluator.Description,
		evaluator.Provider,
		evaluator.Model,
		evaluator.Template,
		evaluator.Config,
		evaluator.OutputType,
		minValue,
		maxValue,
		evaluator.Categories,
		evaluator.Enabled,
		evaluator.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete soft deletes an evaluator
func (r *EvaluatorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE evaluators SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List retrieves evaluators based on filter criteria
func (r *EvaluatorRepository) List(ctx context.Context, filter *domain.EvaluatorFilter) ([]*domain.Evaluator, int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.ProjectID != "" {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, filter.ProjectID)
		argIndex++
	}

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, filter.Type)
		argIndex++
	}

	if filter.Provider != "" {
		conditions = append(conditions, fmt.Sprintf("provider = $%d", argIndex))
		args = append(args, filter.Provider)
		argIndex++
	}

	if filter.OutputType != "" {
		conditions = append(conditions, fmt.Sprintf("output_type = $%d", argIndex))
		args = append(args, filter.OutputType)
		argIndex++
	}

	if filter.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *filter.Enabled)
		argIndex++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM evaluators WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset

	listQuery := fmt.Sprintf(`
		SELECT id, project_id, name, description, type, provider, model,
			template, config, output_type, min_value, max_value, categories,
			enabled, created_at, updated_at
		FROM evaluators
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var evaluators []*domain.Evaluator
	for rows.Next() {
		var evaluator domain.Evaluator
		var minValue, maxValue sql.NullFloat64
		if err := rows.Scan(
			&evaluator.ID,
			&evaluator.ProjectID,
			&evaluator.Name,
			&evaluator.Description,
			&evaluator.Type,
			&evaluator.Provider,
			&evaluator.Model,
			&evaluator.Template,
			&evaluator.Config,
			&evaluator.OutputType,
			&minValue,
			&maxValue,
			&evaluator.Categories,
			&evaluator.Enabled,
			&evaluator.CreatedAt,
			&evaluator.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		// Convert nullable fields to pointers
		if minValue.Valid {
			evaluator.MinValue = &minValue.Float64
		}
		if maxValue.Valid {
			evaluator.MaxValue = &maxValue.Float64
		}

		evaluators = append(evaluators, &evaluator)
	}

	return evaluators, total, nil
}

// EvaluationJobRepository handles database operations for evaluation jobs
type EvaluationJobRepository struct {
	db *pgxpool.Pool
}

// NewEvaluationJobRepository creates a new EvaluationJobRepository
func NewEvaluationJobRepository(db *pgxpool.Pool) *EvaluationJobRepository {
	return &EvaluationJobRepository{db: db}
}

// Create creates a new evaluation job
func (r *EvaluationJobRepository) Create(ctx context.Context, job *domain.EvaluationJob) error {
	targetIDsJSON, err := json.Marshal(job.TargetIDs)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO evaluation_jobs (
			id, project_id, evaluator_id, status, target_type, target_ids,
			total_items, completed, failed, started_at, completed_at,
			total_cost, total_tokens, error_message, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err = r.db.Exec(ctx, query,
		job.ID,
		job.ProjectID,
		job.EvaluatorID,
		job.Status,
		job.TargetType,
		targetIDsJSON,
		job.TotalItems,
		job.Completed,
		job.Failed,
		job.StartedAt,
		job.CompletedAt,
		job.TotalCost,
		job.TotalTokens,
		job.ErrorMessage,
		job.CreatedAt,
		job.UpdatedAt,
	)
	return err
}

// GetByID retrieves an evaluation job by ID
func (r *EvaluationJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EvaluationJob, error) {
	query := `
		SELECT id, project_id, evaluator_id, status, target_type, target_ids,
			total_items, completed, failed, started_at, completed_at,
			total_cost, total_tokens, error_message, created_at, updated_at
		FROM evaluation_jobs
		WHERE id = $1
	`

	var job domain.EvaluationJob
	var targetIDsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.ProjectID,
		&job.EvaluatorID,
		&job.Status,
		&job.TargetType,
		&targetIDsJSON,
		&job.TotalItems,
		&job.Completed,
		&job.Failed,
		&job.StartedAt,
		&job.CompletedAt,
		&job.TotalCost,
		&job.TotalTokens,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(targetIDsJSON, &job.TargetIDs); err != nil {
		return nil, err
	}

	return &job, nil
}

// Update updates an evaluation job
func (r *EvaluationJobRepository) Update(ctx context.Context, job *domain.EvaluationJob) error {
	query := `
		UPDATE evaluation_jobs SET
			status = $2,
			completed = $3,
			failed = $4,
			started_at = $5,
			completed_at = $6,
			total_cost = $7,
			total_tokens = $8,
			error_message = $9,
			updated_at = $10
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		job.ID,
		job.Status,
		job.Completed,
		job.Failed,
		job.StartedAt,
		job.CompletedAt,
		job.TotalCost,
		job.TotalTokens,
		job.ErrorMessage,
		time.Now(),
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List retrieves evaluation jobs based on filter criteria
func (r *EvaluationJobRepository) List(ctx context.Context, filter *domain.EvaluationJobFilter) ([]*domain.EvaluationJob, int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.ProjectID != "" {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, filter.ProjectID)
		argIndex++
	}

	if filter.EvaluatorID != "" {
		conditions = append(conditions, fmt.Sprintf("evaluator_id = $%d", argIndex))
		args = append(args, filter.EvaluatorID)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	whereClause := "1=1"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM evaluation_jobs WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset

	listQuery := fmt.Sprintf(`
		SELECT id, project_id, evaluator_id, status, target_type, target_ids,
			total_items, completed, failed, started_at, completed_at,
			total_cost, total_tokens, error_message, created_at, updated_at
		FROM evaluation_jobs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*domain.EvaluationJob
	for rows.Next() {
		var job domain.EvaluationJob
		var targetIDsJSON []byte
		if err := rows.Scan(
			&job.ID,
			&job.ProjectID,
			&job.EvaluatorID,
			&job.Status,
			&job.TargetType,
			&targetIDsJSON,
			&job.TotalItems,
			&job.Completed,
			&job.Failed,
			&job.StartedAt,
			&job.CompletedAt,
			&job.TotalCost,
			&job.TotalTokens,
			&job.ErrorMessage,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(targetIDsJSON, &job.TargetIDs); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, &job)
	}

	return jobs, total, nil
}

// GetPendingJobs retrieves jobs that are pending execution
func (r *EvaluationJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*domain.EvaluationJob, error) {
	query := `
		SELECT id, project_id, evaluator_id, status, target_type, target_ids,
			total_items, completed, failed, started_at, completed_at,
			total_cost, total_tokens, error_message, created_at, updated_at
		FROM evaluation_jobs
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, domain.EvaluationJobStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.EvaluationJob
	for rows.Next() {
		var job domain.EvaluationJob
		var targetIDsJSON []byte
		if err := rows.Scan(
			&job.ID,
			&job.ProjectID,
			&job.EvaluatorID,
			&job.Status,
			&job.TargetType,
			&targetIDsJSON,
			&job.TotalItems,
			&job.Completed,
			&job.Failed,
			&job.StartedAt,
			&job.CompletedAt,
			&job.TotalCost,
			&job.TotalTokens,
			&job.ErrorMessage,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(targetIDsJSON, &job.TargetIDs); err != nil {
			return nil, err
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}
