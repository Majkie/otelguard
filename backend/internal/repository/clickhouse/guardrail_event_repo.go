package clickhouse

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/otelguard/otelguard/internal/domain"
)

// GuardrailEventRepository handles guardrail event data access in ClickHouse
type GuardrailEventRepository struct {
	conn driver.Conn
}

// NewGuardrailEventRepository creates a new guardrail event repository
func NewGuardrailEventRepository(conn driver.Conn) *GuardrailEventRepository {
	return &GuardrailEventRepository{conn: conn}
}

// Insert inserts a guardrail event into ClickHouse
func (r *GuardrailEventRepository) Insert(ctx context.Context, event *domain.GuardrailEvent) error {
	query := `
		INSERT INTO guardrail_events (
			id, project_id, trace_id, span_id, policy_id, rule_id,
			rule_type, triggered, action, action_taken, input_text,
			output_text, detection_result, latency_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	traceID := ""
	if event.TraceID != nil {
		traceID = event.TraceID.String()
	}
	spanID := ""
	if event.SpanID != nil {
		spanID = event.SpanID.String()
	}
	outputText := ""
	if event.OutputText != nil {
		outputText = *event.OutputText
	}

	return r.conn.Exec(ctx, query,
		event.ID,
		event.ProjectID,
		traceID,
		spanID,
		event.PolicyID,
		event.RuleID,
		event.RuleType,
		event.Triggered,
		event.Action,
		event.ActionTaken,
		event.InputText,
		outputText,
		event.DetectionResult,
		event.LatencyMs,
		event.CreatedAt,
	)
}

// QueryEvents retrieves guardrail events with filtering
func (r *GuardrailEventRepository) QueryEvents(ctx context.Context, projectID string, limit, offset int) ([]*domain.GuardrailEvent, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM guardrail_events WHERE project_id = ?`
	var total uint64
	if err := r.conn.QueryRow(ctx, countQuery, projectID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT
			id, project_id, trace_id, span_id, policy_id, rule_id,
			rule_type, triggered, action, action_taken, input_text,
			output_text, detection_result, latency_ms, created_at
		FROM guardrail_events
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.conn.Query(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.GuardrailEvent
	for rows.Next() {
		var e domain.GuardrailEvent
		var traceID, spanID, outputText string
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &traceID, &spanID, &e.PolicyID, &e.RuleID,
			&e.RuleType, &e.Triggered, &e.Action, &e.ActionTaken, &e.InputText,
			&outputText, &e.DetectionResult, &e.LatencyMs, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if outputText != "" {
			e.OutputText = &outputText
		}
		events = append(events, &e)
	}

	return events, int(total), nil
}
