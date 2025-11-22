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
	ProjectID  string
	SessionID  string
	UserID     string
	Model      string
	Name       string   // Search by name (partial match)
	Status     string   // Filter by status (success, error, pending)
	Tags       []string // Filter by tags (any match)
	StartTime  string   // ISO8601 timestamp
	EndTime    string   // ISO8601 timestamp
	MinLatency int      // Minimum latency in ms
	MaxLatency int      // Maximum latency in ms
	MinCost    float64  // Minimum cost
	MaxCost    float64  // Maximum cost
	SortBy     string   // Field to sort by (start_time, latency_ms, cost, total_tokens)
	SortOrder  string   // ASC or DESC
	Limit      int
	Offset     int
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

	// Build filter clauses
	filterClause := ""

	if opts.ProjectID != "" {
		filterClause += " AND project_id = ?"
		args = append(args, opts.ProjectID)
		countArgs = append(countArgs, opts.ProjectID)
	}
	if opts.SessionID != "" {
		filterClause += " AND session_id = ?"
		args = append(args, opts.SessionID)
		countArgs = append(countArgs, opts.SessionID)
	}
	if opts.UserID != "" {
		filterClause += " AND user_id = ?"
		args = append(args, opts.UserID)
		countArgs = append(countArgs, opts.UserID)
	}
	if opts.Model != "" {
		filterClause += " AND model = ?"
		args = append(args, opts.Model)
		countArgs = append(countArgs, opts.Model)
	}
	if opts.Name != "" {
		filterClause += " AND name ILIKE ?"
		args = append(args, "%"+opts.Name+"%")
		countArgs = append(countArgs, "%"+opts.Name+"%")
	}
	if opts.Status != "" {
		filterClause += " AND status = ?"
		args = append(args, opts.Status)
		countArgs = append(countArgs, opts.Status)
	}
	if len(opts.Tags) > 0 {
		filterClause += " AND hasAny(tags, ?)"
		args = append(args, opts.Tags)
		countArgs = append(countArgs, opts.Tags)
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

	// Get total count
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery+filterClause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine sort order
	sortBy := "start_time"
	if opts.SortBy != "" {
		// Validate sort field to prevent SQL injection
		validSortFields := map[string]bool{
			"start_time":   true,
			"latency_ms":   true,
			"cost":         true,
			"total_tokens": true,
			"name":         true,
			"model":        true,
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

// Session represents aggregated session data
type Session struct {
	SessionID       string  `json:"sessionId"`
	ProjectID       string  `json:"projectId"`
	UserID          string  `json:"userId,omitempty"`
	TraceCount      int     `json:"traceCount"`
	TotalLatencyMs  int     `json:"totalLatencyMs"`
	TotalTokens     int     `json:"totalTokens"`
	TotalCost       float64 `json:"totalCost"`
	SuccessCount    int     `json:"successCount"`
	ErrorCount      int     `json:"errorCount"`
	FirstTraceTime  string  `json:"firstTraceTime"`
	LastTraceTime   string  `json:"lastTraceTime"`
	Models          []string `json:"models,omitempty"`
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
			any(user_id) as user_id,
			count() as trace_count,
			sum(latency_ms) as total_latency_ms,
			sum(total_tokens) as total_tokens,
			sum(cost) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			min(start_time) as first_trace_time,
			max(start_time) as last_trace_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE session_id != ''
	`
	countQuery := `SELECT count(DISTINCT session_id) FROM traces WHERE session_id != ''`
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
			sum(cost) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			min(start_time) as first_trace_time,
			max(start_time) as last_trace_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE session_id = ?
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

// User represents aggregated user activity data
type User struct {
	UserID          string   `json:"userId"`
	ProjectID       string   `json:"projectId"`
	TraceCount      int      `json:"traceCount"`
	SessionCount    int      `json:"sessionCount"`
	TotalLatencyMs  int      `json:"totalLatencyMs"`
	AvgLatencyMs    float64  `json:"avgLatencyMs"`
	TotalTokens     int      `json:"totalTokens"`
	TotalCost       float64  `json:"totalCost"`
	SuccessCount    int      `json:"successCount"`
	ErrorCount      int      `json:"errorCount"`
	SuccessRate     float64  `json:"successRate"`
	FirstSeenTime   string   `json:"firstSeenTime"`
	LastSeenTime    string   `json:"lastSeenTime"`
	Models          []string `json:"models,omitempty"`
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
			uniqExact(session_id) as session_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			sum(cost) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'success') / count(), 0) as success_rate,
			min(start_time) as first_seen_time,
			max(start_time) as last_seen_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE user_id != ''
	`
	countQuery := `SELECT count(DISTINCT user_id) FROM traces WHERE user_id != ''`
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
		if err := rows.Scan(
			&u.UserID, &u.ProjectID,
			&u.TraceCount, &u.SessionCount, &u.TotalLatencyMs, &u.AvgLatencyMs,
			&u.TotalTokens, &u.TotalCost,
			&u.SuccessCount, &u.ErrorCount, &u.SuccessRate,
			&u.FirstSeenTime, &u.LastSeenTime, &u.Models,
		); err != nil {
			return nil, 0, err
		}
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
			uniqExact(session_id) as session_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			sum(cost) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'success') / count(), 0) as success_rate,
			min(start_time) as first_seen_time,
			max(start_time) as last_seen_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE user_id = ?
		GROUP BY user_id, project_id
	`

	var u User
	err := r.conn.QueryRow(ctx, query, userID).Scan(
		&u.UserID, &u.ProjectID,
		&u.TraceCount, &u.SessionCount, &u.TotalLatencyMs, &u.AvgLatencyMs,
		&u.TotalTokens, &u.TotalCost,
		&u.SuccessCount, &u.ErrorCount, &u.SuccessRate,
		&u.FirstSeenTime, &u.LastSeenTime, &u.Models,
	)
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
	ProjectID  string
	StartTime  string
	EndTime    string
	Granularity string // hour, day, week
}

// GetOverviewMetrics retrieves overview metrics for a project
func (r *TraceRepository) GetOverviewMetrics(ctx context.Context, opts *AnalyticsQueryOptions) (*OverviewMetrics, error) {
	query := `
		SELECT
			count() as total_traces,
			sum(total_tokens) as total_tokens,
			sum(cost) as total_cost,
			avg(latency_ms) as avg_latency_ms,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'error') / count(), 0) as error_rate,
			uniqExact(user_id) as unique_users,
			uniqExact(session_id) as unique_sessions
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
	err := r.conn.QueryRow(ctx, query, args...).Scan(
		&m.TotalTraces, &m.TotalTokens, &m.TotalCost, &m.AvgLatencyMs,
		&m.SuccessCount, &m.ErrorCount, &m.ErrorRate,
		&m.UniqueUsers, &m.UniqueSessions,
	)
	if err != nil {
		return nil, err
	}

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
			` + granularity + ` as timestamp,
			sum(cost) as value,
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
		if err := rows.Scan(&p.Timestamp, &p.Value, &p.Count); err != nil {
			return nil, 0, err
		}
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
			` + granularity + ` as timestamp,
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
	var totalTokens int
	for rows.Next() {
		var p TimeSeriesPoint
		var tokens int
		if err := rows.Scan(&p.Timestamp, &tokens, &p.Count); err != nil {
			return nil, 0, err
		}
		p.Value = float64(tokens)
		totalTokens += tokens
		points = append(points, &p)
	}

	return points, totalTokens, nil
}

// GetCostByModel retrieves cost aggregated by model
func (r *TraceRepository) GetCostByModel(ctx context.Context, opts *AnalyticsQueryOptions) ([]*CostByModel, error) {
	query := `
		SELECT
			model,
			sum(cost) as total_cost,
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
		if err := rows.Scan(&c.Model, &c.TotalCost, &c.TotalTokens, &c.TraceCount); err != nil {
			return nil, err
		}
		results = append(results, &c)
	}

	return results, nil
}
