package service

import (
	"context"
	"sort"
	"time"

	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// SessionReplayService handles session replay functionality
type SessionReplayService struct {
	traceRepo *clickhouse.TraceRepository
	logger    *zap.Logger
}

// NewSessionReplayService creates a new session replay service
func NewSessionReplayService(traceRepo *clickhouse.TraceRepository, logger *zap.Logger) *SessionReplayService {
	return &SessionReplayService{
		traceRepo: traceRepo,
		logger:    logger,
	}
}

// ReplayStep represents a single step in the session replay
type ReplayStep struct {
	Index            int              `json:"index"`
	Trace            *domain.Trace    `json:"trace"`
	SpanTree         *domain.SpanTree `json:"spanTree,omitempty"`
	TimeSinceStart   time.Duration    `json:"timeSinceStart"`
	DeltaFromPrev    time.Duration    `json:"deltaFromPrev"`
	CumulativeCost   float64          `json:"cumulativeCost"`
	CumulativeTokens uint32           `json:"cumulativeTokens"`
}

// SessionReplay represents the complete replay data for a session
type SessionReplay struct {
	SessionID     string        `json:"sessionId"`
	UserID        string        `json:"userId,omitempty"`
	TotalSteps    int           `json:"totalSteps"`
	TotalDuration time.Duration `json:"totalDuration"`
	TotalCost     float64       `json:"totalCost"`
	TotalTokens   uint32        `json:"totalTokens"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`
	Steps         []*ReplayStep `json:"steps"`
	Models        []string      `json:"models"`
}

// GetSessionReplay returns the complete replay data for a session
func (s *SessionReplayService) GetSessionReplay(ctx context.Context, sessionID string) (*SessionReplay, error) {
	// Get all traces for this session
	traces, _, err := s.traceRepo.Query(ctx, &clickhouse.QueryOptions{
		SessionID: sessionID,
		SortBy:    "start_time",
		SortOrder: "ASC",
		Limit:     1000, // Cap at 1000 traces per session
		Offset:    0,
	})
	if err != nil {
		return nil, err
	}

	if len(traces) == 0 {
		return &SessionReplay{
			SessionID: sessionID,
			Steps:     []*ReplayStep{},
		}, nil
	}

	// Sort by start time
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].StartTime.Before(traces[j].StartTime)
	})

	replay := &SessionReplay{
		SessionID:  sessionID,
		StartTime:  traces[0].StartTime,
		EndTime:    traces[len(traces)-1].EndTime,
		TotalSteps: len(traces),
		Steps:      make([]*ReplayStep, len(traces)),
	}

	if traces[0].UserID != nil {
		replay.UserID = *traces[0].UserID
	}

	// Build replay steps
	var prevEndTime time.Time
	var cumulativeCost float64
	var cumulativeTokens uint32
	modelSet := make(map[string]bool)

	for i, trace := range traces {
		// Get spans for this trace
		spans, err := s.traceRepo.GetSpans(ctx, trace.ID.String())
		if err != nil {
			s.logger.Warn("failed to get spans for trace", zap.Error(err), zap.String("trace_id", trace.ID.String()))
		}

		var spanTree *domain.SpanTree
		if len(spans) > 0 {
			spanTree = domain.BuildSpanTree(spans)
		}

		// Calculate timing
		var deltaFromPrev time.Duration
		if i > 0 {
			deltaFromPrev = trace.StartTime.Sub(prevEndTime)
			if deltaFromPrev < 0 {
				deltaFromPrev = 0
			}
		}

		timeSinceStart := trace.StartTime.Sub(replay.StartTime)
		cumulativeCost += trace.Cost
		cumulativeTokens += trace.TotalTokens

		// Track models
		if trace.Model != "" {
			modelSet[trace.Model] = true
		}

		replay.Steps[i] = &ReplayStep{
			Index:            i,
			Trace:            trace,
			SpanTree:         spanTree,
			TimeSinceStart:   timeSinceStart,
			DeltaFromPrev:    deltaFromPrev,
			CumulativeCost:   cumulativeCost,
			CumulativeTokens: cumulativeTokens,
		}

		prevEndTime = trace.EndTime
	}

	// Calculate totals
	replay.TotalDuration = replay.EndTime.Sub(replay.StartTime)
	replay.TotalCost = cumulativeCost
	replay.TotalTokens = cumulativeTokens

	for model := range modelSet {
		replay.Models = append(replay.Models, model)
	}

	return replay, nil
}

// GetReplayStep returns a specific step in the session replay
func (s *SessionReplayService) GetReplayStep(ctx context.Context, sessionID string, stepIndex int) (*ReplayStep, error) {
	replay, err := s.GetSessionReplay(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if stepIndex < 0 || stepIndex >= len(replay.Steps) {
		return nil, nil
	}

	return replay.Steps[stepIndex], nil
}

// ReplayTimeline represents the session timeline for visualization
type ReplayTimeline struct {
	SessionID     string           `json:"sessionId"`
	TotalDuration time.Duration    `json:"totalDuration"`
	Events        []*TimelineEvent `json:"events"`
}

// TimelineEvent represents a single event on the timeline
type TimelineEvent struct {
	Index         int           `json:"index"`
	TraceID       string        `json:"traceId"`
	Name          string        `json:"name"`
	StartOffset   time.Duration `json:"startOffset"`
	Duration      time.Duration `json:"duration"`
	WidthPercent  float64       `json:"widthPercent"`
	OffsetPercent float64       `json:"offsetPercent"`
	Status        string        `json:"status"`
	Model         string        `json:"model,omitempty"`
	Tokens        uint32        `json:"tokens"`
	Cost          float64       `json:"cost"`
}

// GetReplayTimeline returns the timeline visualization data
func (s *SessionReplayService) GetReplayTimeline(ctx context.Context, sessionID string) (*ReplayTimeline, error) {
	replay, err := s.GetSessionReplay(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(replay.Steps) == 0 {
		return &ReplayTimeline{
			SessionID: sessionID,
			Events:    []*TimelineEvent{},
		}, nil
	}

	timeline := &ReplayTimeline{
		SessionID:     sessionID,
		TotalDuration: replay.TotalDuration,
		Events:        make([]*TimelineEvent, len(replay.Steps)),
	}

	totalMs := float64(replay.TotalDuration.Milliseconds())
	if totalMs == 0 {
		totalMs = 1 // Avoid division by zero
	}

	for i, step := range replay.Steps {
		duration := step.Trace.EndTime.Sub(step.Trace.StartTime)

		timeline.Events[i] = &TimelineEvent{
			Index:         i,
			TraceID:       step.Trace.ID.String(),
			Name:          step.Trace.Name,
			StartOffset:   step.TimeSinceStart,
			Duration:      duration,
			WidthPercent:  float64(duration.Milliseconds()) / totalMs * 100,
			OffsetPercent: float64(step.TimeSinceStart.Milliseconds()) / totalMs * 100,
			Status:        step.Trace.Status,
			Model:         step.Trace.Model,
			Tokens:        step.Trace.TotalTokens,
			Cost:          step.Trace.Cost,
		}
	}

	return timeline, nil
}
