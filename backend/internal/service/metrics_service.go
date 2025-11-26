package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MetricsService provides aggregated metrics and analytics
type MetricsService struct {
	clickhouse clickhouse.Conn
	logger     *zap.Logger
}

// NewMetricsService creates a new metrics service
func NewMetricsService(clickhouse clickhouse.Conn, logger *zap.Logger) *MetricsService {
	return &MetricsService{
		clickhouse: clickhouse,
		logger:     logger,
	}
}

// MetricsFilter defines filtering options for metrics queries
type MetricsFilter struct {
	ProjectID uuid.UUID
	StartTime time.Time
	EndTime   time.Time
	Model     string
	UserID    string
	SessionID string
}

// CoreMetrics represents aggregate metrics for a time period
type CoreMetrics struct {
	TotalTraces       int64   `json:"totalTraces"`
	TotalSpans        int64   `json:"totalSpans"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
	P50LatencyMs      float64 `json:"p50LatencyMs"`
	P95LatencyMs      float64 `json:"p95LatencyMs"`
	P99LatencyMs      float64 `json:"p99LatencyMs"`
	TotalCost         float64 `json:"totalCost"`
	AvgCost           float64 `json:"avgCost"`
	TotalTokens       int64   `json:"totalTokens"`
	TotalPromptTokens int64   `json:"totalPromptTokens"`
	TotalCompTokens   int64   `json:"totalCompletionTokens"`
	ErrorCount        int64   `json:"errorCount"`
	ErrorRate         float64 `json:"errorRate"`
}

// TimeSeriesPoint represents a single point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count,omitempty"`
}

// TimeSeriesData represents a complete time series
type TimeSeriesData struct {
	MetricName string             `json:"metricName"`
	Points     []*TimeSeriesPoint `json:"points"`
}

// ModelMetrics represents metrics broken down by model
type ModelMetrics struct {
	Model             string  `json:"model"`
	TraceCount        int64   `json:"traceCount"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
	TotalCost         float64 `json:"totalCost"`
	TotalTokens       int64   `json:"totalTokens"`
	TotalPromptTokens int64   `json:"totalPromptTokens"`
	TotalCompTokens   int64   `json:"totalCompletionTokens"`
	ErrorCount        int64   `json:"errorCount"`
	ErrorRate         float64 `json:"errorRate"`
}

// UserMetrics represents metrics broken down by user
type UserMetrics struct {
	UserID       string  `json:"userId"`
	TraceCount   int64   `json:"traceCount"`
	TotalCost    float64 `json:"totalCost"`
	TotalTokens  int64   `json:"totalTokens"`
	AvgLatency   float64 `json:"avgLatency"`
	ErrorCount   int64   `json:"errorCount"`
	LastActivity time.Time `json:"lastActivity"`
}

// GetCoreMetrics retrieves aggregated core metrics for a project
func (s *MetricsService) GetCoreMetrics(ctx context.Context, filter *MetricsFilter) (*CoreMetrics, error) {
	query := `
		SELECT
			count() as total_traces,
			countIf(status = 'error') as error_count,
			avg(latency_ms) as avg_latency,
			quantile(0.5)(latency_ms) as p50_latency,
			quantile(0.95)(latency_ms) as p95_latency,
			quantile(0.99)(latency_ms) as p99_latency,
			sum(cost) as total_cost,
			avg(cost) as avg_cost,
			sum(total_tokens) as total_tokens,
			sum(prompt_tokens) as total_prompt_tokens,
			sum(completion_tokens) as total_completion_tokens
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`

	args := []interface{}{filter.ProjectID, filter.StartTime, filter.EndTime}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, filter.SessionID)
	}

	row := s.clickhouse.QueryRow(ctx, query, args...)

	var metrics CoreMetrics
	var totalSpans int64 // We'll query this separately
	err := row.Scan(
		&metrics.TotalTraces,
		&metrics.ErrorCount,
		&metrics.AvgLatencyMs,
		&metrics.P50LatencyMs,
		&metrics.P95LatencyMs,
		&metrics.P99LatencyMs,
		&metrics.TotalCost,
		&metrics.AvgCost,
		&metrics.TotalTokens,
		&metrics.TotalPromptTokens,
		&metrics.TotalCompTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query core metrics: %w", err)
	}

	// Calculate error rate
	if metrics.TotalTraces > 0 {
		metrics.ErrorRate = float64(metrics.ErrorCount) / float64(metrics.TotalTraces)
	}

	// Query span count
	spanQuery := `
		SELECT count()
		FROM spans
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`
	spanArgs := []interface{}{filter.ProjectID, filter.StartTime, filter.EndTime}

	if err := s.clickhouse.QueryRow(ctx, spanQuery, spanArgs...).Scan(&totalSpans); err != nil {
		s.logger.Warn("failed to query span count", zap.Error(err))
	}
	metrics.TotalSpans = totalSpans

	return &metrics, nil
}

// GetTimeSeriesMetrics retrieves time-series data for a specific metric
func (s *MetricsService) GetTimeSeriesMetrics(ctx context.Context, filter *MetricsFilter, metricName string, interval string) (*TimeSeriesData, error) {
	// Determine aggregation interval
	var timeFunc string
	switch interval {
	case "hour":
		timeFunc = "toStartOfHour(start_time)"
	case "day":
		timeFunc = "toStartOfDay(start_time)"
	case "week":
		timeFunc = "toStartOfWeek(start_time)"
	case "month":
		timeFunc = "toStartOfMonth(start_time)"
	default:
		timeFunc = "toStartOfHour(start_time)"
	}

	// Determine metric aggregation
	var metricAgg string
	switch metricName {
	case "traces":
		metricAgg = "count() as value"
	case "latency":
		metricAgg = "avg(latency_ms) as value"
	case "cost":
		metricAgg = "sum(cost) as value"
	case "tokens":
		metricAgg = "sum(total_tokens) as value"
	case "errors":
		metricAgg = "countIf(status = 'error') as value"
	case "error_rate":
		metricAgg = "countIf(status = 'error') / count() as value"
	default:
		return nil, fmt.Errorf("unknown metric: %s", metricName)
	}

	query := fmt.Sprintf(`
		SELECT
			%s as timestamp,
			%s,
			count() as count
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`, timeFunc, metricAgg)

	args := []interface{}{filter.ProjectID, filter.StartTime, filter.EndTime}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, filter.SessionID)
	}

	query += " GROUP BY timestamp ORDER BY timestamp"

	rows, err := s.clickhouse.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series: %w", err)
	}
	defer rows.Close()

	var points []*TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}
		points = append(points, &point)
	}

	return &TimeSeriesData{
		MetricName: metricName,
		Points:     points,
	}, nil
}

// GetModelBreakdown retrieves metrics broken down by model
func (s *MetricsService) GetModelBreakdown(ctx context.Context, filter *MetricsFilter) ([]*ModelMetrics, error) {
	query := `
		SELECT
			model,
			count() as trace_count,
			avg(latency_ms) as avg_latency,
			sum(cost) as total_cost,
			sum(total_tokens) as total_tokens,
			sum(prompt_tokens) as total_prompt_tokens,
			sum(completion_tokens) as total_completion_tokens,
			countIf(status = 'error') as error_count
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`

	args := []interface{}{filter.ProjectID, filter.StartTime, filter.EndTime}

	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, filter.SessionID)
	}

	query += " GROUP BY model ORDER BY trace_count DESC"

	rows, err := s.clickhouse.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query model breakdown: %w", err)
	}
	defer rows.Close()

	var results []*ModelMetrics
	for rows.Next() {
		var m ModelMetrics
		if err := rows.Scan(
			&m.Model,
			&m.TraceCount,
			&m.AvgLatencyMs,
			&m.TotalCost,
			&m.TotalTokens,
			&m.TotalPromptTokens,
			&m.TotalCompTokens,
			&m.ErrorCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan model metrics: %w", err)
		}

		if m.TraceCount > 0 {
			m.ErrorRate = float64(m.ErrorCount) / float64(m.TraceCount)
		}

		results = append(results, &m)
	}

	return results, nil
}

// GetUserBreakdown retrieves metrics broken down by user
func (s *MetricsService) GetUserBreakdown(ctx context.Context, filter *MetricsFilter, limit int) ([]*UserMetrics, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			user_id,
			count() as trace_count,
			sum(cost) as total_cost,
			sum(total_tokens) as total_tokens,
			avg(latency_ms) as avg_latency,
			countIf(status = 'error') as error_count,
			max(start_time) as last_activity
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
		  AND user_id != ''
	`

	args := []interface{}{filter.ProjectID, filter.StartTime, filter.EndTime}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}

	query += fmt.Sprintf(" GROUP BY user_id ORDER BY trace_count DESC LIMIT %d", limit)

	rows, err := s.clickhouse.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user breakdown: %w", err)
	}
	defer rows.Close()

	var results []*UserMetrics
	for rows.Next() {
		var m UserMetrics
		if err := rows.Scan(
			&m.UserID,
			&m.TraceCount,
			&m.TotalCost,
			&m.TotalTokens,
			&m.AvgLatency,
			&m.ErrorCount,
			&m.LastActivity,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user metrics: %w", err)
		}

		results = append(results, &m)
	}

	return results, nil
}

// CostBreakdown represents cost metrics for different dimensions
type CostBreakdown struct {
	TotalCost      float64                  `json:"totalCost"`
	CostByModel    map[string]float64       `json:"costByModel"`
	CostByUser     map[string]float64       `json:"costByUser"`
	CostOverTime   []*TimeSeriesPoint       `json:"costOverTime"`
	TopCostModels  []*ModelCostSummary      `json:"topCostModels"`
	TopCostUsers   []*UserCostSummary       `json:"topCostUsers"`
}

type ModelCostSummary struct {
	Model       string  `json:"model"`
	TotalCost   float64 `json:"totalCost"`
	TraceCount  int64   `json:"traceCount"`
	AvgCost     float64 `json:"avgCost"`
	TotalTokens int64   `json:"totalTokens"`
}

type UserCostSummary struct {
	UserID      string  `json:"userId"`
	TotalCost   float64 `json:"totalCost"`
	TraceCount  int64   `json:"traceCount"`
	AvgCost     float64 `json:"avgCost"`
	TotalTokens int64   `json:"totalTokens"`
}

// GetCostBreakdown retrieves comprehensive cost analytics
func (s *MetricsService) GetCostBreakdown(ctx context.Context, filter *MetricsFilter) (*CostBreakdown, error) {
	breakdown := &CostBreakdown{
		CostByModel: make(map[string]float64),
		CostByUser:  make(map[string]float64),
	}

	// Get total cost
	totalQuery := `
		SELECT sum(cost) as total_cost
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`
	if err := s.clickhouse.QueryRow(ctx, totalQuery, filter.ProjectID, filter.StartTime, filter.EndTime).Scan(&breakdown.TotalCost); err != nil {
		return nil, fmt.Errorf("failed to query total cost: %w", err)
	}

	// Get cost by model
	modelQuery := `
		SELECT
			model,
			sum(cost) as total_cost,
			count() as trace_count,
			sum(total_tokens) as total_tokens
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
		GROUP BY model
		ORDER BY total_cost DESC
		LIMIT 20
	`
	rows, err := s.clickhouse.Query(ctx, modelQuery, filter.ProjectID, filter.StartTime, filter.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost by model: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var summary ModelCostSummary
		if err := rows.Scan(&summary.Model, &summary.TotalCost, &summary.TraceCount, &summary.TotalTokens); err != nil {
			return nil, err
		}
		if summary.TraceCount > 0 {
			summary.AvgCost = summary.TotalCost / float64(summary.TraceCount)
		}
		breakdown.CostByModel[summary.Model] = summary.TotalCost
		breakdown.TopCostModels = append(breakdown.TopCostModels, &summary)
	}

	// Get cost by user
	userQuery := `
		SELECT
			user_id,
			sum(cost) as total_cost,
			count() as trace_count,
			sum(total_tokens) as total_tokens
		FROM traces
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
		  AND user_id != ''
		GROUP BY user_id
		ORDER BY total_cost DESC
		LIMIT 20
	`
	userRows, err := s.clickhouse.Query(ctx, userQuery, filter.ProjectID, filter.StartTime, filter.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost by user: %w", err)
	}
	defer userRows.Close()

	for userRows.Next() {
		var summary UserCostSummary
		if err := userRows.Scan(&summary.UserID, &summary.TotalCost, &summary.TraceCount, &summary.TotalTokens); err != nil {
			return nil, err
		}
		if summary.TraceCount > 0 {
			summary.AvgCost = summary.TotalCost / float64(summary.TraceCount)
		}
		breakdown.CostByUser[summary.UserID] = summary.TotalCost
		breakdown.TopCostUsers = append(breakdown.TopCostUsers, &summary)
	}

	// Get cost over time
	timeSeriesData, err := s.GetTimeSeriesMetrics(ctx, filter, "cost", "day")
	if err != nil {
		s.logger.Warn("failed to get cost time series", zap.Error(err))
	} else {
		breakdown.CostOverTime = timeSeriesData.Points
	}

	return breakdown, nil
}

// QualityMetrics represents quality-related metrics
type QualityMetrics struct {
	TotalScores      int64            `json:"totalScores"`
	AvgScore         float64          `json:"avgScore"`
	ScoresByName     map[string]float64 `json:"scoresByName"`
	FeedbackCount    int64            `json:"feedbackCount"`
	PositiveFeedback int64            `json:"positiveFeedback"`
	NegativeFeedback int64            `json:"negativeFeedback"`
	FeedbackRate     float64          `json:"feedbackRate"`
}

// GetQualityMetrics retrieves quality and evaluation metrics
func (s *MetricsService) GetQualityMetrics(ctx context.Context, filter *MetricsFilter) (*QualityMetrics, error) {
	metrics := &QualityMetrics{
		ScoresByName: make(map[string]float64),
	}

	// Get score metrics
	scoreQuery := `
		SELECT
			count() as total_scores,
			avg(value) as avg_score
		FROM scores
		WHERE project_id = ?
		  AND created_at >= ?
		  AND created_at <= ?
		  AND data_type = 'numeric'
	`
	if err := s.clickhouse.QueryRow(ctx, scoreQuery, filter.ProjectID, filter.StartTime, filter.EndTime).Scan(&metrics.TotalScores, &metrics.AvgScore); err != nil {
		s.logger.Warn("failed to query score metrics", zap.Error(err))
	}

	// Get scores by name
	scoreNameQuery := `
		SELECT
			name,
			avg(value) as avg_value
		FROM scores
		WHERE project_id = ?
		  AND created_at >= ?
		  AND created_at <= ?
		  AND data_type = 'numeric'
		GROUP BY name
	`
	rows, err := s.clickhouse.Query(ctx, scoreNameQuery, filter.ProjectID, filter.StartTime, filter.EndTime)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var avgValue float64
			if err := rows.Scan(&name, &avgValue); err == nil {
				metrics.ScoresByName[name] = avgValue
			}
		}
	}

	return metrics, nil
}
