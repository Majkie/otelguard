package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
)

// AgentRepository handles agent data access in ClickHouse
type AgentRepository struct {
	conn driver.Conn
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(conn driver.Conn) *AgentRepository {
	return &AgentRepository{conn: conn}
}

// AgentQueryOptions contains options for querying agents
type AgentQueryOptions struct {
	ProjectID   string
	TraceID     string
	AgentType   string
	Status      string
	ParentAgent string
	StartTime   string
	EndTime     string
	SortBy      string
	SortOrder   string
	Limit       int
	Offset      int
}

// InsertAgent inserts an agent into ClickHouse
func (r *AgentRepository) InsertAgent(ctx context.Context, agent *domain.Agent) error {
	query := `
		INSERT INTO agents (
			id, project_id, trace_id, span_id, parent_agent_id, name,
			agent_type, role, model, system_prompt, start_time, end_time,
			latency_ms, total_tokens, cost, status, error_message,
			metadata, tags, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	parentAgentID := ""
	if agent.ParentAgent != nil {
		parentAgentID = agent.ParentAgent.String()
	}
	model := ""
	if agent.Model != nil {
		model = *agent.Model
	}
	systemPrompt := ""
	if agent.SystemPrompt != nil {
		systemPrompt = *agent.SystemPrompt
	}
	errorMsg := ""
	if agent.ErrorMessage != nil {
		errorMsg = *agent.ErrorMessage
	}

	return r.conn.Exec(ctx, query,
		agent.ID,
		agent.ProjectID,
		agent.TraceID,
		agent.SpanID,
		parentAgentID,
		agent.Name,
		agent.Type,
		agent.Role,
		model,
		systemPrompt,
		agent.StartTime,
		agent.EndTime,
		agent.LatencyMs,
		agent.TotalTokens,
		agent.Cost,
		agent.Status,
		errorMsg,
		agent.Metadata,
		agent.Tags,
		agent.CreatedAt,
	)
}

// InsertAgents inserts multiple agents in a batch
func (r *AgentRepository) InsertAgents(ctx context.Context, agents []*domain.Agent) error {
	if len(agents) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO agents (
			id, project_id, trace_id, span_id, parent_agent_id, name,
			agent_type, role, model, system_prompt, start_time, end_time,
			latency_ms, total_tokens, cost, status, error_message,
			metadata, tags, created_at
		)
	`)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		parentAgentID := ""
		if agent.ParentAgent != nil {
			parentAgentID = agent.ParentAgent.String()
		}
		model := ""
		if agent.Model != nil {
			model = *agent.Model
		}
		systemPrompt := ""
		if agent.SystemPrompt != nil {
			systemPrompt = *agent.SystemPrompt
		}
		errorMsg := ""
		if agent.ErrorMessage != nil {
			errorMsg = *agent.ErrorMessage
		}

		err := batch.Append(
			agent.ID,
			agent.ProjectID,
			agent.TraceID,
			agent.SpanID,
			parentAgentID,
			agent.Name,
			agent.Type,
			agent.Role,
			model,
			systemPrompt,
			agent.StartTime,
			agent.EndTime,
			agent.LatencyMs,
			agent.TotalTokens,
			agent.Cost,
			agent.Status,
			errorMsg,
			agent.Metadata,
			agent.Tags,
			agent.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// GetAgentByID retrieves an agent by its ID
func (r *AgentRepository) GetAgentByID(ctx context.Context, projectID, agentID string) (*domain.Agent, error) {
	query := `
		SELECT
			id, project_id, trace_id, span_id, parent_agent_id, name,
			agent_type, role, model, system_prompt, start_time, end_time,
			latency_ms, total_tokens, cast(cost as Float64) as cost, status, error_message,
			metadata, tags, created_at
		FROM agents
		WHERE project_id = ? AND id = ?
		LIMIT 1
	`

	row := r.conn.QueryRow(ctx, query, projectID, agentID)

	var agent domain.Agent
	var parentAgentID, model, systemPrompt, errorMsg string
	var tags []string

	err := row.Scan(
		&agent.ID,
		&agent.ProjectID,
		&agent.TraceID,
		&agent.SpanID,
		&parentAgentID,
		&agent.Name,
		&agent.Type,
		&agent.Role,
		&model,
		&systemPrompt,
		&agent.StartTime,
		&agent.EndTime,
		&agent.LatencyMs,
		&agent.TotalTokens,
		&agent.Cost,
		&agent.Status,
		&errorMsg,
		&agent.Metadata,
		&tags,
		&agent.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if parentAgentID != "" {
		if parsed, err := uuid.Parse(parentAgentID); err == nil {
			agent.ParentAgent = &parsed
		}
	}
	if model != "" {
		agent.Model = &model
	}
	if systemPrompt != "" {
		agent.SystemPrompt = &systemPrompt
	}
	if errorMsg != "" {
		agent.ErrorMessage = &errorMsg
	}
	agent.Tags = tags

	return &agent, nil
}

// GetAgentsByTraceID retrieves all agents for a trace
func (r *AgentRepository) GetAgentsByTraceID(ctx context.Context, projectID, traceID string) ([]*domain.Agent, error) {
	query := `
		SELECT
			id, project_id, trace_id, span_id, parent_agent_id, name,
			agent_type, role, model, system_prompt, start_time, end_time,
			latency_ms, total_tokens, cast(cost as Float64) as cost, status, error_message,
			metadata, tags, created_at
		FROM agents
		WHERE project_id = ? AND trace_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAgents(rows)
}

// QueryAgents queries agents with filtering and pagination
func (r *AgentRepository) QueryAgents(ctx context.Context, opts *AgentQueryOptions) ([]*domain.Agent, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}

	if opts.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, opts.ProjectID)
	}
	if opts.TraceID != "" {
		conditions = append(conditions, "trace_id = ?")
		args = append(args, opts.TraceID)
	}
	if opts.AgentType != "" {
		conditions = append(conditions, "agent_type = ?")
		args = append(args, opts.AgentType)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.ParentAgent != "" {
		conditions = append(conditions, "parent_agent_id = ?")
		args = append(args, opts.ParentAgent)
	}
	if opts.StartTime != "" {
		t, err := time.Parse(time.RFC3339, opts.StartTime)
		if err == nil {
			conditions = append(conditions, "start_time >= ?")
			args = append(args, t)
		}
	}
	if opts.EndTime != "" {
		t, err := time.Parse(time.RFC3339, opts.EndTime)
		if err == nil {
			conditions = append(conditions, "end_time <= ?")
			args = append(args, t)
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT count() FROM agents %s", whereClause)
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sort
	sortBy := "start_time"
	sortOrder := "DESC"
	if opts.SortBy != "" {
		sortBy = opts.SortBy
	}
	if opts.SortOrder != "" {
		sortOrder = strings.ToUpper(opts.SortOrder)
	}

	// Limit and offset
	limit := 50
	offset := 0
	if opts.Limit > 0 {
		limit = opts.Limit
	}
	if opts.Offset > 0 {
		offset = opts.Offset
	}

	query := fmt.Sprintf(`
		SELECT
			id, project_id, trace_id, span_id, parent_agent_id, name,
			agent_type, role, model, system_prompt, start_time, end_time,
			latency_ms, total_tokens, cast(cost as Float64) as cost, status, error_message,
			metadata, tags, created_at
		FROM agents
		%s
		ORDER BY %s %s
		LIMIT %d OFFSET %d
	`, whereClause, sortBy, sortOrder, limit, offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	agents, err := scanAgents(rows)
	if err != nil {
		return nil, 0, err
	}

	return agents, int(total), nil
}

// scanAgents scans multiple agent rows
func scanAgents(rows driver.Rows) ([]*domain.Agent, error) {
	var agents []*domain.Agent

	for rows.Next() {
		var agent domain.Agent
		var parentAgentID, model, systemPrompt, errorMsg string
		var tags []string

		err := rows.Scan(
			&agent.ID,
			&agent.ProjectID,
			&agent.TraceID,
			&agent.SpanID,
			&parentAgentID,
			&agent.Name,
			&agent.Type,
			&agent.Role,
			&model,
			&systemPrompt,
			&agent.StartTime,
			&agent.EndTime,
			&agent.LatencyMs,
			&agent.TotalTokens,
			&agent.Cost,
			&agent.Status,
			&errorMsg,
			&agent.Metadata,
			&tags,
			&agent.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentAgentID != "" {
			if parsed, err := uuid.Parse(parentAgentID); err == nil {
				agent.ParentAgent = &parsed
			}
		}
		if model != "" {
			agent.Model = &model
		}
		if systemPrompt != "" {
			agent.SystemPrompt = &systemPrompt
		}
		if errorMsg != "" {
			agent.ErrorMessage = &errorMsg
		}
		agent.Tags = tags

		agents = append(agents, &agent)
	}

	return agents, nil
}

// InsertAgentRelationship inserts an agent relationship
func (r *AgentRepository) InsertAgentRelationship(ctx context.Context, rel *domain.AgentRelationship) error {
	query := `
		INSERT INTO agent_relationships (
			id, project_id, trace_id, source_agent_id, target_agent_id,
			relation_type, timestamp, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return r.conn.Exec(ctx, query,
		rel.ID,
		rel.ProjectID,
		rel.TraceID,
		rel.SourceAgentID,
		rel.TargetAgentID,
		rel.RelationType,
		rel.Timestamp,
		rel.Metadata,
		rel.CreatedAt,
	)
}

// GetAgentRelationships retrieves relationships for a trace
func (r *AgentRepository) GetAgentRelationships(ctx context.Context, projectID, traceID string) ([]*domain.AgentRelationship, error) {
	query := `
		SELECT
			id, project_id, trace_id, source_agent_id, target_agent_id,
			relation_type, timestamp, metadata, created_at
		FROM agent_relationships
		WHERE project_id = ? AND trace_id = ?
		ORDER BY timestamp ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []*domain.AgentRelationship

	for rows.Next() {
		var rel domain.AgentRelationship
		err := rows.Scan(
			&rel.ID,
			&rel.ProjectID,
			&rel.TraceID,
			&rel.SourceAgentID,
			&rel.TargetAgentID,
			&rel.RelationType,
			&rel.Timestamp,
			&rel.Metadata,
			&rel.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, &rel)
	}

	return relationships, nil
}

// InsertToolCall inserts a tool call
func (r *AgentRepository) InsertToolCall(ctx context.Context, tc *domain.ToolCall) error {
	query := `
		INSERT INTO tool_calls (
			id, project_id, trace_id, span_id, agent_id, name,
			description, input, output, start_time, end_time,
			latency_ms, status, error_message, retry_count,
			metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	agentID := ""
	if tc.AgentID != nil {
		agentID = tc.AgentID.String()
	}
	errorMsg := ""
	if tc.ErrorMessage != nil {
		errorMsg = *tc.ErrorMessage
	}

	return r.conn.Exec(ctx, query,
		tc.ID,
		tc.ProjectID,
		tc.TraceID,
		tc.SpanID,
		agentID,
		tc.Name,
		tc.Description,
		tc.Input,
		tc.Output,
		tc.StartTime,
		tc.EndTime,
		tc.LatencyMs,
		tc.Status,
		errorMsg,
		tc.RetryCount,
		tc.Metadata,
		tc.CreatedAt,
	)
}

// InsertToolCalls inserts multiple tool calls in a batch
func (r *AgentRepository) InsertToolCalls(ctx context.Context, toolCalls []*domain.ToolCall) error {
	if len(toolCalls) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO tool_calls (
			id, project_id, trace_id, span_id, agent_id, name,
			description, input, output, start_time, end_time,
			latency_ms, status, error_message, retry_count,
			metadata, created_at
		)
	`)
	if err != nil {
		return err
	}

	for _, tc := range toolCalls {
		agentID := ""
		if tc.AgentID != nil {
			agentID = tc.AgentID.String()
		}
		errorMsg := ""
		if tc.ErrorMessage != nil {
			errorMsg = *tc.ErrorMessage
		}

		err := batch.Append(
			tc.ID,
			tc.ProjectID,
			tc.TraceID,
			tc.SpanID,
			agentID,
			tc.Name,
			tc.Description,
			tc.Input,
			tc.Output,
			tc.StartTime,
			tc.EndTime,
			tc.LatencyMs,
			tc.Status,
			errorMsg,
			tc.RetryCount,
			tc.Metadata,
			tc.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// GetToolCallsByTraceID retrieves tool calls for a trace
func (r *AgentRepository) GetToolCallsByTraceID(ctx context.Context, projectID, traceID string) ([]*domain.ToolCall, error) {
	query := `
		SELECT
			id, project_id, trace_id, span_id, agent_id, name,
			description, input, output, start_time, end_time,
			latency_ms, status, error_message, retry_count,
			metadata, created_at
		FROM tool_calls
		WHERE project_id = ? AND trace_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToolCalls(rows)
}

// GetToolCallsByAgentID retrieves tool calls for an agent
func (r *AgentRepository) GetToolCallsByAgentID(ctx context.Context, projectID, agentID string) ([]*domain.ToolCall, error) {
	query := `
		SELECT
			id, project_id, trace_id, span_id, agent_id, name,
			description, input, output, start_time, end_time,
			latency_ms, status, error_message, retry_count,
			metadata, created_at
		FROM tool_calls
		WHERE project_id = ? AND agent_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToolCalls(rows)
}

func scanToolCalls(rows driver.Rows) ([]*domain.ToolCall, error) {
	var toolCalls []*domain.ToolCall

	for rows.Next() {
		var tc domain.ToolCall
		var agentID, errorMsg string

		err := rows.Scan(
			&tc.ID,
			&tc.ProjectID,
			&tc.TraceID,
			&tc.SpanID,
			&agentID,
			&tc.Name,
			&tc.Description,
			&tc.Input,
			&tc.Output,
			&tc.StartTime,
			&tc.EndTime,
			&tc.LatencyMs,
			&tc.Status,
			&errorMsg,
			&tc.RetryCount,
			&tc.Metadata,
			&tc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if agentID != "" {
			if parsed, err := uuid.Parse(agentID); err == nil {
				tc.AgentID = &parsed
			}
		}
		if errorMsg != "" {
			tc.ErrorMessage = &errorMsg
		}

		toolCalls = append(toolCalls, &tc)
	}

	return toolCalls, nil
}

// InsertAgentMessage inserts an agent message
func (r *AgentRepository) InsertAgentMessage(ctx context.Context, msg *domain.AgentMessage) error {
	query := `
		INSERT INTO agent_messages (
			id, project_id, trace_id, span_id, from_agent_id, to_agent_id,
			message_type, role, content, content_type, sequence_num,
			parent_msg_id, token_count, timestamp, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	spanID := ""
	if msg.SpanID != nil {
		spanID = msg.SpanID.String()
	}
	parentMsgID := ""
	if msg.ParentMsgID != nil {
		parentMsgID = msg.ParentMsgID.String()
	}

	return r.conn.Exec(ctx, query,
		msg.ID,
		msg.ProjectID,
		msg.TraceID,
		spanID,
		msg.FromAgentID,
		msg.ToAgentID,
		msg.MessageType,
		msg.Role,
		msg.Content,
		msg.ContentType,
		msg.SequenceNum,
		parentMsgID,
		msg.TokenCount,
		msg.Timestamp,
		msg.Metadata,
		msg.CreatedAt,
	)
}

// GetAgentMessagesByTraceID retrieves messages for a trace
func (r *AgentRepository) GetAgentMessagesByTraceID(ctx context.Context, projectID, traceID string) ([]*domain.AgentMessage, error) {
	query := `
		SELECT
			id, project_id, trace_id, span_id, from_agent_id, to_agent_id,
			message_type, role, content, content_type, sequence_num,
			parent_msg_id, token_count, timestamp, metadata, created_at
		FROM agent_messages
		WHERE project_id = ? AND trace_id = ?
		ORDER BY sequence_num ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.AgentMessage

	for rows.Next() {
		var msg domain.AgentMessage
		var spanID, parentMsgID string

		err := rows.Scan(
			&msg.ID,
			&msg.ProjectID,
			&msg.TraceID,
			&spanID,
			&msg.FromAgentID,
			&msg.ToAgentID,
			&msg.MessageType,
			&msg.Role,
			&msg.Content,
			&msg.ContentType,
			&msg.SequenceNum,
			&parentMsgID,
			&msg.TokenCount,
			&msg.Timestamp,
			&msg.Metadata,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if spanID != "" {
			if parsed, err := uuid.Parse(spanID); err == nil {
				msg.SpanID = &parsed
			}
		}
		if parentMsgID != "" {
			if parsed, err := uuid.Parse(parentMsgID); err == nil {
				msg.ParentMsgID = &parsed
			}
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

// InsertAgentState inserts an agent state snapshot
func (r *AgentRepository) InsertAgentState(ctx context.Context, state *domain.AgentState) error {
	query := `
		INSERT INTO agent_states (
			id, project_id, trace_id, agent_id, span_id, sequence_num,
			state, variables, memory, plan, reasoning,
			timestamp, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	spanID := ""
	if state.SpanID != nil {
		spanID = state.SpanID.String()
	}

	return r.conn.Exec(ctx, query,
		state.ID,
		state.ProjectID,
		state.TraceID,
		state.AgentID,
		spanID,
		state.SequenceNum,
		state.State,
		state.Variables,
		state.Memory,
		state.Plan,
		state.Reasoning,
		state.Timestamp,
		state.Metadata,
		state.CreatedAt,
	)
}

// GetAgentStatesByAgentID retrieves state snapshots for an agent
func (r *AgentRepository) GetAgentStatesByAgentID(ctx context.Context, projectID, agentID string) ([]*domain.AgentState, error) {
	query := `
		SELECT
			id, project_id, trace_id, agent_id, span_id, sequence_num,
			state, variables, memory, plan, reasoning,
			timestamp, metadata, created_at
		FROM agent_states
		WHERE project_id = ? AND agent_id = ?
		ORDER BY sequence_num ASC
	`

	rows, err := r.conn.Query(ctx, query, projectID, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.AgentState

	for rows.Next() {
		var state domain.AgentState
		var spanID string

		err := rows.Scan(
			&state.ID,
			&state.ProjectID,
			&state.TraceID,
			&state.AgentID,
			&spanID,
			&state.SequenceNum,
			&state.State,
			&state.Variables,
			&state.Memory,
			&state.Plan,
			&state.Reasoning,
			&state.Timestamp,
			&state.Metadata,
			&state.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if spanID != "" {
			if parsed, err := uuid.Parse(spanID); err == nil {
				state.SpanID = &parsed
			}
		}

		states = append(states, &state)
	}

	return states, nil
}

// AgentStatistics represents aggregated statistics for agents
type AgentStatistics struct {
	TotalAgents       int     `json:"totalAgents"`
	OrchestratorCount int     `json:"orchestratorCount"`
	WorkerCount       int     `json:"workerCount"`
	ToolCallerCount   int     `json:"toolCallerCount"`
	TotalLatencyMs    uint64  `json:"totalLatencyMs"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
	TotalTokens       uint64  `json:"totalTokens"`
	TotalCost         float64 `json:"totalCost"`
	ErrorCount        int     `json:"errorCount"`
	SuccessRate       float64 `json:"successRate"`
}

// GetAgentStatistics retrieves aggregated agent statistics for a project
func (r *AgentRepository) GetAgentStatistics(ctx context.Context, projectID string, startTime, endTime time.Time) (*AgentStatistics, error) {
	query := `
		SELECT
			count() as total_agents,
			countIf(agent_type = 'orchestrator') as orchestrator_count,
			countIf(agent_type = 'worker') as worker_count,
			countIf(agent_type = 'tool_caller') as tool_caller_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			sum(cast(cost as Float64)) as total_cost,
			countIf(status = 'error') as error_count
		FROM agents
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
	`

	row := r.conn.QueryRow(ctx, query, projectID, startTime, endTime)

	var stats AgentStatistics
	err := row.Scan(
		&stats.TotalAgents,
		&stats.OrchestratorCount,
		&stats.WorkerCount,
		&stats.ToolCallerCount,
		&stats.TotalLatencyMs,
		&stats.AvgLatencyMs,
		&stats.TotalTokens,
		&stats.TotalCost,
		&stats.ErrorCount,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalAgents > 0 {
		stats.SuccessRate = float64(stats.TotalAgents-stats.ErrorCount) / float64(stats.TotalAgents) * 100
	}

	return &stats, nil
}

// ToolCallStatistics represents aggregated statistics for tool calls
type ToolCallStatistics struct {
	Name           string  `json:"name"`
	CallCount      int     `json:"callCount"`
	TotalLatencyMs uint64  `json:"totalLatencyMs"`
	AvgLatencyMs   float64 `json:"avgLatencyMs"`
	SuccessCount   int     `json:"successCount"`
	ErrorCount     int     `json:"errorCount"`
	TotalRetries   int     `json:"totalRetries"`
	SuccessRate    float64 `json:"successRate"`
}

// GetToolCallStatistics retrieves aggregated tool call statistics
func (r *AgentRepository) GetToolCallStatistics(ctx context.Context, projectID string, startTime, endTime time.Time) ([]*ToolCallStatistics, error) {
	query := `
		SELECT
			name,
			count() as call_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			sum(retry_count) as total_retries
		FROM tool_calls
		WHERE project_id = ?
		  AND start_time >= ?
		  AND start_time <= ?
		GROUP BY name
		ORDER BY call_count DESC
	`

	rows, err := r.conn.Query(ctx, query, projectID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*ToolCallStatistics

	for rows.Next() {
		var s ToolCallStatistics
		err := rows.Scan(
			&s.Name,
			&s.CallCount,
			&s.TotalLatencyMs,
			&s.AvgLatencyMs,
			&s.SuccessCount,
			&s.ErrorCount,
			&s.TotalRetries,
		)
		if err != nil {
			return nil, err
		}

		if s.CallCount > 0 {
			s.SuccessRate = float64(s.SuccessCount) / float64(s.CallCount) * 100
		}

		stats = append(stats, &s)
	}

	return stats, nil
}
