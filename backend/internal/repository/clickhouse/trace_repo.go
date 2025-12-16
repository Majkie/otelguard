package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
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
	ProjectID     string
	SessionID     string
	UserID        string
	Model         string
	Name          string   // Search by name (partial match)
	Status        string   // Filter by status (success, error, pending)
	Tags          []string // Filter by tags (any match)
	StartTime     string   // ISO8601 timestamp
	EndTime       string   // ISO8601 timestamp
	MinLatency    int      // Minimum latency in ms
	MaxLatency    int      // Maximum latency in ms
	MinCost       float64  // Minimum cost
	MaxCost       float64  // Maximum cost
	PromptID      string   // Filter by prompt ID
	PromptVersion string   // Filter by prompt version
	SortBy        string   // Field to sort by (start_time, latency_ms, cost, total_tokens)
	SortOrder     string   // ASC or DESC
	Limit         int
	Offset        int
}

// PromptPerformanceMetrics represents aggregated performance metrics for a prompt version
type PromptPerformanceMetrics struct {
	PromptID      string  `json:"promptId"`
	PromptVersion int     `json:"promptVersion"`
	Date          string  `json:"date"`
	Model         string  `json:"model"`
	TraceCount    uint64  `json:"traceCount"`
	TotalLatency  uint64  `json:"totalLatency"`
	AvgLatency    float64 `json:"avgLatency"`
	TotalTokens   uint64  `json:"totalTokens"`
	AvgTokens     float64 `json:"avgTokens"`
	TotalCost     float64 `json:"totalCost"`
	AvgCost       float64 `json:"avgCost"`
	ErrorCount    uint64  `json:"errorCount"`
}

// Insert inserts traces into ClickHouse
func (r *TraceRepository) Insert(ctx context.Context, traces []*domain.Trace) error {
	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO traces (
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			cost, model, tags, status, error_message,
			prompt_id, prompt_version
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
		promptID := ""
		if trace.PromptID != nil {
			promptID = trace.PromptID.String()
		}
		promptVersion := ""
		if trace.PromptVersion != nil {
			promptVersion = fmt.Sprintf("%d", *trace.PromptVersion)
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
			promptID,
			promptVersion,
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
			id, trace_id, parent_span_id, project_id, name, span_type,
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
	// Test basic connectivity first
	var testCount uint64
	if err := r.conn.QueryRow(ctx, "SELECT COUNT(*) FROM traces").Scan(&testCount); err != nil {
		return nil, 0, fmt.Errorf("basic connectivity test failed: %w", err)
	}

	// Build the query based on options
	baseQuery := `
		SELECT
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			toFloat64(cost) as cost, model, tags, status, error_message,
			prompt_id, prompt_version
		FROM traces
		WHERE 1=1
	`

	countQuery := `SELECT COUNT(*) FROM traces WHERE 1=1`
	args := make([]interface{}, 0)
	countArgs := make([]interface{}, 0)

	// Build filter clauses
	filterClause := ""

	if opts.ProjectID != "" {
		filterClause += " AND project_id = ?"
		args = append(args, opts.ProjectID)
		countArgs = append(countArgs, opts.ProjectID)
	}
	if opts.SessionID != "" {
		filterClause += " AND session_id IS NOT NULL AND session_id = ?"
		args = append(args, opts.SessionID)
		countArgs = append(countArgs, opts.SessionID)
	}
	if opts.UserID != "" {
		filterClause += " AND user_id IS NOT NULL AND user_id = ?"
		args = append(args, opts.UserID)
		countArgs = append(countArgs, opts.UserID)
	}
	if opts.Model != "" {
		filterClause += " AND model = ?"
		args = append(args, opts.Model)
		countArgs = append(countArgs, opts.Model)
	}
	if opts.Name != "" {
		filterClause += " AND name LIKE ?"
		args = append(args, "%"+opts.Name+"%")
		countArgs = append(countArgs, "%"+opts.Name+"%")
	}
	if opts.Status != "" {
		filterClause += " AND status = ?"
		args = append(args, opts.Status)
		countArgs = append(countArgs, opts.Status)
	}
	if len(opts.Tags) > 0 {
		// For now, skip tag filtering to test if that's the issue
		// filterClause += " AND hasAny(tags, ?)"
		// args = append(args, opts.Tags)
		// countArgs = append(countArgs, opts.Tags)
	}
	if opts.StartTime != "" {
		filterClause += " AND start_time >= ?"
		args = append(args, opts.StartTime)
		countArgs = append(countArgs, opts.StartTime)
	}
	if opts.EndTime != "" {
		filterClause += " AND start_time <= ?"
		args = append(args, opts.EndTime)
		countArgs = append(countArgs, opts.EndTime)
	}
	if opts.MinLatency > 0 {
		filterClause += " AND latency_ms >= ?"
		args = append(args, opts.MinLatency)
		countArgs = append(countArgs, opts.MinLatency)
	}
	if opts.MaxLatency > 0 {
		filterClause += " AND latency_ms <= ?"
		args = append(args, opts.MaxLatency)
		countArgs = append(countArgs, opts.MaxLatency)
	}
	if opts.MinCost > 0 {
		filterClause += " AND cost >= ?"
		args = append(args, opts.MinCost)
		countArgs = append(countArgs, opts.MinCost)
	}
	if opts.MaxCost > 0 {
		filterClause += " AND cost <= ?"
		args = append(args, opts.MaxCost)
		countArgs = append(countArgs, opts.MaxCost)
	}
	if opts.PromptID != "" {
		filterClause += " AND prompt_id = ?"
		args = append(args, opts.PromptID)
		countArgs = append(countArgs, opts.PromptID)
	}
	if opts.PromptVersion != "" {
		filterClause += " AND prompt_version = ?"
		args = append(args, opts.PromptVersion)
		countArgs = append(countArgs, opts.PromptVersion)
	}

	// Get total count
	var total uint64
	countSQL := countQuery + filterClause
	if err := r.conn.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %s with args %v: %w", countSQL, countArgs, err)
	}

	// Determine sort order
	sortBy := "start_time"
	if opts.SortBy != "" {
		// Validate sort field to prevent SQL injection
		validSortFields := map[string]bool{
			"start_time":     true,
			"latency_ms":     true,
			"cost":           true,
			"total_tokens":   true,
			"name":           true,
			"model":          true,
			"prompt_id":      true,
			"prompt_version": true,
		}
		if validSortFields[opts.SortBy] {
			sortBy = opts.SortBy
		}
	}

	sortOrder := "DESC"
	if opts.SortOrder == "ASC" || opts.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Add sorting and pagination
	query := baseQuery + filterClause + " ORDER BY " + sortBy + " " + sortOrder + " LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var traces []*domain.Trace
	for rows.Next() {
		var t domain.Trace
		var sessionID, userID, errorMsg, promptIDStr, promptVersionStr *string
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &sessionID, &userID, &t.Name,
			&t.Input, &t.Output, &t.Metadata, &t.StartTime, &t.EndTime,
			&t.LatencyMs, &t.TotalTokens, &t.PromptTokens, &t.CompletionTokens,
			&t.Cost, &t.Model, &t.Tags, &t.Status, &errorMsg,
			&promptIDStr, &promptVersionStr,
		); err != nil {
			return nil, 0, err
		}
		t.SessionID = sessionID
		t.UserID = userID
		t.ErrorMessage = errorMsg

		// Parse prompt fields
		if promptIDStr != nil && *promptIDStr != "" {
			if promptID, err := uuid.Parse(*promptIDStr); err == nil {
				t.PromptID = &promptID
			}
		}
		if promptVersionStr != nil && *promptVersionStr != "" {
			if promptVersion, err := strconv.Atoi(*promptVersionStr); err == nil {
				t.PromptVersion = &promptVersion
			}
		}

		traces = append(traces, &t)
	}

	return traces, int(total), nil
}

// GetPromptPerformanceMetrics retrieves performance metrics for prompts
func (r *TraceRepository) GetPromptPerformanceMetrics(ctx context.Context, projectID string, promptID string, startTime, endTime string) ([]*PromptPerformanceMetrics, error) {
	query := `
		SELECT
			prompt_id,
			prompt_version,
			date,
			model,
			trace_count,
			total_latency_ms,
			avg_latency_ms,
			total_tokens,
			avg_tokens,
			total_cost,
			avg_cost,
			error_count
		FROM prompt_daily_stats
		WHERE project_id = ?
	`

	args := []interface{}{projectID}

	if promptID != "" {
		query += " AND prompt_id = ?"
		args = append(args, promptID)
	}

	if startTime != "" {
		query += " AND date >= ?"
		args = append(args, startTime)
	}

	if endTime != "" {
		query += " AND date <= ?"
		args = append(args, endTime)
	}

	query += " ORDER BY date DESC, prompt_version DESC"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*PromptPerformanceMetrics
	for rows.Next() {
		var m PromptPerformanceMetrics
		err := rows.Scan(
			&m.PromptID,
			&m.PromptVersion,
			&m.Date,
			&m.Model,
			&m.TraceCount,
			&m.TotalLatency,
			&m.AvgLatency,
			&m.TotalTokens,
			&m.AvgTokens,
			&m.TotalCost,
			&m.AvgCost,
			&m.ErrorCount,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, &m)
	}

	return metrics, nil
}

// GetByID retrieves a trace by ID
func (r *TraceRepository) GetByID(ctx context.Context, id string) (*domain.Trace, error) {
	query := `
		SELECT
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			toFloat64(cost) as cost, model, tags, status, error_message,
			prompt_id, prompt_version
		FROM traces
		WHERE id = ?
	`

	var t domain.Trace
	var sessionID, userID, errorMsg, promptIDStr, promptVersionStr *string
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.ProjectID, &sessionID, &userID, &t.Name,
		&t.Input, &t.Output, &t.Metadata, &t.StartTime, &t.EndTime,
		&t.LatencyMs, &t.TotalTokens, &t.PromptTokens, &t.CompletionTokens,
		&t.Cost, &t.Model, &t.Tags, &t.Status, &errorMsg,
		&promptIDStr, &promptVersionStr,
	)
	if err != nil {
		return nil, err
	}

	t.SessionID = sessionID
	t.UserID = userID
	t.ErrorMessage = errorMsg

	// Parse prompt fields
	if promptIDStr != nil && *promptIDStr != "" {
		if promptID, err := uuid.Parse(*promptIDStr); err == nil {
			t.PromptID = &promptID
		}
	}
	if promptVersionStr != nil && *promptVersionStr != "" {
		if promptVersion, err := strconv.Atoi(*promptVersionStr); err == nil {
			t.PromptVersion = &promptVersion
		}
	}

	return &t, nil
}

// Session represents aggregated session data
type Session struct {
	SessionID      string    `json:"sessionId"`
	ProjectID      string    `json:"projectId"`
	UserID         string    `json:"userId,omitempty"`
	TraceCount     uint64    `json:"traceCount"`
	TotalLatencyMs uint64    `json:"totalLatencyMs"`
	TotalTokens    uint64    `json:"totalTokens"`
	TotalCost      float64   `json:"totalCost"`
	SuccessCount   uint64    `json:"successCount"`
	ErrorCount     uint64    `json:"errorCount"`
	FirstTraceTime time.Time `json:"firstTraceTime"`
	LastTraceTime  time.Time `json:"lastTraceTime"`
	Models         []string  `json:"models,omitempty"`
}

// SessionQueryOptions contains options for querying sessions
type SessionQueryOptions struct {
	ProjectID string
	UserID    string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// ListSessions retrieves sessions with aggregated metrics
func (r *TraceRepository) ListSessions(ctx context.Context, opts *SessionQueryOptions) ([]*Session, int, error) {
	baseQuery := `
		SELECT
			session_id,
			project_id,
			any(user_id) as session_user_id,
			count() as trace_count,
			sum(latency_ms) as total_latency_ms,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			min(start_time) as first_trace_time,
			max(start_time) as last_trace_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE session_id IS NOT NULL AND session_id != ''
	`
	countQuery := `SELECT count(DISTINCT session_id) FROM traces WHERE session_id IS NOT NULL AND session_id != ''`
	args := make([]interface{}, 0)
	countArgs := make([]interface{}, 0)

	filterClause := ""
	if opts.ProjectID != "" {
		filterClause += " AND project_id = ?"
		args = append(args, opts.ProjectID)
		countArgs = append(countArgs, opts.ProjectID)
	}
	if opts.UserID != "" {
		filterClause += " AND user_id = ?"
		args = append(args, opts.UserID)
		countArgs = append(countArgs, opts.UserID)
	}
	if opts.StartTime != "" {
		filterClause += " AND start_time >= ?"
		args = append(args, opts.StartTime)
		countArgs = append(countArgs, opts.StartTime)
	}
	if opts.EndTime != "" {
		filterClause += " AND start_time <= ?"
		args = append(args, opts.EndTime)
		countArgs = append(countArgs, opts.EndTime)
	}

	// Get total count
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery+filterClause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Add grouping, sorting and pagination
	query := baseQuery + filterClause + " GROUP BY session_id, project_id ORDER BY last_trace_time DESC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(
			&s.SessionID, &s.ProjectID, &s.UserID,
			&s.TraceCount, &s.TotalLatencyMs, &s.TotalTokens, &s.TotalCost,
			&s.SuccessCount, &s.ErrorCount,
			&s.FirstTraceTime, &s.LastTraceTime, &s.Models,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, &s)
	}

	return sessions, int(total), nil
}

// GetSessionByID retrieves a session by ID with all its traces
func (r *TraceRepository) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT
			session_id,
			project_id,
			any(user_id) as user_id,
			count() as trace_count,
			sum(latency_ms) as total_latency_ms,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			min(start_time) as first_trace_time,
			max(start_time) as last_trace_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE session_id IS NOT NULL AND session_id = ?
		GROUP BY session_id, project_id
	`

	var s Session
	err := r.conn.QueryRow(ctx, query, sessionID).Scan(
		&s.SessionID, &s.ProjectID, &s.UserID,
		&s.TraceCount, &s.TotalLatencyMs, &s.TotalTokens, &s.TotalCost,
		&s.SuccessCount, &s.ErrorCount,
		&s.FirstTraceTime, &s.LastTraceTime, &s.Models,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetSpans retrieves spans for a trace
func (r *TraceRepository) GetSpans(ctx context.Context, traceID string) ([]*domain.Span, error) {
	query := `
		SELECT
			id, trace_id, parent_span_id, project_id, name, span_type,
			input, output, metadata, start_time, end_time,
			latency_ms, tokens, toFloat64(cost) as cost, model, status, error_message
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
		var parentSpanID, model, errorMsg *string
		if err := rows.Scan(
			&s.ID, &s.TraceID, &parentSpanID, &s.ProjectID, &s.Name, &s.Type,
			&s.Input, &s.Output, &s.Metadata, &s.StartTime, &s.EndTime,
			&s.LatencyMs, &s.Tokens, &s.Cost, &model, &s.Status, &errorMsg,
		); err != nil {
			return nil, err
		}
		if parentSpanID != nil && *parentSpanID != "" {
			if id, err := uuid.Parse(*parentSpanID); err == nil {
				s.ParentSpanID = &id
			}
		}
		s.Model = model
		s.ErrorMessage = errorMsg
		spans = append(spans, &s)
	}

	return spans, nil
}

// User represents aggregated user activity data
type User struct {
	UserID         string   `json:"userId"`
	ProjectID      string   `json:"projectId"`
	TraceCount     int      `json:"traceCount"`
	SessionCount   int      `json:"sessionCount"`
	TotalLatencyMs int      `json:"totalLatencyMs"`
	AvgLatencyMs   float64  `json:"avgLatencyMs"`
	TotalTokens    int      `json:"totalTokens"`
	TotalCost      float64  `json:"totalCost"`
	SuccessCount   int      `json:"successCount"`
	ErrorCount     int      `json:"errorCount"`
	SuccessRate    float64  `json:"successRate"`
	FirstSeenTime  string   `json:"firstSeenTime"`
	LastSeenTime   string   `json:"lastSeenTime"`
	Models         []string `json:"models,omitempty"`
}

// UserQueryOptions contains options for querying users
type UserQueryOptions struct {
	ProjectID string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// ListUsers retrieves users with aggregated metrics
func (r *TraceRepository) ListUsers(ctx context.Context, opts *UserQueryOptions) ([]*User, int, error) {
	baseQuery := `
		SELECT
			user_id,
			project_id,
			count() as trace_count,
			uniqExact(ifNull(session_id, '')) as session_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'success') / count(), 0) as success_rate,
			toString(min(start_time)) as first_seen_time,
			toString(max(start_time)) as last_seen_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE user_id IS NOT NULL AND length(user_id) > 0
	`
	countQuery := `SELECT count(DISTINCT user_id) FROM traces WHERE user_id IS NOT NULL AND length(user_id) > 0`
	args := make([]interface{}, 0)
	countArgs := make([]interface{}, 0)

	filterClause := ""
	if opts.ProjectID != "" {
		filterClause += " AND project_id = ?"
		args = append(args, opts.ProjectID)
		countArgs = append(countArgs, opts.ProjectID)
	}
	if opts.StartTime != "" {
		filterClause += " AND start_time >= ?"
		args = append(args, opts.StartTime)
		countArgs = append(countArgs, opts.StartTime)
	}
	if opts.EndTime != "" {
		filterClause += " AND start_time <= ?"
		args = append(args, opts.EndTime)
		countArgs = append(countArgs, opts.EndTime)
	}

	// Get total count
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery+filterClause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Add grouping, sorting and pagination
	query := baseQuery + filterClause + " GROUP BY user_id, project_id ORDER BY last_seen_time DESC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		var traceCount, sessionCount, totalLatency, totalTokens, successCount, errorCount uint64
		if err := rows.Scan(
			&u.UserID, &u.ProjectID,
			&traceCount, &sessionCount, &totalLatency, &u.AvgLatencyMs,
			&totalTokens, &u.TotalCost,
			&successCount, &errorCount, &u.SuccessRate,
			&u.FirstSeenTime, &u.LastSeenTime, &u.Models,
		); err != nil {
			return nil, 0, err
		}
		u.TraceCount = int(traceCount)
		u.SessionCount = int(sessionCount)
		u.TotalLatencyMs = int(totalLatency)
		u.TotalTokens = int(totalTokens)
		u.SuccessCount = int(successCount)
		u.ErrorCount = int(errorCount)
		users = append(users, &u)
	}

	return users, int(total), nil
}

// GetUserByID retrieves a user by ID with aggregated metrics
func (r *TraceRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT
			user_id,
			project_id,
			count() as trace_count,
			uniqExact(ifNull(session_id, '')) as session_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'success') / count(), 0) as success_rate,
			toString(min(start_time)) as first_seen_time,
			toString(max(start_time)) as last_seen_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE user_id IS NOT NULL AND user_id = ? AND length(user_id) > 0
		GROUP BY user_id, project_id
	`

	var u User
	var traceCount, sessionCount, totalLatency, totalTokens, successCount, errorCount uint64
	err := r.conn.QueryRow(ctx, query, userID).Scan(
		&u.UserID, &u.ProjectID,
		&traceCount, &sessionCount, &totalLatency, &u.AvgLatencyMs,
		&totalTokens, &u.TotalCost,
		&successCount, &errorCount, &u.SuccessRate,
		&u.FirstSeenTime, &u.LastSeenTime, &u.Models,
	)
	if err != nil {
		return nil, err
	}
	u.TraceCount = int(traceCount)
	u.SessionCount = int(sessionCount)
	u.TotalLatencyMs = int(totalLatency)
	u.TotalTokens = int(totalTokens)
	u.SuccessCount = int(successCount)
	u.ErrorCount = int(errorCount)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetUserSessions retrieves all sessions for a user
func (r *TraceRepository) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*Session, int, error) {
	return r.ListSessions(ctx, &SessionQueryOptions{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// SearchTraces performs full-text search on trace content
func (r *TraceRepository) SearchTraces(ctx context.Context, opts *SearchOptions) ([]*domain.Trace, int, error) {
	baseQuery := `
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
	countArgs := make([]interface{}, 0)

	filterClause := ""

	if opts.ProjectID != "" {
		filterClause += " AND project_id = ?"
		args = append(args, opts.ProjectID)
		countArgs = append(countArgs, opts.ProjectID)
	}

	// Full-text search on input, output, and name
	if opts.Query != "" {
		filterClause += " AND (input ILIKE ? OR output ILIKE ? OR name ILIKE ?)"
		pattern := "%" + opts.Query + "%"
		args = append(args, pattern, pattern, pattern)
		countArgs = append(countArgs, pattern, pattern, pattern)
	}

	if opts.StartTime != "" {
		filterClause += " AND start_time >= ?"
		args = append(args, opts.StartTime)
		countArgs = append(countArgs, opts.StartTime)
	}
	if opts.EndTime != "" {
		filterClause += " AND start_time <= ?"
		args = append(args, opts.EndTime)
		countArgs = append(countArgs, opts.EndTime)
	}

	// Get total count
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery+filterClause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Add sorting and pagination
	query := baseQuery + filterClause + " ORDER BY start_time DESC LIMIT ? OFFSET ?"
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

// SearchOptions contains options for searching traces
type SearchOptions struct {
	ProjectID string
	Query     string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}

// OverviewMetrics contains aggregated overview metrics
type OverviewMetrics struct {
	TotalTraces    int     `json:"totalTraces"`
	TotalTokens    int     `json:"totalTokens"`
	TotalCost      float64 `json:"totalCost"`
	AvgLatencyMs   float64 `json:"avgLatencyMs"`
	ErrorRate      float64 `json:"errorRate"`
	SuccessCount   int     `json:"successCount"`
	ErrorCount     int     `json:"errorCount"`
	UniqueUsers    int     `json:"uniqueUsers"`
	UniqueSessions int     `json:"uniqueSessions"`
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Count     int     `json:"count,omitempty"`
}

// CostByModel represents cost aggregated by model
type CostByModel struct {
	Model       string  `json:"model"`
	TotalCost   float64 `json:"totalCost"`
	TotalTokens int     `json:"totalTokens"`
	TraceCount  int     `json:"traceCount"`
}

// AnalyticsQueryOptions contains options for analytics queries
type AnalyticsQueryOptions struct {
	ProjectID   string
	StartTime   string
	EndTime     string
	Granularity string // hour, day, week
}

// GetOverviewMetrics retrieves overview metrics for a project
func (r *TraceRepository) GetOverviewMetrics(ctx context.Context, opts *AnalyticsQueryOptions) (*OverviewMetrics, error) {
	query := `
		SELECT
			count() as total_traces,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			if(count() > 0, avg(latency_ms), 0.0) as avg_latency_ms,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, toFloat64(countIf(status = 'error')) / toFloat64(count()), 0.0) as error_rate,
			uniqExactIf(user_id, user_id IS NOT NULL AND user_id != '') as unique_users,
			uniqExactIf(session_id, session_id IS NOT NULL AND session_id != '') as unique_sessions
		FROM traces
		WHERE 1=1
	`
	args := make([]interface{}, 0)

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if opts.StartTime != "" {
		query += " AND start_time >= ?"
		args = append(args, opts.StartTime)
	}
	if opts.EndTime != "" {
		query += " AND start_time <= ?"
		args = append(args, opts.EndTime)
	}

	var m OverviewMetrics
	var totalTraces, totalTokens, successCount, errorCount, uniqueUsers, uniqueSessions uint64
	err := r.conn.QueryRow(ctx, query, args...).Scan(
		&totalTraces, &totalTokens, &m.TotalCost, &m.AvgLatencyMs,
		&successCount, &errorCount, &m.ErrorRate,
		&uniqueUsers, &uniqueSessions,
	)
	if err != nil {
		return nil, err
	}

	m.TotalTraces = int(totalTraces)
	m.TotalTokens = int(totalTokens)
	m.SuccessCount = int(successCount)
	m.ErrorCount = int(errorCount)
	m.UniqueUsers = int(uniqueUsers)
	m.UniqueSessions = int(uniqueSessions)

	return &m, nil
}

// GetCostTimeSeries retrieves cost over time
func (r *TraceRepository) GetCostTimeSeries(ctx context.Context, opts *AnalyticsQueryOptions) ([]*TimeSeriesPoint, float64, error) {
	granularity := "toStartOfDay(start_time)"
	if opts.Granularity == "hour" {
		granularity = "toStartOfHour(start_time)"
	} else if opts.Granularity == "week" {
		granularity = "toStartOfWeek(start_time)"
	}

	query := `
		SELECT
			toString(` + granularity + `) as timestamp,
			toFloat64(sum(cost)) as value,
			count() as count
		FROM traces
		WHERE 1=1
	`
	args := make([]interface{}, 0)

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if opts.StartTime != "" {
		query += " AND start_time >= ?"
		args = append(args, opts.StartTime)
	}
	if opts.EndTime != "" {
		query += " AND start_time <= ?"
		args = append(args, opts.EndTime)
	}

	query += " GROUP BY timestamp ORDER BY timestamp ASC"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var points []*TimeSeriesPoint
	var totalCost float64
	for rows.Next() {
		var p TimeSeriesPoint
		var count uint64
		if err := rows.Scan(&p.Timestamp, &p.Value, &count); err != nil {
			return nil, 0, err
		}
		p.Count = int(count)
		totalCost += p.Value
		points = append(points, &p)
	}

	return points, totalCost, nil
}

// GetUsageTimeSeries retrieves token usage over time
func (r *TraceRepository) GetUsageTimeSeries(ctx context.Context, opts *AnalyticsQueryOptions) ([]*TimeSeriesPoint, int, error) {
	granularity := "toStartOfDay(start_time)"
	if opts.Granularity == "hour" {
		granularity = "toStartOfHour(start_time)"
	} else if opts.Granularity == "week" {
		granularity = "toStartOfWeek(start_time)"
	}

	query := `
		SELECT
			toString(` + granularity + `) as timestamp,
			sum(total_tokens) as value,
			count() as count
		FROM traces
		WHERE 1=1
	`
	args := make([]interface{}, 0)

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if opts.StartTime != "" {
		query += " AND start_time >= ?"
		args = append(args, opts.StartTime)
	}
	if opts.EndTime != "" {
		query += " AND start_time <= ?"
		args = append(args, opts.EndTime)
	}

	query += " GROUP BY timestamp ORDER BY timestamp ASC"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var points []*TimeSeriesPoint
	var totalTokens uint64
	for rows.Next() {
		var p TimeSeriesPoint
		var tokens, count uint64
		if err := rows.Scan(&p.Timestamp, &tokens, &count); err != nil {
			return nil, 0, err
		}
		p.Value = float64(tokens)
		p.Count = int(count)
		totalTokens += tokens
		points = append(points, &p)
	}

	return points, int(totalTokens), nil
}

// GetCostByModel retrieves cost aggregated by model
func (r *TraceRepository) GetCostByModel(ctx context.Context, opts *AnalyticsQueryOptions) ([]*CostByModel, error) {
	query := `
		SELECT
			model,
			toFloat64(sum(cost)) as total_cost,
			sum(total_tokens) as total_tokens,
			count() as trace_count
		FROM traces
		WHERE model != ''
	`
	args := make([]interface{}, 0)

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if opts.StartTime != "" {
		query += " AND start_time >= ?"
		args = append(args, opts.StartTime)
	}
	if opts.EndTime != "" {
		query += " AND start_time <= ?"
		args = append(args, opts.EndTime)
	}

	query += " GROUP BY model ORDER BY total_cost DESC LIMIT 10"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*CostByModel
	for rows.Next() {
		var c CostByModel
		var totalTokens, traceCount uint64
		if err := rows.Scan(&c.Model, &c.TotalCost, &totalTokens, &traceCount); err != nil {
			return nil, err
		}
		c.TotalTokens = int(totalTokens)
		c.TraceCount = int(traceCount)
		results = append(results, &c)
	}

	return results, nil
}

// ScoreFilter represents filtering options for score queries
type ScoreFilter struct {
	ProjectID uuid.UUID
	TraceID   *uuid.UUID
	SpanID    *uuid.UUID
	Name      *string
	Source    *string
	DataType  *string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// GetScores retrieves scores with filtering
func (r *TraceRepository) GetScores(ctx context.Context, filter *ScoreFilter) ([]*domain.Score, int, error) {
	where := []string{"project_id = ?"}
	args := []interface{}{filter.ProjectID.String()}

	if filter.TraceID != nil {
		where = append(where, "trace_id = ?")
		args = append(args, filter.TraceID.String())
	}

	if filter.SpanID != nil {
		where = append(where, "span_id = ?")
		args = append(args, filter.SpanID.String())
	}

	if filter.Name != nil {
		where = append(where, "name = ?")
		args = append(args, *filter.Name)
	}

	if filter.Source != nil {
		where = append(where, "source = ?")
		args = append(args, *filter.Source)
	}

	if filter.DataType != nil {
		where = append(where, "data_type = ?")
		args = append(args, *filter.DataType)
	}

	if filter.StartTime != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		where = append(where, "created_at <= ?")
		args = append(args, *filter.EndTime)
	}

	whereClause := strings.Join(where, " AND ")
	orderBy := "ORDER BY created_at DESC, id DESC"

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT count() FROM scores WHERE %s", whereClause)
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to get score count: %w", err)
	}

	// Get scores
	query := fmt.Sprintf(`
		SELECT id, project_id, trace_id, span_id, name, value, string_value,
			   data_type, source, config_id, comment, created_at
		FROM scores
		WHERE %s
		%s
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	args = append(args, limit, offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query scores: %w", err)
	}
	defer rows.Close()

	var scores []*domain.Score
	for rows.Next() {
		var score domain.Score
		var spanID, stringValue, configID, comment sql.NullString

		err := rows.Scan(
			&score.ID, &score.ProjectID, &score.TraceID, &spanID,
			&score.Name, &score.Value, &stringValue, &score.DataType,
			&score.Source, &configID, &comment, &score.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan score: %w", err)
		}

		// Convert nullable fields
		if spanID.Valid && spanID.String != "" {
			parsedSpanID, err := uuid.Parse(spanID.String)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid span_id: %w", err)
			}
			score.SpanID = &parsedSpanID
		}

		if stringValue.Valid {
			score.StringValue = &stringValue.String
		}

		if configID.Valid && configID.String != "" {
			parsedConfigID, err := uuid.Parse(configID.String)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid config_id: %w", err)
			}
			score.ConfigID = &parsedConfigID
		}

		if comment.Valid {
			score.Comment = &comment.String
		}

		scores = append(scores, &score)
	}

	return scores, int(total), nil
}

// GetScoreByID retrieves a single score by ID
func (r *TraceRepository) GetScoreByID(ctx context.Context, projectID, scoreID uuid.UUID) (*domain.Score, error) {
	query := `
		SELECT id, project_id, trace_id, span_id, name, value, string_value,
			   data_type, source, config_id, comment, created_at
		FROM scores
		WHERE project_id = ? AND id = ?
		LIMIT 1`

	var score domain.Score
	var spanID, stringValue, configID, comment sql.NullString

	err := r.conn.QueryRow(ctx, query, projectID.String(), scoreID.String()).Scan(
		&score.ID, &score.ProjectID, &score.TraceID, &spanID,
		&score.Name, &score.Value, &stringValue, &score.DataType,
		&score.Source, &configID, &comment, &score.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Score not found
		}
		return nil, fmt.Errorf("failed to get score: %w", err)
	}

	// Convert nullable fields
	if spanID.Valid && spanID.String != "" {
		parsedSpanID, err := uuid.Parse(spanID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid span_id: %w", err)
		}
		score.SpanID = &parsedSpanID
	}

	if stringValue.Valid {
		score.StringValue = &stringValue.String
	}

	if configID.Valid && configID.String != "" {
		parsedConfigID, err := uuid.Parse(configID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid config_id: %w", err)
		}
		score.ConfigID = &parsedConfigID
	}

	if comment.Valid {
		score.Comment = &comment.String
	}

	return &score, nil
}

// ScoreAggregation represents aggregated score statistics
type ScoreAggregation struct {
	Name       string         `json:"name"`
	DataType   string         `json:"dataType"`
	Count      int            `json:"count"`
	AvgValue   float64        `json:"avgValue,omitempty"`
	MinValue   float64        `json:"minValue,omitempty"`
	MaxValue   float64        `json:"maxValue,omitempty"`
	SumValue   float64        `json:"sumValue,omitempty"`
	Categories map[string]int `json:"categories,omitempty"` // For categorical scores
}

// GetScoreAggregations retrieves aggregated statistics for scores
func (r *TraceRepository) GetScoreAggregations(ctx context.Context, filter *ScoreFilter) ([]*ScoreAggregation, error) {
	where := []string{"project_id = ?"}
	args := []interface{}{filter.ProjectID.String()}

	if filter.TraceID != nil {
		where = append(where, "trace_id = ?")
		args = append(args, filter.TraceID.String())
	}

	if filter.SpanID != nil {
		where = append(where, "span_id = ?")
		args = append(args, filter.SpanID.String())
	}

	if filter.Name != nil {
		where = append(where, "name = ?")
		args = append(args, *filter.Name)
	}

	if filter.Source != nil {
		where = append(where, "source = ?")
		args = append(args, *filter.Source)
	}

	if filter.StartTime != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		where = append(where, "created_at <= ?")
		args = append(args, *filter.EndTime)
	}

	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
		SELECT
			name,
			data_type,
			count() as count,
			avg(value) as avg_value,
			min(value) as min_value,
			max(value) as max_value,
			sum(value) as sum_value
		FROM scores
		WHERE %s
		GROUP BY name, data_type
		ORDER BY name`, whereClause)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query score aggregations: %w", err)
	}
	defer rows.Close()

	var aggregations []*ScoreAggregation
	for rows.Next() {
		var agg ScoreAggregation
		err := rows.Scan(&agg.Name, &agg.DataType, &agg.Count, &agg.AvgValue, &agg.MinValue, &agg.MaxValue, &agg.SumValue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregation: %w", err)
		}
		aggregations = append(aggregations, &agg)
	}

	// For categorical scores, get category distributions
	for _, agg := range aggregations {
		if agg.DataType == "categorical" {
			catQuery := fmt.Sprintf(`
				SELECT string_value, count() as count
				FROM scores
				WHERE %s AND name = ? AND string_value IS NOT NULL
				GROUP BY string_value
				ORDER BY count DESC`, whereClause)

			catArgs := append(args, agg.Name)
			catRows, err := r.conn.Query(ctx, catQuery, catArgs...)
			if err != nil {
				return nil, fmt.Errorf("failed to query categories: %w", err)
			}

			agg.Categories = make(map[string]int)
			for catRows.Next() {
				var category string
				var count int
				if err := catRows.Scan(&category, &count); err != nil {
					catRows.Close()
					return nil, fmt.Errorf("failed to scan category: %w", err)
				}
				agg.Categories[category] = count
			}
			catRows.Close()
		}
	}

	return aggregations, nil
}

// ScoreTrend represents score trends over time
type ScoreTrend struct {
	TimePeriod time.Time      `json:"timePeriod"`
	Name       string         `json:"name"`
	DataType   string         `json:"dataType"`
	Count      int            `json:"count"`
	AvgValue   float64        `json:"avgValue,omitempty"`
	Categories map[string]int `json:"categories,omitempty"`
}

// GetScoreTrends retrieves score trends over time
func (r *TraceRepository) GetScoreTrends(ctx context.Context, filter *ScoreFilter, groupBy string) ([]*ScoreTrend, error) {
	where := []string{"project_id = ?"}
	args := []interface{}{filter.ProjectID.String()}

	if filter.TraceID != nil {
		where = append(where, "trace_id = ?")
		args = append(args, filter.TraceID.String())
	}

	if filter.SpanID != nil {
		where = append(where, "span_id = ?")
		args = append(args, filter.SpanID.String())
	}

	if filter.Name != nil {
		where = append(where, "name = ?")
		args = append(args, *filter.Name)
	}

	if filter.Source != nil {
		where = append(where, "source = ?")
		args = append(args, *filter.Source)
	}

	if filter.StartTime != nil {
		where = append(where, "created_at >= ?")
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		where = append(where, "created_at <= ?")
		args = append(args, *filter.EndTime)
	}

	whereClause := strings.Join(where, " AND ")

	var timeGroup string
	switch groupBy {
	case "hour":
		timeGroup = "toStartOfHour(created_at)"
	case "day":
		timeGroup = "toDate(created_at)"
	case "week":
		timeGroup = "toMonday(created_at)"
	case "month":
		timeGroup = "toStartOfMonth(created_at)"
	default:
		timeGroup = "toDate(created_at)" // Default to daily
	}

	query := fmt.Sprintf(`
		SELECT
			%s as time_period,
			name,
			data_type,
			count() as count,
			avg(value) as avg_value
		FROM scores
		WHERE %s
		GROUP BY time_period, name, data_type
		ORDER BY time_period, name`, timeGroup, whereClause)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query score trends: %w", err)
	}
	defer rows.Close()

	var trends []*ScoreTrend
	for rows.Next() {
		var trend ScoreTrend
		err := rows.Scan(&trend.TimePeriod, &trend.Name, &trend.DataType, &trend.Count, &trend.AvgValue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trends = append(trends, &trend)
	}

	// For categorical scores, get category trends
	for _, trend := range trends {
		if trend.DataType == "categorical" {
			catQuery := fmt.Sprintf(`
				SELECT string_value, count() as count
				FROM scores
				WHERE %s AND name = ? AND %s = ? AND string_value IS NOT NULL
				GROUP BY string_value
				ORDER BY count DESC`, whereClause, timeGroup)

			catArgs := append(args, trend.Name, trend.TimePeriod)
			catRows, err := r.conn.Query(ctx, catQuery, catArgs...)
			if err != nil {
				return nil, fmt.Errorf("failed to query category trends: %w", err)
			}

			trend.Categories = make(map[string]int)
			for catRows.Next() {
				var category string
				var count int
				if err := catRows.Scan(&category, &count); err != nil {
					catRows.Close()
					return nil, fmt.Errorf("failed to scan category trend: %w", err)
				}
				trend.Categories[category] = count
			}
			catRows.Close()
		}
	}

	return trends, nil
}

// ScoreComparison represents score comparisons across dimensions
type ScoreComparison struct {
	Dimension  string         `json:"dimension"`
	Value      string         `json:"value"`
	Name       string         `json:"name"`
	DataType   string         `json:"dataType"`
	Count      int            `json:"count"`
	AvgValue   float64        `json:"avgValue,omitempty"`
	Categories map[string]int `json:"categories,omitempty"`
}

// GetScoreComparisons retrieves score comparisons across dimensions
func (r *TraceRepository) GetScoreComparisons(ctx context.Context, filter *ScoreFilter, dimension string) ([]*ScoreComparison, error) {
	where := []string{"s.project_id = ?"}
	args := []interface{}{filter.ProjectID.String()}

	if filter.Name != nil {
		where = append(where, "s.name = ?")
		args = append(args, *filter.Name)
	}

	if filter.Source != nil {
		where = append(where, "s.source = ?")
		args = append(args, *filter.Source)
	}

	if filter.StartTime != nil {
		where = append(where, "s.created_at >= ?")
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		where = append(where, "s.created_at <= ?")
		args = append(args, *filter.EndTime)
	}

	whereClause := strings.Join(where, " AND ")

	var dimensionColumn, joinClause string
	switch dimension {
	case "model":
		dimensionColumn = "t.model"
		joinClause = "JOIN traces t ON s.trace_id = t.id"
	case "user":
		dimensionColumn = "t.user_id"
		joinClause = "LEFT JOIN traces t ON s.trace_id = t.id"
	case "session":
		dimensionColumn = "t.session_id"
		joinClause = "LEFT JOIN traces t ON s.trace_id = t.id"
	case "prompt":
		dimensionColumn = "t.prompt_id"
		joinClause = "LEFT JOIN traces t ON s.trace_id = t.id"
	default:
		return nil, fmt.Errorf("unsupported dimension: %s", dimension)
	}

	query := fmt.Sprintf(`
		SELECT
			%s as dimension_value,
			s.name,
			s.data_type,
			count() as count,
			avg(s.value) as avg_value
		FROM scores s
		%s
		WHERE %s AND %s IS NOT NULL
		GROUP BY dimension_value, s.name, s.data_type
		ORDER BY dimension_value, s.name`, dimensionColumn, joinClause, whereClause, dimensionColumn)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query score comparisons: %w", err)
	}
	defer rows.Close()

	var comparisons []*ScoreComparison
	for rows.Next() {
		var comp ScoreComparison
		err := rows.Scan(&comp.Value, &comp.Name, &comp.DataType, &comp.Count, &comp.AvgValue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comparison: %w", err)
		}
		comp.Dimension = dimension
		comparisons = append(comparisons, &comp)
	}

	// For categorical scores, get category comparisons
	for _, comp := range comparisons {
		if comp.DataType == "categorical" {
			catQuery := fmt.Sprintf(`
				SELECT s.string_value, count() as count
				FROM scores s
				%s
				WHERE %s AND %s = ? AND s.name = ? AND s.string_value IS NOT NULL
				GROUP BY s.string_value
				ORDER BY count DESC`, joinClause, whereClause, dimensionColumn)

			catArgs := append(args, comp.Value, comp.Name)
			catRows, err := r.conn.Query(ctx, catQuery, catArgs...)
			if err != nil {
				return nil, fmt.Errorf("failed to query category comparisons: %w", err)
			}

			comp.Categories = make(map[string]int)
			for catRows.Next() {
				var category string
				var count int
				if err := catRows.Scan(&category, &count); err != nil {
					catRows.Close()
					return nil, fmt.Errorf("failed to scan category comparison: %w", err)
				}
				comp.Categories[category] = count
			}
			catRows.Close()
		}
	}

	return comparisons, nil
}
