package clickhouse

import (
	"context"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/otelguard/otelguard/internal/domain"
	"go.uber.org/zap"
)

// BatchWriterConfig contains configuration for the batch writer
type BatchWriterConfig struct {
	BatchSize     int           `envconfig:"CLICKHOUSE_BATCH_SIZE" default:"1000"`
	FlushInterval time.Duration `envconfig:"CLICKHOUSE_FLUSH_INTERVAL" default:"5s"`
	MaxRetries    int           `envconfig:"CLICKHOUSE_MAX_RETRIES" default:"3"`
	RetryDelay    time.Duration `envconfig:"CLICKHOUSE_RETRY_DELAY" default:"1s"`
}

// DefaultBatchWriterConfig returns default configuration
func DefaultBatchWriterConfig() *BatchWriterConfig {
	return &BatchWriterConfig{
		BatchSize:     1000,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		RetryDelay:    time.Second,
	}
}

// TraceBatchWriter handles async batched writes to ClickHouse
type TraceBatchWriter struct {
	conn   driver.Conn
	config *BatchWriterConfig
	logger *zap.Logger

	traceBuffer []*domain.Trace
	spanBuffer  []*domain.Span
	scoreBuffer []*domain.Score

	traceMu sync.Mutex
	spanMu  sync.Mutex
	scoreMu sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup

	// Metrics
	tracesWritten int64
	spansWritten  int64
	scoresWritten int64
	flushCount    int64
	errorCount    int64
	metricsMu     sync.RWMutex
}

// NewTraceBatchWriter creates a new batch writer
func NewTraceBatchWriter(conn driver.Conn, config *BatchWriterConfig, logger *zap.Logger) *TraceBatchWriter {
	if config == nil {
		config = DefaultBatchWriterConfig()
	}

	return &TraceBatchWriter{
		conn:        conn,
		config:      config,
		logger:      logger,
		traceBuffer: make([]*domain.Trace, 0, config.BatchSize),
		spanBuffer:  make([]*domain.Span, 0, config.BatchSize),
		scoreBuffer: make([]*domain.Score, 0, config.BatchSize),
		stopCh:      make(chan struct{}),
	}
}

// Start begins the background flush goroutine
func (w *TraceBatchWriter) Start() {
	w.wg.Add(1)
	go w.flushLoop()
	w.logger.Info("batch writer started",
		zap.Int("batch_size", w.config.BatchSize),
		zap.Duration("flush_interval", w.config.FlushInterval),
	)
}

// Stop gracefully stops the batch writer, flushing any remaining data
func (w *TraceBatchWriter) Stop(ctx context.Context) error {
	close(w.stopCh)

	// Wait for flush loop to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Flush any remaining data
		if err := w.FlushAll(ctx); err != nil {
			w.logger.Error("failed to flush remaining data on shutdown", zap.Error(err))
			return err
		}
		w.logger.Info("batch writer stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// flushLoop runs periodically to flush buffered data
func (w *TraceBatchWriter) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := w.FlushAll(ctx); err != nil {
				w.logger.Error("periodic flush failed", zap.Error(err))
			}
			cancel()
		case <-w.stopCh:
			return
		}
	}
}

// WriteTrace adds a trace to the buffer
func (w *TraceBatchWriter) WriteTrace(ctx context.Context, trace *domain.Trace) error {
	w.traceMu.Lock()
	w.traceBuffer = append(w.traceBuffer, trace)
	shouldFlush := len(w.traceBuffer) >= w.config.BatchSize
	w.traceMu.Unlock()

	if shouldFlush {
		return w.FlushTraces(ctx)
	}
	return nil
}

// WriteTraces adds multiple traces to the buffer
func (w *TraceBatchWriter) WriteTraces(ctx context.Context, traces []*domain.Trace) error {
	w.traceMu.Lock()
	w.traceBuffer = append(w.traceBuffer, traces...)
	shouldFlush := len(w.traceBuffer) >= w.config.BatchSize
	w.traceMu.Unlock()

	if shouldFlush {
		return w.FlushTraces(ctx)
	}
	return nil
}

// WriteSpan adds a span to the buffer
func (w *TraceBatchWriter) WriteSpan(ctx context.Context, span *domain.Span) error {
	w.spanMu.Lock()
	w.spanBuffer = append(w.spanBuffer, span)
	shouldFlush := len(w.spanBuffer) >= w.config.BatchSize
	w.spanMu.Unlock()

	if shouldFlush {
		return w.FlushSpans(ctx)
	}
	return nil
}

// WriteScore adds a score to the buffer
func (w *TraceBatchWriter) WriteScore(ctx context.Context, score *domain.Score) error {
	w.scoreMu.Lock()
	w.scoreBuffer = append(w.scoreBuffer, score)
	shouldFlush := len(w.scoreBuffer) >= w.config.BatchSize
	w.scoreMu.Unlock()

	if shouldFlush {
		return w.FlushScores(ctx)
	}
	return nil
}

// FlushAll flushes all buffers
func (w *TraceBatchWriter) FlushAll(ctx context.Context) error {
	var lastErr error

	if err := w.FlushTraces(ctx); err != nil {
		lastErr = err
	}
	if err := w.FlushSpans(ctx); err != nil {
		lastErr = err
	}
	if err := w.FlushScores(ctx); err != nil {
		lastErr = err
	}

	return lastErr
}

// FlushTraces flushes the trace buffer to ClickHouse
func (w *TraceBatchWriter) FlushTraces(ctx context.Context) error {
	w.traceMu.Lock()
	if len(w.traceBuffer) == 0 {
		w.traceMu.Unlock()
		return nil
	}

	// Swap out the buffer
	traces := w.traceBuffer
	w.traceBuffer = make([]*domain.Trace, 0, w.config.BatchSize)
	w.traceMu.Unlock()

	// Write with retries
	err := w.writeTracesWithRetry(ctx, traces)
	if err != nil {
		w.incrementErrorCount()
		// Re-add traces to buffer on failure
		w.traceMu.Lock()
		w.traceBuffer = append(traces, w.traceBuffer...)
		w.traceMu.Unlock()
		return err
	}

	w.incrementTracesWritten(int64(len(traces)))
	w.incrementFlushCount()

	w.logger.Debug("flushed traces",
		zap.Int("count", len(traces)),
	)

	return nil
}

// FlushSpans flushes the span buffer to ClickHouse
func (w *TraceBatchWriter) FlushSpans(ctx context.Context) error {
	w.spanMu.Lock()
	if len(w.spanBuffer) == 0 {
		w.spanMu.Unlock()
		return nil
	}

	// Swap out the buffer
	spans := w.spanBuffer
	w.spanBuffer = make([]*domain.Span, 0, w.config.BatchSize)
	w.spanMu.Unlock()

	// Write with retries
	err := w.writeSpansWithRetry(ctx, spans)
	if err != nil {
		w.incrementErrorCount()
		// Re-add spans to buffer on failure
		w.spanMu.Lock()
		w.spanBuffer = append(spans, w.spanBuffer...)
		w.spanMu.Unlock()
		return err
	}

	w.incrementSpansWritten(int64(len(spans)))
	w.incrementFlushCount()

	w.logger.Debug("flushed spans",
		zap.Int("count", len(spans)),
	)

	return nil
}

// FlushScores flushes the score buffer to ClickHouse
func (w *TraceBatchWriter) FlushScores(ctx context.Context) error {
	w.scoreMu.Lock()
	if len(w.scoreBuffer) == 0 {
		w.scoreMu.Unlock()
		return nil
	}

	// Swap out the buffer
	scores := w.scoreBuffer
	w.scoreBuffer = make([]*domain.Score, 0, w.config.BatchSize)
	w.scoreMu.Unlock()

	// Write with retries
	err := w.writeScoresWithRetry(ctx, scores)
	if err != nil {
		w.incrementErrorCount()
		// Re-add scores to buffer on failure
		w.scoreMu.Lock()
		w.scoreBuffer = append(scores, w.scoreBuffer...)
		w.scoreMu.Unlock()
		return err
	}

	w.incrementScoresWritten(int64(len(scores)))
	w.incrementFlushCount()

	w.logger.Debug("flushed scores",
		zap.Int("count", len(scores)),
	)

	return nil
}

// writeTracesWithRetry writes traces with retry logic
func (w *TraceBatchWriter) writeTracesWithRetry(ctx context.Context, traces []*domain.Trace) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryDelay * time.Duration(attempt)):
			}
			w.logger.Debug("retrying trace write",
				zap.Int("attempt", attempt),
				zap.Int("count", len(traces)),
			)
		}

		batch, err := w.conn.PrepareBatch(ctx, `
			INSERT INTO traces (
				id, project_id, session_id, user_id, name,
				input, output, metadata, start_time, end_time,
				latency_ms, total_tokens, prompt_tokens, completion_tokens,
				cost, model, tags, status, error_message
			)
		`)
		if err != nil {
			lastErr = err
			continue
		}

		for _, trace := range traces {
			sessionID := ""
			if trace.SessionID != nil {
				sessionID = *trace.SessionID
			}
			userID := ""
			if trace.UserID != nil {
				userID = *trace.UserID
			}
			errorMsg := ""
			if trace.ErrorMessage != nil {
				errorMsg = *trace.ErrorMessage
			}

			err := batch.Append(
				trace.ID,
				trace.ProjectID,
				sessionID,
				userID,
				trace.Name,
				trace.Input,
				trace.Output,
				trace.Metadata,
				trace.StartTime,
				trace.EndTime,
				trace.LatencyMs,
				trace.TotalTokens,
				trace.PromptTokens,
				trace.CompletionTokens,
				trace.Cost,
				trace.Model,
				trace.Tags,
				trace.Status,
				errorMsg,
			)
			if err != nil {
				lastErr = err
				break
			}
		}

		if lastErr != nil {
			continue
		}

		if err := batch.Send(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}

// writeSpansWithRetry writes spans with retry logic
func (w *TraceBatchWriter) writeSpansWithRetry(ctx context.Context, spans []*domain.Span) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryDelay * time.Duration(attempt)):
			}
		}

		batch, err := w.conn.PrepareBatch(ctx, `
			INSERT INTO spans (
				id, trace_id, parent_span_id, project_id, name, type,
				input, output, metadata, start_time, end_time,
				latency_ms, tokens, cost, model, status, error_message
			)
		`)
		if err != nil {
			lastErr = err
			continue
		}

		for _, span := range spans {
			parentSpanID := ""
			if span.ParentSpanID != nil {
				parentSpanID = span.ParentSpanID.String()
			}
			model := ""
			if span.Model != nil {
				model = *span.Model
			}
			errorMsg := ""
			if span.ErrorMessage != nil {
				errorMsg = *span.ErrorMessage
			}

			err := batch.Append(
				span.ID,
				span.TraceID,
				parentSpanID,
				span.ProjectID,
				span.Name,
				span.Type,
				span.Input,
				span.Output,
				span.Metadata,
				span.StartTime,
				span.EndTime,
				span.LatencyMs,
				span.Tokens,
				span.Cost,
				model,
				span.Status,
				errorMsg,
			)
			if err != nil {
				lastErr = err
				break
			}
		}

		if lastErr != nil {
			continue
		}

		if err := batch.Send(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}

// writeScoresWithRetry writes scores with retry logic
func (w *TraceBatchWriter) writeScoresWithRetry(ctx context.Context, scores []*domain.Score) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryDelay * time.Duration(attempt)):
			}
		}

		batch, err := w.conn.PrepareBatch(ctx, `
			INSERT INTO scores (
				id, project_id, trace_id, span_id, name, value,
				string_value, data_type, source, config_id, comment, created_at
			)
		`)
		if err != nil {
			lastErr = err
			continue
		}

		for _, score := range scores {
			spanID := ""
			if score.SpanID != nil {
				spanID = score.SpanID.String()
			}
			stringValue := ""
			if score.StringValue != nil {
				stringValue = *score.StringValue
			}
			configID := ""
			if score.ConfigID != nil {
				configID = score.ConfigID.String()
			}
			comment := ""
			if score.Comment != nil {
				comment = *score.Comment
			}

			err := batch.Append(
				score.ID,
				score.ProjectID,
				score.TraceID,
				spanID,
				score.Name,
				score.Value,
				stringValue,
				score.DataType,
				score.Source,
				configID,
				comment,
				score.CreatedAt,
			)
			if err != nil {
				lastErr = err
				break
			}
		}

		if lastErr != nil {
			continue
		}

		if err := batch.Send(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}

// Metrics helpers
func (w *TraceBatchWriter) incrementTracesWritten(n int64) {
	w.metricsMu.Lock()
	w.tracesWritten += n
	w.metricsMu.Unlock()
}

func (w *TraceBatchWriter) incrementSpansWritten(n int64) {
	w.metricsMu.Lock()
	w.spansWritten += n
	w.metricsMu.Unlock()
}

func (w *TraceBatchWriter) incrementScoresWritten(n int64) {
	w.metricsMu.Lock()
	w.scoresWritten += n
	w.metricsMu.Unlock()
}

func (w *TraceBatchWriter) incrementFlushCount() {
	w.metricsMu.Lock()
	w.flushCount++
	w.metricsMu.Unlock()
}

func (w *TraceBatchWriter) incrementErrorCount() {
	w.metricsMu.Lock()
	w.errorCount++
	w.metricsMu.Unlock()
}

// BatchWriterMetrics contains metrics about the batch writer
type BatchWriterMetrics struct {
	TracesWritten int64 `json:"tracesWritten"`
	SpansWritten  int64 `json:"spansWritten"`
	ScoresWritten int64 `json:"scoresWritten"`
	FlushCount    int64 `json:"flushCount"`
	ErrorCount    int64 `json:"errorCount"`
	TraceBuffer   int   `json:"traceBuffer"`
	SpanBuffer    int   `json:"spanBuffer"`
	ScoreBuffer   int   `json:"scoreBuffer"`
}

// GetMetrics returns current metrics
func (w *TraceBatchWriter) GetMetrics() *BatchWriterMetrics {
	w.metricsMu.RLock()
	defer w.metricsMu.RUnlock()

	w.traceMu.Lock()
	traceBuffer := len(w.traceBuffer)
	w.traceMu.Unlock()

	w.spanMu.Lock()
	spanBuffer := len(w.spanBuffer)
	w.spanMu.Unlock()

	w.scoreMu.Lock()
	scoreBuffer := len(w.scoreBuffer)
	w.scoreMu.Unlock()

	return &BatchWriterMetrics{
		TracesWritten: w.tracesWritten,
		SpansWritten:  w.spansWritten,
		ScoresWritten: w.scoresWritten,
		FlushCount:    w.flushCount,
		ErrorCount:    w.errorCount,
		TraceBuffer:   traceBuffer,
		SpanBuffer:    spanBuffer,
		ScoreBuffer:   scoreBuffer,
	}
}
