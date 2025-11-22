package service

import (
	"context"

	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// TraceService handles trace business logic
type TraceService struct {
	traceRepo *clickhouse.TraceRepository
	logger    *zap.Logger
}

// NewTraceService creates a new trace service
func NewTraceService(traceRepo *clickhouse.TraceRepository, logger *zap.Logger) *TraceService {
	return &TraceService{
		traceRepo: traceRepo,
		logger:    logger,
	}
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
	return s.traceRepo.Insert(ctx, []*domain.Trace{trace})
}

// IngestBatch ingests multiple traces
func (s *TraceService) IngestBatch(ctx context.Context, traces []*domain.Trace) error {
	return s.traceRepo.Insert(ctx, traces)
}

// IngestSpan ingests a single span
func (s *TraceService) IngestSpan(ctx context.Context, span *domain.Span) error {
	return s.traceRepo.InsertSpan(ctx, span)
}

// SubmitScore submits an evaluation score
func (s *TraceService) SubmitScore(ctx context.Context, score *domain.Score) error {
	return s.traceRepo.InsertScore(ctx, score)
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
