package service

import (
	"context"

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
	ProjectID  string
	SessionID  string
	UserID     string
	Model      string
	Name       string   // Search by name
	Status     string   // Filter by status
	Tags       []string // Filter by tags
	StartTime  string   // ISO8601 timestamp
	EndTime    string   // ISO8601 timestamp
	MinLatency int      // Minimum latency in ms
	MaxLatency int      // Maximum latency in ms
	MinCost    float64  // Minimum cost
	MaxCost    float64  // Maximum cost
	SortBy     string   // Field to sort by
	SortOrder  string   // ASC or DESC
	Limit      int
	Offset     int
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
		ProjectID:  opts.ProjectID,
		SessionID:  opts.SessionID,
		UserID:     opts.UserID,
		Model:      opts.Model,
		Name:       opts.Name,
		Status:     opts.Status,
		Tags:       opts.Tags,
		StartTime:  opts.StartTime,
		EndTime:    opts.EndTime,
		MinLatency: opts.MinLatency,
		MaxLatency: opts.MaxLatency,
		MinCost:    opts.MinCost,
		MaxCost:    opts.MaxCost,
		SortBy:     opts.SortBy,
		SortOrder:  opts.SortOrder,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
	})
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
