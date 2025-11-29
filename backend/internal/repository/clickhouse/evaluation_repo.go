package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"github.com/otelguard/otelguard/internal/domain"
)

// EvaluationResultRepository handles ClickHouse operations for evaluation results
type EvaluationResultRepository struct {
	conn clickhouse.Conn
}

// NewEvaluationResultRepository creates a new EvaluationResultRepository
func NewEvaluationResultRepository(conn clickhouse.Conn) *EvaluationResultRepository {
	return &EvaluationResultRepository{conn: conn}
}

// Insert inserts a single evaluation result
func (r *EvaluationResultRepository) Insert(ctx context.Context, result *domain.EvaluationResult) error {
	query := `
		INSERT INTO evaluation_results (
			id, job_id, evaluator_id, project_id, trace_id, span_id,
			score, string_value, reasoning, raw_response,
			prompt_tokens, completion_tokens, cost, latency_ms,
			status, error_message, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return r.conn.Exec(ctx, query,
		result.ID,
		result.JobID,
		result.EvaluatorID,
		result.ProjectID,
		result.TraceID,
		result.SpanID,
		result.Score,
		result.StringValue,
		result.Reasoning,
		result.RawResponse,
		result.PromptTokens,
		result.CompletionTokens,
		result.Cost,
		result.LatencyMs,
		result.Status,
		result.ErrorMessage,
		result.CreatedAt,
	)
}

// InsertBatch inserts multiple evaluation results in a batch
func (r *EvaluationResultRepository) InsertBatch(ctx context.Context, results []*domain.EvaluationResult) error {
	if len(results) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO evaluation_results (
			id, job_id, evaluator_id, project_id, trace_id, span_id,
			score, string_value, reasoning, raw_response,
			prompt_tokens, completion_tokens, cost, latency_ms,
			status, error_message, created_at
		)
	`)
	if err != nil {
		return err
	}

	for _, result := range results {
		err := batch.Append(
			result.ID,
			result.JobID,
			result.EvaluatorID,
			result.ProjectID,
			result.TraceID,
			result.SpanID,
			result.Score,
			result.StringValue,
			result.Reasoning,
			result.RawResponse,
			result.PromptTokens,
			result.CompletionTokens,
			result.Cost,
			result.LatencyMs,
			result.Status,
			result.ErrorMessage,
			result.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// GetByID retrieves an evaluation result by ID
func (r *EvaluationResultRepository) GetByID(ctx context.Context, projectID, id uuid.UUID) (*domain.EvaluationResult, error) {
	query := `
		SELECT id, job_id, evaluator_id, project_id, trace_id, span_id,
			score, string_value, reasoning, raw_response,
			prompt_tokens, completion_tokens, cost, latency_ms,
			status, error_message, created_at
		FROM evaluation_results
		WHERE project_id = ? AND id = ?
		LIMIT 1
	`

	row := r.conn.QueryRow(ctx, query, projectID, id)

	var result domain.EvaluationResult
	err := row.Scan(
		&result.ID,
		&result.JobID,
		&result.EvaluatorID,
		&result.ProjectID,
		&result.TraceID,
		&result.SpanID,
		&result.Score,
		&result.StringValue,
		&result.Reasoning,
		&result.RawResponse,
		&result.PromptTokens,
		&result.CompletionTokens,
		&result.Cost,
		&result.LatencyMs,
		&result.Status,
		&result.ErrorMessage,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// List retrieves evaluation results based on filter criteria
func (r *EvaluationResultRepository) List(ctx context.Context, filter *domain.EvaluationResultFilter) ([]*domain.EvaluationResult, int, error) {
	var conditions []string
	var args []interface{}

	if filter.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, filter.ProjectID)
	}

	if filter.EvaluatorID != "" {
		conditions = append(conditions, "evaluator_id = ?")
		args = append(args, filter.EvaluatorID)
	}

	if filter.JobID != "" {
		conditions = append(conditions, "job_id = ?")
		args = append(args, filter.JobID)
	}

	if filter.TraceID != "" {
		conditions = append(conditions, "trace_id = ?")
		args = append(args, filter.TraceID)
	}

	if filter.SpanID != "" {
		conditions = append(conditions, "span_id = ?")
		args = append(args, filter.SpanID)
	}

	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}

	if !filter.StartDate.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.EndDate)
	}

	whereClause := "1=1"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT count() FROM evaluation_results WHERE %s", whereClause)
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset

	listQuery := fmt.Sprintf(`
		SELECT id, job_id, evaluator_id, project_id, trace_id, span_id,
			score, string_value, reasoning, raw_response,
			prompt_tokens, completion_tokens, cost, latency_ms,
			status, error_message, created_at
		FROM evaluation_results
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := r.conn.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*domain.EvaluationResult
	for rows.Next() {
		var result domain.EvaluationResult
		var jobID, spanID sql.NullString
		var stringValue, reasoning, errorMessage sql.NullString

		if err := rows.Scan(
			&result.ID,
			&jobID,
			&result.EvaluatorID,
			&result.ProjectID,
			&result.TraceID,
			&spanID,
			&result.Score,
			&stringValue,
			&reasoning,
			&result.RawResponse,
			&result.PromptTokens,
			&result.CompletionTokens,
			&result.Cost,
			&result.LatencyMs,
			&result.Status,
			&errorMessage,
			&result.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		// Convert nullable fields to pointers
		if jobID.Valid && jobID.String != "" {
			parsedJobID, err := uuid.Parse(jobID.String)
			if err == nil {
				result.JobID = &parsedJobID
			}
		}

		if spanID.Valid && spanID.String != "" {
			parsedSpanID, err := uuid.Parse(spanID.String)
			if err == nil {
				result.SpanID = &parsedSpanID
			}
		}

		if stringValue.Valid {
			result.StringValue = &stringValue.String
		}

		if reasoning.Valid {
			result.Reasoning = &reasoning.String
		}

		if errorMessage.Valid {
			result.ErrorMessage = &errorMessage.String
		}

		results = append(results, &result)
	}

	return results, int(total), nil
}

// GetByTrace retrieves all evaluation results for a specific trace
func (r *EvaluationResultRepository) GetByTrace(ctx context.Context, projectID, traceID uuid.UUID) ([]*domain.EvaluationResult, error) {
	query := `
		SELECT id, job_id, evaluator_id, project_id, trace_id, span_id,
			score, string_value, reasoning, raw_response,
			prompt_tokens, completion_tokens, cost, latency_ms,
			status, error_message, created_at
		FROM evaluation_results
		WHERE project_id = ? AND trace_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.conn.Query(ctx, query, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.EvaluationResult
	for rows.Next() {
		var result domain.EvaluationResult
		var jobID, spanID sql.NullString
		var stringValue, reasoning, errorMessage sql.NullString

		if err := rows.Scan(
			&result.ID,
			&jobID,
			&result.EvaluatorID,
			&result.ProjectID,
			&result.TraceID,
			&spanID,
			&result.Score,
			&stringValue,
			&reasoning,
			&result.RawResponse,
			&result.PromptTokens,
			&result.CompletionTokens,
			&result.Cost,
			&result.LatencyMs,
			&result.Status,
			&errorMessage,
			&result.CreatedAt,
		); err != nil {
			return nil, err
		}

		// Convert nullable fields to pointers
		if jobID.Valid && jobID.String != "" {
			parsedJobID, err := uuid.Parse(jobID.String)
			if err == nil {
				result.JobID = &parsedJobID
			}
		}

		if spanID.Valid && spanID.String != "" {
			parsedSpanID, err := uuid.Parse(spanID.String)
			if err == nil {
				result.SpanID = &parsedSpanID
			}
		}

		if stringValue.Valid {
			result.StringValue = &stringValue.String
		}

		if reasoning.Valid {
			result.Reasoning = &reasoning.String
		}

		if errorMessage.Valid {
			result.ErrorMessage = &errorMessage.String
		}

		results = append(results, &result)
	}

	return results, nil
}

// GetStats retrieves aggregated evaluation statistics
func (r *EvaluationResultRepository) GetStats(ctx context.Context, projectID uuid.UUID, evaluatorID *uuid.UUID, startDate, endDate time.Time) (*domain.EvaluationStats, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "project_id = ?")
	args = append(args, projectID)

	if evaluatorID != nil {
		conditions = append(conditions, "evaluator_id = ?")
		args = append(args, *evaluatorID)
	}

	if !startDate.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, startDate)
	}

	if !endDate.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, endDate)
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			evaluator_id,
			count() as total_evaluations,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			ifNull(avg(score), 0) as avg_score,
			ifNull(min(score), 0) as min_score,
			ifNull(max(score), 0) as max_score,
			ifNull(sum(cost), 0) as total_cost,
			sum(prompt_tokens + completion_tokens) as total_tokens,
			ifNull(avg(latency_ms), 0) as avg_latency_ms
		FROM evaluation_results
		WHERE %s
		GROUP BY evaluator_id
	`, whereClause)

	var stats domain.EvaluationStats
	row := r.conn.QueryRow(ctx, query, args...)
	err := row.Scan(
		&stats.EvaluatorID,
		&stats.TotalEvaluations,
		&stats.SuccessCount,
		&stats.ErrorCount,
		&stats.AvgScore,
		&stats.MinScore,
		&stats.MaxScore,
		&stats.TotalCost,
		&stats.TotalTokens,
		&stats.AvgLatencyMs,
	)
	if err != nil {
		// If no rows found, return empty stats with evaluatorID if provided
		if err.Error() == "EOF" || strings.Contains(err.Error(), "no rows") {
			emptyStats := &domain.EvaluationStats{
				TotalEvaluations: 0,
				SuccessCount:     0,
				ErrorCount:       0,
				AvgScore:         0,
				MinScore:         0,
				MaxScore:         0,
				TotalCost:        0,
				TotalTokens:      0,
				AvgLatencyMs:     0,
			}
			if evaluatorID != nil {
				emptyStats.EvaluatorID = *evaluatorID
			}
			return emptyStats, nil
		}
		return nil, err
	}

	return &stats, nil
}

// GetCostSummary retrieves cost summary by evaluator
func (r *EvaluationResultRepository) GetCostSummary(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]*domain.EvaluationCostSummary, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "r.project_id = ?")
	args = append(args, projectID)

	if !startDate.IsZero() {
		conditions = append(conditions, "r.created_at >= ?")
		args = append(args, startDate)
	}

	if !endDate.IsZero() {
		conditions = append(conditions, "r.created_at <= ?")
		args = append(args, endDate)
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			r.evaluator_id,
			sum(r.cost) as total_cost,
			sum(r.prompt_tokens + r.completion_tokens) as total_tokens,
			count() as eval_count,
			sum(r.cost) / count() as avg_cost_per_eval
		FROM evaluation_results r
		WHERE %s
		GROUP BY r.evaluator_id
		ORDER BY total_cost DESC
	`, whereClause)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*domain.EvaluationCostSummary
	for rows.Next() {
		var summary domain.EvaluationCostSummary
		if err := rows.Scan(
			&summary.EvaluatorID,
			&summary.TotalCost,
			&summary.TotalTokens,
			&summary.EvalCount,
			&summary.AvgCostPerEval,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// GetScoreDistribution retrieves score distribution for an evaluator
func (r *EvaluationResultRepository) GetScoreDistribution(ctx context.Context, projectID, evaluatorID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "project_id = ?")
	args = append(args, projectID)

	conditions = append(conditions, "evaluator_id = ?")
	args = append(args, evaluatorID)

	conditions = append(conditions, "status = 'success'")

	if !startDate.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, startDate)
	}

	if !endDate.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, endDate)
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			toString(round(score, 1)) as score_bucket,
			count() as count
		FROM evaluation_results
		WHERE %s
		GROUP BY score_bucket
		ORDER BY score_bucket
	`, whereClause)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[string]int64)
	for rows.Next() {
		var bucket string
		var count int64
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		distribution[bucket] = count
	}

	return distribution, nil
}
