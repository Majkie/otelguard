package domain

import (
	"time"

	"github.com/google/uuid"
)

// Trace represents a complete trace/request in the system
type Trace struct {
	ID               uuid.UUID  `ch:"id" json:"id"`
	ProjectID        uuid.UUID  `ch:"project_id" json:"projectId"`
	SessionID        *string    `ch:"session_id" json:"sessionId,omitempty"`
	UserID           *string    `ch:"user_id" json:"userId,omitempty"`
	Name             string     `ch:"name" json:"name"`
	Input            string     `ch:"input" json:"input"`
	Output           string     `ch:"output" json:"output"`
	Metadata         string     `ch:"metadata" json:"metadata,omitempty"`
	StartTime        time.Time  `ch:"start_time" json:"startTime"`
	EndTime          time.Time  `ch:"end_time" json:"endTime"`
	LatencyMs        uint32     `ch:"latency_ms" json:"latencyMs"`
	TotalTokens      uint32     `ch:"total_tokens" json:"totalTokens"`
	PromptTokens     uint32     `ch:"prompt_tokens" json:"promptTokens"`
	CompletionTokens uint32     `ch:"completion_tokens" json:"completionTokens"`
	Cost             float64    `ch:"cost" json:"cost"`
	Model            string     `ch:"model" json:"model"`
	Tags             []string   `ch:"tags" json:"tags,omitempty"`
	Status           string     `ch:"status" json:"status"`
	ErrorMessage     *string    `ch:"error_message" json:"errorMessage,omitempty"`
}

// Span represents a single operation within a trace
type Span struct {
	ID           uuid.UUID  `ch:"id" json:"id"`
	TraceID      uuid.UUID  `ch:"trace_id" json:"traceId"`
	ParentSpanID *uuid.UUID `ch:"parent_span_id" json:"parentSpanId,omitempty"`
	ProjectID    uuid.UUID  `ch:"project_id" json:"projectId"`
	Name         string     `ch:"name" json:"name"`
	Type         string     `ch:"type" json:"type"` // llm, retrieval, tool, agent, embedding, custom
	Input        string     `ch:"input" json:"input"`
	Output       string     `ch:"output" json:"output"`
	Metadata     string     `ch:"metadata" json:"metadata,omitempty"`
	StartTime    time.Time  `ch:"start_time" json:"startTime"`
	EndTime      time.Time  `ch:"end_time" json:"endTime"`
	LatencyMs    uint32     `ch:"latency_ms" json:"latencyMs"`
	Tokens       uint32     `ch:"tokens" json:"tokens"`
	Cost         float64    `ch:"cost" json:"cost"`
	Model        *string    `ch:"model" json:"model,omitempty"`
	Status       string     `ch:"status" json:"status"`
	ErrorMessage *string    `ch:"error_message" json:"errorMessage,omitempty"`
}

// Score represents an evaluation score for a trace or span
type Score struct {
	ID          uuid.UUID  `ch:"id" json:"id"`
	ProjectID   uuid.UUID  `ch:"project_id" json:"projectId"`
	TraceID     uuid.UUID  `ch:"trace_id" json:"traceId"`
	SpanID      *uuid.UUID `ch:"span_id" json:"spanId,omitempty"`
	Name        string     `ch:"name" json:"name"`
	Value       float64    `ch:"value" json:"value"`
	StringValue *string    `ch:"string_value" json:"stringValue,omitempty"`
	DataType    string     `ch:"data_type" json:"dataType"` // numeric, boolean, categorical
	Source      string     `ch:"source" json:"source"`      // api, llm_judge, human, user_feedback
	ConfigID    *uuid.UUID `ch:"config_id" json:"configId,omitempty"`
	Comment     *string    `ch:"comment" json:"comment,omitempty"`
	CreatedAt   time.Time  `ch:"created_at" json:"createdAt"`
}

// GuardrailEvent represents a guardrail evaluation event
type GuardrailEvent struct {
	ID              uuid.UUID  `ch:"id" json:"id"`
	ProjectID       uuid.UUID  `ch:"project_id" json:"projectId"`
	TraceID         *uuid.UUID `ch:"trace_id" json:"traceId,omitempty"`
	SpanID          *uuid.UUID `ch:"span_id" json:"spanId,omitempty"`
	PolicyID        uuid.UUID  `ch:"policy_id" json:"policyId"`
	RuleID          uuid.UUID  `ch:"rule_id" json:"ruleId"`
	RuleType        string     `ch:"rule_type" json:"ruleType"`
	Triggered       bool       `ch:"triggered" json:"triggered"`
	Action          string     `ch:"action" json:"action"`
	ActionTaken     bool       `ch:"action_taken" json:"actionTaken"`
	InputText       string     `ch:"input_text" json:"inputText"`
	OutputText      *string    `ch:"output_text" json:"outputText,omitempty"`
	DetectionResult string     `ch:"detection_result" json:"detectionResult"`
	LatencyMs       uint32     `ch:"latency_ms" json:"latencyMs"`
	CreatedAt       time.Time  `ch:"created_at" json:"createdAt"`
}

// SpanType constants
const (
	SpanTypeLLM       = "llm"
	SpanTypeRetrieval = "retrieval"
	SpanTypeTool      = "tool"
	SpanTypeAgent     = "agent"
	SpanTypeEmbedding = "embedding"
	SpanTypeCustom    = "custom"
)

// TraceStatus constants
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusPending = "pending"
)
