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

// Event represents a generic event (log, exception, custom event, etc.)
type Event struct {
	ID                 uuid.UUID         `ch:"id" json:"id"`
	ProjectID          uuid.UUID         `ch:"project_id" json:"projectId"`
	TraceID            *uuid.UUID        `ch:"trace_id" json:"traceId,omitempty"`
	SpanID             *uuid.UUID        `ch:"span_id" json:"spanId,omitempty"`
	SessionID          *string           `ch:"session_id" json:"sessionId,omitempty"`
	UserID             *string           `ch:"user_id" json:"userId,omitempty"`
	Name               string            `ch:"name" json:"name"`
	Type               string            `ch:"type" json:"type"` // log, exception, custom, user_action, system
	Level              string            `ch:"level" json:"level"`
	Message            string            `ch:"message" json:"message"`
	Data               string            `ch:"data" json:"data,omitempty"`
	ExceptionType      *string           `ch:"exception_type" json:"exceptionType,omitempty"`
	ExceptionMessage   *string           `ch:"exception_message" json:"exceptionMessage,omitempty"`
	ExceptionStacktrace *string           `ch:"exception_stacktrace" json:"exceptionStacktrace,omitempty"`
	Source             string            `ch:"source" json:"source"`
	Environment        string            `ch:"environment" json:"environment"`
	Version            string            `ch:"version" json:"version"`
	Tags               []string          `ch:"tags" json:"tags,omitempty"`
	Attributes         map[string]string `ch:"attributes" json:"attributes,omitempty"`
	Timestamp          time.Time         `ch:"timestamp" json:"timestamp"`
	CreatedAt          time.Time         `ch:"created_at" json:"createdAt"`
}

// EventType constants
const (
	EventTypeLog        = "log"
	EventTypeException  = "exception"
	EventTypeCustom     = "custom"
	EventTypeUserAction = "user_action"
	EventTypeSystem     = "system"
)

// EventLevel constants
const (
	EventLevelDebug = "debug"
	EventLevelInfo  = "info"
	EventLevelWarn  = "warn"
	EventLevelError = "error"
	EventLevelFatal = "fatal"
)

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

// SpanNode represents a span with its children in a tree structure
type SpanNode struct {
	Span     *Span       `json:"span"`
	Children []*SpanNode `json:"children,omitempty"`
	Depth    int         `json:"depth"`
}

// SpanTree represents a hierarchical tree of spans
type SpanTree struct {
	TraceID   uuid.UUID   `json:"traceId"`
	RootSpans []*SpanNode `json:"rootSpans"`
	TotalSpans int        `json:"totalSpans"`
	MaxDepth   int        `json:"maxDepth"`
}

// BuildSpanTree builds a tree structure from a flat list of spans
func BuildSpanTree(spans []*Span) *SpanTree {
	if len(spans) == 0 {
		return &SpanTree{RootSpans: []*SpanNode{}}
	}

	tree := &SpanTree{
		TraceID:    spans[0].TraceID,
		TotalSpans: len(spans),
	}

	// Create a map for quick lookup
	nodeMap := make(map[uuid.UUID]*SpanNode)
	for _, span := range spans {
		nodeMap[span.ID] = &SpanNode{
			Span:     span,
			Children: []*SpanNode{},
		}
	}

	// Build tree relationships
	var rootSpans []*SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == nil {
			// This is a root span
			rootSpans = append(rootSpans, node)
		} else {
			// Find parent and add as child
			if parentNode, ok := nodeMap[*span.ParentSpanID]; ok {
				parentNode.Children = append(parentNode.Children, node)
			} else {
				// Parent not found, treat as root
				rootSpans = append(rootSpans, node)
			}
		}
	}

	tree.RootSpans = rootSpans

	// Calculate depths
	tree.MaxDepth = calculateDepths(rootSpans, 0)

	return tree
}

// calculateDepths recursively sets depths and returns max depth
func calculateDepths(nodes []*SpanNode, depth int) int {
	maxDepth := depth
	for _, node := range nodes {
		node.Depth = depth
		if len(node.Children) > 0 {
			childMaxDepth := calculateDepths(node.Children, depth+1)
			if childMaxDepth > maxDepth {
				maxDepth = childMaxDepth
			}
		}
	}
	return maxDepth
}

// FlattenSpanTree converts a span tree back to a flat slice in DFS order
func (t *SpanTree) FlattenSpanTree() []*Span {
	var spans []*Span
	var flatten func(nodes []*SpanNode)
	flatten = func(nodes []*SpanNode) {
		for _, node := range nodes {
			spans = append(spans, node.Span)
			if len(node.Children) > 0 {
				flatten(node.Children)
			}
		}
	}
	flatten(t.RootSpans)
	return spans
}

// GetSpanByID finds a span in the tree by its ID
func (t *SpanTree) GetSpanByID(spanID uuid.UUID) *SpanNode {
	var find func(nodes []*SpanNode) *SpanNode
	find = func(nodes []*SpanNode) *SpanNode {
		for _, node := range nodes {
			if node.Span.ID == spanID {
				return node
			}
			if found := find(node.Children); found != nil {
				return found
			}
		}
		return nil
	}
	return find(t.RootSpans)
}

// GetAncestors returns all ancestor spans for a given span ID
func (t *SpanTree) GetAncestors(spanID uuid.UUID) []*Span {
	node := t.GetSpanByID(spanID)
	if node == nil || node.Span.ParentSpanID == nil {
		return nil
	}

	var ancestors []*Span
	currentParentID := node.Span.ParentSpanID
	for currentParentID != nil {
		parentNode := t.GetSpanByID(*currentParentID)
		if parentNode == nil {
			break
		}
		ancestors = append(ancestors, parentNode.Span)
		currentParentID = parentNode.Span.ParentSpanID
	}
	return ancestors
}

// GetDescendants returns all descendant spans for a given span ID
func (t *SpanTree) GetDescendants(spanID uuid.UUID) []*Span {
	node := t.GetSpanByID(spanID)
	if node == nil {
		return nil
	}

	var descendants []*Span
	var collect func(nodes []*SpanNode)
	collect = func(nodes []*SpanNode) {
		for _, n := range nodes {
			descendants = append(descendants, n.Span)
			if len(n.Children) > 0 {
				collect(n.Children)
			}
		}
	}
	collect(node.Children)
	return descendants
}
