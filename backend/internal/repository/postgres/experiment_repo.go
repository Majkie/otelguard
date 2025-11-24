package postgres

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// ExperimentRepository handles experiment data access
type ExperimentRepository struct {
	db *pgxpool.Pool
}

// NewExperimentRepository creates a new experiment repository
func NewExperimentRepository(db *pgxpool.Pool) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

// Create creates a new experiment
func (r *ExperimentRepository) Create(ctx context.Context, experiment *domain.Experiment) error {
	query := `
		INSERT INTO experiments (id, project_id, dataset_id, name, description, config, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		experiment.ID,
		experiment.ProjectID,
		experiment.DatasetID,
		experiment.Name,
		experiment.Description,
		experiment.Config,
		experiment.Status,
		experiment.CreatedBy,
		experiment.CreatedAt,
		experiment.UpdatedAt,
	)
	return err
}

// GetByID retrieves an experiment by ID
func (r *ExperimentRepository) GetByID(ctx context.Context, id string) (*domain.Experiment, error) {
	var experiment domain.Experiment
	query := `
		SELECT id, project_id, dataset_id, name, description, config, status, created_by, created_at, updated_at
		FROM experiments
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &experiment, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &experiment, nil
}

// List returns experiments for a project with pagination
func (r *ExperimentRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Experiment, int, error) {
	var experiments []*domain.Experiment
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
	countQuery := `SELECT COUNT(*) FROM experiments WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, project_id, dataset_id, name, description, config, status, created_by, created_at, updated_at
		FROM experiments
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &experiments, listQuery, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return experiments, total, nil
}

// ListByDataset returns experiments for a specific dataset
func (r *ExperimentRepository) ListByDataset(ctx context.Context, datasetID string) ([]*domain.Experiment, error) {
	var experiments []*domain.Experiment
	query := `
		SELECT id, project_id, dataset_id, name, description, config, status, created_by, created_at, updated_at
		FROM experiments
		WHERE dataset_id = $1
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &experiments, query, datasetID)
	return experiments, err
}

// UpdateStatus updates an experiment's status
func (r *ExperimentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE experiments
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CreateRun creates a new experiment run
func (r *ExperimentRepository) CreateRun(ctx context.Context, run *domain.ExperimentRun) error {
	query := `
		INSERT INTO experiment_runs (id, experiment_id, run_number, status, started_at, total_items, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		run.ID,
		run.ExperimentID,
		run.RunNumber,
		run.Status,
		run.StartedAt,
		run.TotalItems,
		run.CreatedAt,
	)
	return err
}

// GetRunByID retrieves an experiment run by ID
func (r *ExperimentRepository) GetRunByID(ctx context.Context, id string) (*domain.ExperimentRun, error) {
	var run domain.ExperimentRun
	query := `
		SELECT id, experiment_id, run_number, status, started_at, completed_at,
		       total_items, completed_items, failed_items, total_cost, total_latency_ms, error, created_at
		FROM experiment_runs
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &run, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// ListRuns returns runs for an experiment
func (r *ExperimentRepository) ListRuns(ctx context.Context, experimentID string) ([]*domain.ExperimentRun, error) {
	var runs []*domain.ExperimentRun
	query := `
		SELECT id, experiment_id, run_number, status, started_at, completed_at,
		       total_items, completed_items, failed_items, total_cost, total_latency_ms, error, created_at
		FROM experiment_runs
		WHERE experiment_id = $1
		ORDER BY run_number DESC
	`
	err := pgxscan.Select(ctx, r.db, &runs, query, experimentID)
	return runs, err
}

// GetNextRunNumber gets the next run number for an experiment
func (r *ExperimentRepository) GetNextRunNumber(ctx context.Context, experimentID string) (int, error) {
	var runNumber int
	query := `SELECT COALESCE(MAX(run_number), 0) + 1 FROM experiment_runs WHERE experiment_id = $1`
	err := pgxscan.Get(ctx, r.db, &runNumber, query, experimentID)
	return runNumber, err
}

// UpdateRun updates an experiment run
func (r *ExperimentRepository) UpdateRun(ctx context.Context, run *domain.ExperimentRun) error {
	query := `
		UPDATE experiment_runs
		SET status = $2, completed_at = $3, completed_items = $4, failed_items = $5,
		    total_cost = $6, total_latency_ms = $7, error = $8
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query,
		run.ID,
		run.Status,
		run.CompletedAt,
		run.CompletedItems,
		run.FailedItems,
		run.TotalCost,
		run.TotalLatency,
		run.Error,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// IncrementRunProgress increments the progress counters for a run
func (r *ExperimentRepository) IncrementRunProgress(ctx context.Context, runID string, success bool, cost float64, latencyMs int64) error {
	var query string
	if success {
		query = `
			UPDATE experiment_runs
			SET completed_items = completed_items + 1,
			    total_cost = total_cost + $2,
			    total_latency_ms = total_latency_ms + $3
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE experiment_runs
			SET failed_items = failed_items + 1
			WHERE id = $1
		`
	}
	_, err := r.db.Exec(ctx, query, runID, cost, latencyMs)
	return err
}

// CreateResult creates a new experiment result
func (r *ExperimentRepository) CreateResult(ctx context.Context, result *domain.ExperimentResult) error {
	query := `
		INSERT INTO experiment_results (id, run_id, dataset_item_id, trace_id, output, scores, latency_ms, tokens_used, cost, status, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		result.ID,
		result.RunID,
		result.DatasetItemID,
		result.TraceID,
		result.Output,
		result.Scores,
		result.LatencyMs,
		result.TokensUsed,
		result.Cost,
		result.Status,
		result.Error,
		result.CreatedAt,
	)
	return err
}

// GetResultsByRunID returns all results for a run
func (r *ExperimentRepository) GetResultsByRunID(ctx context.Context, runID string) ([]*domain.ExperimentResult, error) {
	var results []*domain.ExperimentResult
	query := `
		SELECT id, run_id, dataset_item_id, trace_id, output, scores, latency_ms, tokens_used, cost, status, error, created_at
		FROM experiment_results
		WHERE run_id = $1
		ORDER BY created_at ASC
	`
	err := pgxscan.Select(ctx, r.db, &results, query, runID)
	return results, err
}

// GetResultByRunAndItem gets a specific result by run and dataset item
func (r *ExperimentRepository) GetResultByRunAndItem(ctx context.Context, runID, datasetItemID string) (*domain.ExperimentResult, error) {
	var result domain.ExperimentResult
	query := `
		SELECT id, run_id, dataset_item_id, trace_id, output, scores, latency_ms, tokens_used, cost, status, error, created_at
		FROM experiment_results
		WHERE run_id = $1 AND dataset_item_id = $2
	`
	err := pgxscan.Get(ctx, r.db, &result, query, runID, datasetItemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

// BulkCreateResults creates multiple experiment results in a transaction
func (r *ExperimentRepository) BulkCreateResults(ctx context.Context, results []*domain.ExperimentResult) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO experiment_results (id, run_id, dataset_item_id, trace_id, output, scores, latency_ms, tokens_used, cost, status, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	for _, result := range results {
		_, err := tx.Exec(ctx, query,
			result.ID,
			result.RunID,
			result.DatasetItemID,
			result.TraceID,
			result.Output,
			result.Scores,
			result.LatencyMs,
			result.TokensUsed,
			result.Cost,
			result.Status,
			result.Error,
			result.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
