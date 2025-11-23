package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// TraceService handles trace business logic
type TraceService struct {
	traceRepo   *clickhouse.TraceRepository
	batchWriter *clickhouse.TraceBatchWriter
	sampler     *ProjectSampler
	asyncWrite  bool
	logger      *zap.Logger
}

// TraceServiceConfig contains configuration for trace service
type TraceServiceConfig struct {
	AsyncWrite     bool
	SamplerEnabled bool
	SamplerConfig  *SamplerConfig
}

// NewTraceService creates a new trace service
func NewTraceService(traceRepo *clickhouse.TraceRepository, logger *zap.Logger) *TraceService {
	return &TraceService{
		traceRepo:  traceRepo,
		asyncWrite: false,
		logger:     logger,
	}
}

// NewTraceServiceWithBatchWriter creates a trace service with batch writer support
func NewTraceServiceWithBatchWriter(
	traceRepo *clickhouse.TraceRepository,
	batchWriter *clickhouse.TraceBatchWriter,
	asyncWrite bool,
	logger *zap.Logger,
) *TraceService {
	return &TraceService{
		traceRepo:   traceRepo,
		batchWriter: batchWriter,
		asyncWrite:  asyncWrite,
		logger:      logger,
	}
}

// NewTraceServiceFull creates a trace service with all features
func NewTraceServiceFull(
	traceRepo *clickhouse.TraceRepository,
	batchWriter *clickhouse.TraceBatchWriter,
	asyncWrite bool,
	samplerConfig *SamplerConfig,
	logger *zap.Logger,
) *TraceService {
	var sampler *ProjectSampler
	if samplerConfig != nil {
		sampler = NewProjectSampler(samplerConfig, logger)
	}

	return &TraceService{
		traceRepo:   traceRepo,
		batchWriter: batchWriter,
		sampler:     sampler,
		asyncWrite:  asyncWrite,
		logger:      logger,
	}
}

// SetProjectSamplerConfig sets sampling configuration for a specific project
func (s *TraceService) SetProjectSamplerConfig(projectID string, config *SamplerConfig) {
	if s.sampler != nil {
		s.sampler.SetProjectConfig(projectID, config)
	}
}

// GetSamplerStats returns current sampler statistics
func (s *TraceService) GetSamplerStats() *SamplerStats {
	if s.sampler == nil {
		return nil
	}
	return s.sampler.GetStats()
}

// ListTracesOptions contains options for listing traces
type ListTracesOptions struct {
	ProjectID     string
	SessionID     string
	UserID        string
	Model         string
	Name          string   // Search by name
	Status        string   // Filter by status
	Tags          []string // Filter by tags
	StartTime     string   // ISO8601 timestamp
	EndTime       string   // ISO8601 timestamp
	MinLatency    int      // Minimum latency in ms
	MaxLatency    int      // Maximum latency in ms
	MinCost       float64  // Minimum cost
	MaxCost       float64  // Maximum cost
	PromptID      string   // Filter by prompt ID
	PromptVersion string   // Filter by prompt version
	SortBy        string   // Field to sort by
	SortOrder     string   // ASC or DESC
	Limit         int
	Offset        int
}

// IngestTrace ingests a single trace
func (s *TraceService) IngestTrace(ctx context.Context, trace *domain.Trace) error {
	// Apply sampling if configured
	if s.sampler != nil && !s.sampler.ShouldSample(ctx, trace) {
		s.logger.Debug("trace dropped by sampler",
			zap.String("trace_id", trace.ID.String()),
			zap.String("project_id", trace.ProjectID.String()),
		)
		return nil
	}

	if s.asyncWrite && s.batchWriter != nil {
		return s.batchWriter.WriteTrace(ctx, trace)
	}
	return s.traceRepo.Insert(ctx, []*domain.Trace{trace})
}

// IngestBatch ingests multiple traces
func (s *TraceService) IngestBatch(ctx context.Context, traces []*domain.Trace) error {
	// Apply sampling if configured
	if s.sampler != nil {
		sampledTraces := make([]*domain.Trace, 0, len(traces))
		for _, trace := range traces {
			if s.sampler.ShouldSample(ctx, trace) {
				sampledTraces = append(sampledTraces, trace)
			} else {
				s.logger.Debug("trace dropped by sampler",
					zap.String("trace_id", trace.ID.String()),
					zap.String("project_id", trace.ProjectID.String()),
				)
			}
		}
		traces = sampledTraces

		if len(traces) == 0 {
			return nil
		}
	}

	if s.asyncWrite && s.batchWriter != nil {
		return s.batchWriter.WriteTraces(ctx, traces)
	}
	return s.traceRepo.Insert(ctx, traces)
}

// IngestSpan ingests a single span
func (s *TraceService) IngestSpan(ctx context.Context, span *domain.Span) error {
	if s.asyncWrite && s.batchWriter != nil {
		return s.batchWriter.WriteSpan(ctx, span)
	}
	return s.traceRepo.InsertSpan(ctx, span)
}

// SubmitScore submits an evaluation score
func (s *TraceService) SubmitScore(ctx context.Context, score *domain.Score) error {
	if s.asyncWrite && s.batchWriter != nil {
		return s.batchWriter.WriteScore(ctx, score)
	}
	return s.traceRepo.InsertScore(ctx, score)
}

// GetScores retrieves scores with filtering
func (s *TraceService) GetScores(ctx context.Context, filter *clickhouse.ScoreFilter) ([]*domain.Score, int, error) {
	return s.traceRepo.GetScores(ctx, filter)
}

// GetScoreByID retrieves a single score by ID
func (s *TraceService) GetScoreByID(ctx context.Context, projectID, scoreID uuid.UUID) (*domain.Score, error) {
	return s.traceRepo.GetScoreByID(ctx, projectID, scoreID)
}

// GetScoreAggregations retrieves aggregated statistics for scores
func (s *TraceService) GetScoreAggregations(ctx context.Context, filter *clickhouse.ScoreFilter) ([]*clickhouse.ScoreAggregation, error) {
	return s.traceRepo.GetScoreAggregations(ctx, filter)
}

// GetScoreTrends retrieves score trends over time
func (s *TraceService) GetScoreTrends(ctx context.Context, filter *clickhouse.ScoreFilter, groupBy string) ([]*clickhouse.ScoreTrend, error) {
	return s.traceRepo.GetScoreTrends(ctx, filter, groupBy)
}

// GetScoreComparisons retrieves score comparisons across dimensions
func (s *TraceService) GetScoreComparisons(ctx context.Context, filter *clickhouse.ScoreFilter, dimension string) ([]*clickhouse.ScoreComparison, error) {
	return s.traceRepo.GetScoreComparisons(ctx, filter, dimension)
}

// GetBatchWriterMetrics returns metrics from the batch writer
func (s *TraceService) GetBatchWriterMetrics() *clickhouse.BatchWriterMetrics {
	if s.batchWriter == nil {
		return nil
	}
	return s.batchWriter.GetMetrics()
}

// ListTraces returns paginated traces
func (s *TraceService) ListTraces(ctx context.Context, opts *ListTracesOptions) ([]*domain.Trace, int, error) {
	return s.traceRepo.Query(ctx, &clickhouse.QueryOptions{
		ProjectID:     opts.ProjectID,
		SessionID:     opts.SessionID,
		UserID:        opts.UserID,
		Model:         opts.Model,
		Name:          opts.Name,
		Status:        opts.Status,
		Tags:          opts.Tags,
		StartTime:     opts.StartTime,
		EndTime:       opts.EndTime,
		MinLatency:    opts.MinLatency,
		MaxLatency:    opts.MaxLatency,
		MinCost:       opts.MinCost,
		MaxCost:       opts.MaxCost,
		PromptID:      opts.PromptID,
		PromptVersion: opts.PromptVersion,
		SortBy:        opts.SortBy,
		SortOrder:     opts.SortOrder,
		Limit:         opts.Limit,
		Offset:        opts.Offset,
	})
}

// GetPromptPerformanceMetrics returns performance metrics for prompts
func (s *TraceService) GetPromptPerformanceMetrics(ctx context.Context, projectID, promptID, startTime, endTime string) ([]*clickhouse.PromptPerformanceMetrics, error) {
	return s.traceRepo.GetPromptPerformanceMetrics(ctx, projectID, promptID, startTime, endTime)
}

// PromptRegressionResult represents the result of regression detection
type PromptRegressionResult struct {
	PromptID          string   `json:"promptId"`
	CurrentVersion    int      `json:"currentVersion"`
	PreviousVersion   int      `json:"previousVersion"`
	LatencyChange     float64  `json:"latencyChange"`               // Percentage change in average latency
	CostChange        float64  `json:"costChange"`                  // Percentage change in average cost
	TokenChange       float64  `json:"tokenChange"`                 // Percentage change in average tokens
	ErrorRateChange   float64  `json:"errorRateChange"`             // Change in error rate (percentage points)
	IsRegression      bool     `json:"isRegression"`                // True if any metric regressed significantly
	RegressionReasons []string `json:"regressionReasons,omitempty"` // Reasons for regression
}

// DetectPromptRegressions detects performance regressions between prompt versions
func (s *TraceService) DetectPromptRegressions(ctx context.Context, projectID, promptID string) ([]*PromptRegressionResult, error) {
	// Get performance metrics for all versions of this prompt
	metrics, err := s.traceRepo.GetPromptPerformanceMetrics(ctx, projectID, promptID, "", "")
	if err != nil {
		return nil, err
	}

	// Group metrics by version
	versionMetrics := make(map[int][]*clickhouse.PromptPerformanceMetrics)
	for _, m := range metrics {
		versionMetrics[m.PromptVersion] = append(versionMetrics[m.PromptVersion], m)
	}

	// Find all versions, sorted
	var versions []int
	for v := range versionMetrics {
		versions = append(versions, v)
	}

	// Sort versions in descending order (newest first)
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i] < versions[j] {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}

	var results []*PromptRegressionResult

	// Compare each version with the next newer version
	for i := 0; i < len(versions)-1; i++ {
		currentVersion := versions[i]
		previousVersion := versions[i+1]

		currentMetrics := versionMetrics[currentVersion]
		previousMetrics := versionMetrics[previousVersion]

		// Aggregate metrics across all models for each version
		currentAgg := s.aggregateMetrics(currentMetrics)
		previousAgg := s.aggregateMetrics(previousMetrics)

		result := &PromptRegressionResult{
			PromptID:        promptID,
			CurrentVersion:  currentVersion,
			PreviousVersion: previousVersion,
		}

		// Calculate percentage changes
		if previousAgg.avgLatency > 0 {
			result.LatencyChange = ((currentAgg.avgLatency - previousAgg.avgLatency) / previousAgg.avgLatency) * 100
		}
		if previousAgg.avgCost > 0 {
			result.CostChange = ((currentAgg.avgCost - previousAgg.avgCost) / previousAgg.avgCost) * 100
		}
		if previousAgg.avgTokens > 0 {
			result.TokenChange = ((currentAgg.avgTokens - previousAgg.avgTokens) / previousAgg.avgTokens) * 100
		}

		currentErrorRate := float64(currentAgg.errorCount) / float64(currentAgg.traceCount) * 100
		previousErrorRate := float64(previousAgg.errorCount) / float64(previousAgg.traceCount) * 100
		result.ErrorRateChange = currentErrorRate - previousErrorRate

		// Check for regressions (significant increases in latency, cost, tokens, or error rate)
		var reasons []string
		if result.LatencyChange > 10 { // 10% increase in latency
			reasons = append(reasons, "latency")
		}
		if result.CostChange > 15 { // 15% increase in cost
			reasons = append(reasons, "cost")
		}
		if result.TokenChange > 20 { // 20% increase in token usage
			reasons = append(reasons, "token_usage")
		}
		if result.ErrorRateChange > 5 { // 5 percentage point increase in error rate
			reasons = append(reasons, "error_rate")
		}

		result.IsRegression = len(reasons) > 0
		result.RegressionReasons = reasons

		results = append(results, result)
	}

	return results, nil
}

// aggregatedMetrics represents aggregated metrics for a version
type aggregatedMetrics struct {
	traceCount   uint64
	totalLatency uint64
	avgLatency   float64
	totalTokens  uint64
	avgTokens    float64
	totalCost    float64
	avgCost      float64
	errorCount   uint64
}

// aggregateMetrics aggregates metrics across multiple records (e.g., different models/dates)
func (s *TraceService) aggregateMetrics(metrics []*clickhouse.PromptPerformanceMetrics) *aggregatedMetrics {
	if len(metrics) == 0 {
		return &aggregatedMetrics{}
	}

	var totalTraceCount, totalLatency, totalTokens, totalErrorCount uint64
	var totalCost float64

	for _, m := range metrics {
		totalTraceCount += m.TraceCount
		totalLatency += m.TotalLatency
		totalTokens += m.TotalTokens
		totalCost += m.TotalCost
		totalErrorCount += m.ErrorCount
	}

	avgLatency := float64(totalLatency) / float64(totalTraceCount)
	avgTokens := float64(totalTokens) / float64(totalTraceCount)
	avgCost := totalCost / float64(totalTraceCount)

	return &aggregatedMetrics{
		traceCount:   totalTraceCount,
		totalLatency: totalLatency,
		avgLatency:   avgLatency,
		totalTokens:  totalTokens,
		avgTokens:    avgTokens,
		totalCost:    totalCost,
		avgCost:      avgCost,
		errorCount:   totalErrorCount,
	}
}

// GetTrace retrieves a single trace by ID
func (s *TraceService) GetTrace(ctx context.Context, id string) (*domain.Trace, error) {
	return s.traceRepo.GetByID(ctx, id)
}

// GetSpans retrieves spans for a trace
func (s *TraceService) GetSpans(ctx context.Context, traceID string) ([]*domain.Span, error) {
	return s.traceRepo.GetSpans(ctx, traceID)
}

// DeleteTrace deletes a trace
func (s *TraceService) DeleteTrace(ctx context.Context, id string) error {
	// Note: ClickHouse doesn't support deletes easily
	// This would need to be handled via TTL or ALTER DELETE
	s.logger.Warn("trace deletion not fully implemented", zap.String("trace_id", id))
	return nil
}

// ListSessionsOptions contains options for listing sessions
type ListSessionsOptions struct {
	ProjectID string
	UserID    string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// ListSessions returns paginated sessions with aggregated metrics
func (s *TraceService) ListSessions(ctx context.Context, opts *ListSessionsOptions) ([]*clickhouse.Session, int, error) {
	return s.traceRepo.ListSessions(ctx, &clickhouse.SessionQueryOptions{
		ProjectID: opts.ProjectID,
		UserID:    opts.UserID,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
}

// GetSession retrieves a single session with its aggregated metrics
func (s *TraceService) GetSession(ctx context.Context, sessionID string) (*clickhouse.Session, error) {
	return s.traceRepo.GetSessionByID(ctx, sessionID)
}

// GetSessionTraces retrieves all traces for a session
func (s *TraceService) GetSessionTraces(ctx context.Context, sessionID string, limit, offset int) ([]*domain.Trace, int, error) {
	return s.traceRepo.Query(ctx, &clickhouse.QueryOptions{
		SessionID: sessionID,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "start_time",
		SortOrder: "ASC",
	})
}

// ListUsersOptions contains options for listing users
type ListUsersOptions struct {
	ProjectID string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// ListUsers returns paginated users with aggregated metrics
func (s *TraceService) ListUsers(ctx context.Context, opts *ListUsersOptions) ([]*clickhouse.User, int, error) {
	return s.traceRepo.ListUsers(ctx, &clickhouse.UserQueryOptions{
		ProjectID: opts.ProjectID,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
}

// GetUser retrieves a single user with aggregated metrics
func (s *TraceService) GetUser(ctx context.Context, userID string) (*clickhouse.User, error) {
	return s.traceRepo.GetUserByID(ctx, userID)
}

// GetUserTraces retrieves all traces for a user
func (s *TraceService) GetUserTraces(ctx context.Context, userID string, limit, offset int) ([]*domain.Trace, int, error) {
	return s.traceRepo.Query(ctx, &clickhouse.QueryOptions{
		UserID:    userID,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "start_time",
		SortOrder: "DESC",
	})
}

// GetUserSessions retrieves all sessions for a user
func (s *TraceService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*clickhouse.Session, int, error) {
	return s.traceRepo.GetUserSessions(ctx, userID, limit, offset)
}

// SearchTracesOptions contains options for searching traces
type SearchTracesOptions struct {
	ProjectID string
	Query     string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// SearchTraces performs full-text search on trace content
func (s *TraceService) SearchTraces(ctx context.Context, opts *SearchTracesOptions) ([]*domain.Trace, int, error) {
	return s.traceRepo.SearchTraces(ctx, &clickhouse.SearchOptions{
		ProjectID: opts.ProjectID,
		Query:     opts.Query,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
}

// AnalyticsOptions contains options for analytics queries
type AnalyticsOptions struct {
	ProjectID   string
	StartTime   string
	EndTime     string
	Granularity string // hour, day, week
}

// GetOverviewMetrics retrieves overview metrics
func (s *TraceService) GetOverviewMetrics(ctx context.Context, opts *AnalyticsOptions) (*clickhouse.OverviewMetrics, error) {
	return s.traceRepo.GetOverviewMetrics(ctx, &clickhouse.AnalyticsQueryOptions{
		ProjectID:   opts.ProjectID,
		StartTime:   opts.StartTime,
		EndTime:     opts.EndTime,
		Granularity: opts.Granularity,
	})
}

// GetCostAnalytics retrieves cost analytics over time
func (s *TraceService) GetCostAnalytics(ctx context.Context, opts *AnalyticsOptions) ([]*clickhouse.TimeSeriesPoint, float64, []*clickhouse.CostByModel, error) {
	timeSeries, totalCost, err := s.traceRepo.GetCostTimeSeries(ctx, &clickhouse.AnalyticsQueryOptions{
		ProjectID:   opts.ProjectID,
		StartTime:   opts.StartTime,
		EndTime:     opts.EndTime,
		Granularity: opts.Granularity,
	})
	if err != nil {
		return nil, 0, nil, err
	}

	byModel, err := s.traceRepo.GetCostByModel(ctx, &clickhouse.AnalyticsQueryOptions{
		ProjectID: opts.ProjectID,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
	})
	if err != nil {
		return nil, 0, nil, err
	}

	return timeSeries, totalCost, byModel, nil
}

// GetUsageAnalytics retrieves token usage analytics over time
func (s *TraceService) GetUsageAnalytics(ctx context.Context, opts *AnalyticsOptions) ([]*clickhouse.TimeSeriesPoint, int, error) {
	return s.traceRepo.GetUsageTimeSeries(ctx, &clickhouse.AnalyticsQueryOptions{
		ProjectID:   opts.ProjectID,
		StartTime:   opts.StartTime,
		EndTime:     opts.EndTime,
		Granularity: opts.Granularity,
	})
}
