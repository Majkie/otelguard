package clickhouse

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/otelguard/otelguard/internal/domain"
)

// TraceRepository handles trace data access in ClickHouse
type TraceRepository struct {
	conn driver.Conn
}

// NewTraceRepository creates a new trace repository
func NewTraceRepository(conn driver.Conn) *TraceRepository {
	return &TraceRepository{conn: conn}
}

// QueryOptions contains options for querying traces
type QueryOptions struct {
	ProjectID string
	SessionID string
	UserID    string
	Model     string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// Insert inserts traces into ClickHouse
func (r *TraceRepository) Insert(ctx context.Context, traces []*domain.Trace) error {
	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO traces (
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			cost, model, tags, status, error_message
		)
	`)
	if err != nil {
		return err
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
			return err
		}
	}

	return batch.Send()
}

// InsertSpan inserts a span into ClickHouse
func (r *TraceRepository) InsertSpan(ctx context.Context, span *domain.Span) error {
	query := `
		INSERT INTO spans (
			id, trace_id, parent_span_id, project_id, name, type,
			input, output, metadata, start_time, end_time,
			latency_ms, tokens, cost, model, status, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

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

	return r.conn.Exec(ctx, query,
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
}

// InsertScore inserts a score into ClickHouse
func (r *TraceRepository) InsertScore(ctx context.Context, score *domain.Score) error {
	query := `
		INSERT INTO scores (
			id, project_id, trace_id, span_id, name, value,
			string_value, data_type, source, config_id, comment, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

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

	return r.conn.Exec(ctx, query,
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
}

// Query retrieves traces with filtering
func (r *TraceRepository) Query(ctx context.Context, opts *QueryOptions) ([]*domain.Trace, int, error) {
	// Build the query based on options
	query := `
		SELECT
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			cost, model, tags, status, error_message
		FROM traces
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM traces WHERE 1=1`
	args := make([]interface{}, 0)

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		countQuery += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if opts.SessionID != "" {
		query += " AND session_id = ?"
		countQuery += " AND session_id = ?"
		args = append(args, opts.SessionID)
	}
	if opts.UserID != "" {
		query += " AND user_id = ?"
		countQuery += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.Model != "" {
		query += " AND model = ?"
		countQuery += " AND model = ?"
		args = append(args, opts.Model)
	}

	// Get total count
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Add pagination
	query += " ORDER BY start_time DESC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var traces []*domain.Trace
	for rows.Next() {
		var t domain.Trace
		var sessionID, userID, errorMsg string
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &sessionID, &userID, &t.Name,
			&t.Input, &t.Output, &t.Metadata, &t.StartTime, &t.EndTime,
			&t.LatencyMs, &t.TotalTokens, &t.PromptTokens, &t.CompletionTokens,
			&t.Cost, &t.Model, &t.Tags, &t.Status, &errorMsg,
		); err != nil {
			return nil, 0, err
		}
		if sessionID != "" {
			t.SessionID = &sessionID
		}
		if userID != "" {
			t.UserID = &userID
		}
		if errorMsg != "" {
			t.ErrorMessage = &errorMsg
		}
		traces = append(traces, &t)
	}

	return traces, int(total), nil
}

// GetByID retrieves a trace by ID
func (r *TraceRepository) GetByID(ctx context.Context, id string) (*domain.Trace, error) {
	query := `
		SELECT
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			cost, model, tags, status, error_message
		FROM traces
		WHERE id = ?
	`

	var t domain.Trace
	var sessionID, userID, errorMsg string
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.ProjectID, &sessionID, &userID, &t.Name,
		&t.Input, &t.Output, &t.Metadata, &t.StartTime, &t.EndTime,
		&t.LatencyMs, &t.TotalTokens, &t.PromptTokens, &t.CompletionTokens,
		&t.Cost, &t.Model, &t.Tags, &t.Status, &errorMsg,
	)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		t.SessionID = &sessionID
	}
	if userID != "" {
		t.UserID = &userID
	}
	if errorMsg != "" {
		t.ErrorMessage = &errorMsg
	}

	return &t, nil
}

// GetSpans retrieves spans for a trace
func (r *TraceRepository) GetSpans(ctx context.Context, traceID string) ([]*domain.Span, error) {
	query := `
		SELECT
			id, trace_id, parent_span_id, project_id, name, type,
			input, output, metadata, start_time, end_time,
			latency_ms, tokens, cost, model, status, error_message
		FROM spans
		WHERE trace_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.conn.Query(ctx, query, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*domain.Span
	for rows.Next() {
		var s domain.Span
		var parentSpanID, model, errorMsg string
		if err := rows.Scan(
			&s.ID, &s.TraceID, &parentSpanID, &s.ProjectID, &s.Name, &s.Type,
			&s.Input, &s.Output, &s.Metadata, &s.StartTime, &s.EndTime,
			&s.LatencyMs, &s.Tokens, &s.Cost, &model, &s.Status, &errorMsg,
		); err != nil {
			return nil, err
		}
		if model != "" {
			s.Model = &model
		}
		if errorMsg != "" {
			s.ErrorMessage = &errorMsg
		}
		spans = append(spans, &s)
	}

	return spans, nil
}
