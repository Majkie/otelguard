package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Agent represents an identified agent within a multi-agent system
type Agent struct {
	ID          uuid.UUID  `ch:"id" json:"id"`
	ProjectID   uuid.UUID  `ch:"project_id" json:"projectId"`
	TraceID     uuid.UUID  `ch:"trace_id" json:"traceId"`
	SpanID      uuid.UUID  `ch:"span_id" json:"spanId"`               // The span that represents this agent's execution
	ParentAgent *uuid.UUID `ch:"parent_agent_id" json:"parentAgentId,omitempty"` // Parent agent if delegated
	Name        string     `ch:"name" json:"name"`
	Type        string     `ch:"agent_type" json:"agentType"` // orchestrator, worker, tool_caller, planner, executor, custom
	Role        string     `ch:"role" json:"role"`            // Human-readable role description
	Model       *string    `ch:"model" json:"model,omitempty"`
	SystemPrompt *string   `ch:"system_prompt" json:"systemPrompt,omitempty"`
	StartTime   time.Time  `ch:"start_time" json:"startTime"`
	EndTime     time.Time  `ch:"end_time" json:"endTime"`
	LatencyMs   uint32     `ch:"latency_ms" json:"latencyMs"`
	TotalTokens uint32     `ch:"total_tokens" json:"totalTokens"`
	Cost        float64    `ch:"cost" json:"cost"`
	Status      string     `ch:"status" json:"status"` // running, success, error, timeout
	ErrorMessage *string   `ch:"error_message" json:"errorMessage,omitempty"`
	Metadata    string     `ch:"metadata" json:"metadata,omitempty"` // JSON metadata
	Tags        []string   `ch:"tags" json:"tags,omitempty"`
	CreatedAt   time.Time  `ch:"created_at" json:"createdAt"`
}

// AgentType constants
const (
	AgentTypeOrchestrator = "orchestrator" // Coordinates other agents
	AgentTypeWorker       = "worker"       // Performs specific tasks
	AgentTypeToolCaller   = "tool_caller"  // Specializes in calling external tools
	AgentTypePlanner      = "planner"      // Creates execution plans
	AgentTypeExecutor     = "executor"     // Executes planned steps
	AgentTypeReviewer     = "reviewer"     // Reviews and validates outputs
	AgentTypeCustom       = "custom"       // Custom agent type
)

// AgentStatus constants
const (
	AgentStatusRunning = "running"
	AgentStatusSuccess = "success"
	AgentStatusError   = "error"
	AgentStatusTimeout = "timeout"
)

// AgentRelationship represents a relationship between two agents
type AgentRelationship struct {
	ID             uuid.UUID `ch:"id" json:"id"`
	ProjectID      uuid.UUID `ch:"project_id" json:"projectId"`
	TraceID        uuid.UUID `ch:"trace_id" json:"traceId"`
	SourceAgentID  uuid.UUID `ch:"source_agent_id" json:"sourceAgentId"`
	TargetAgentID  uuid.UUID `ch:"target_agent_id" json:"targetAgentId"`
	RelationType   string    `ch:"relation_type" json:"relationType"` // delegates_to, calls, responds_to, supervises
	Timestamp      time.Time `ch:"timestamp" json:"timestamp"`
	Metadata       string    `ch:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time `ch:"created_at" json:"createdAt"`
}

// RelationType constants
const (
	RelationTypeDelegatesTo = "delegates_to" // Agent delegates work to another agent
	RelationTypeCalls       = "calls"        // Agent calls another agent
	RelationTypeRespondsTo  = "responds_to"  // Agent responds to another agent
	RelationTypeSupervises  = "supervises"   // Agent supervises another agent
	RelationTypeCollaborates = "collaborates" // Agents work together
)

// ToolCall represents a tool invocation by an agent
type ToolCall struct {
	ID          uuid.UUID  `ch:"id" json:"id"`
	ProjectID   uuid.UUID  `ch:"project_id" json:"projectId"`
	TraceID     uuid.UUID  `ch:"trace_id" json:"traceId"`
	SpanID      uuid.UUID  `ch:"span_id" json:"spanId"`
	AgentID     *uuid.UUID `ch:"agent_id" json:"agentId,omitempty"` // Optional link to agent
	Name        string     `ch:"name" json:"name"`
	Description string     `ch:"description" json:"description,omitempty"`
	Input       string     `ch:"input" json:"input"`   // JSON input parameters
	Output      string     `ch:"output" json:"output"` // JSON output
	StartTime   time.Time  `ch:"start_time" json:"startTime"`
	EndTime     time.Time  `ch:"end_time" json:"endTime"`
	LatencyMs   uint32     `ch:"latency_ms" json:"latencyMs"`
	Status      string     `ch:"status" json:"status"` // success, error, timeout
	ErrorMessage *string   `ch:"error_message" json:"errorMessage,omitempty"`
	RetryCount  int        `ch:"retry_count" json:"retryCount"`
	Metadata    string     `ch:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time  `ch:"created_at" json:"createdAt"`
}

// ToolCallStatus constants
const (
	ToolCallStatusSuccess = "success"
	ToolCallStatusError   = "error"
	ToolCallStatusTimeout = "timeout"
	ToolCallStatusPending = "pending"
)

// AgentMessage represents a message passed between agents
type AgentMessage struct {
	ID            uuid.UUID  `ch:"id" json:"id"`
	ProjectID     uuid.UUID  `ch:"project_id" json:"projectId"`
	TraceID       uuid.UUID  `ch:"trace_id" json:"traceId"`
	SpanID        *uuid.UUID `ch:"span_id" json:"spanId,omitempty"`
	FromAgentID   uuid.UUID  `ch:"from_agent_id" json:"fromAgentId"`
	ToAgentID     uuid.UUID  `ch:"to_agent_id" json:"toAgentId"`
	MessageType   string     `ch:"message_type" json:"messageType"` // request, response, notification, broadcast
	Role          string     `ch:"role" json:"role"`                // user, assistant, system, function, tool
	Content       string     `ch:"content" json:"content"`
	ContentType   string     `ch:"content_type" json:"contentType"` // text, json, tool_call, tool_result
	SequenceNum   int        `ch:"sequence_num" json:"sequenceNum"` // Order in conversation
	ParentMsgID   *uuid.UUID `ch:"parent_msg_id" json:"parentMsgId,omitempty"` // For threading
	TokenCount    uint32     `ch:"token_count" json:"tokenCount"`
	Timestamp     time.Time  `ch:"timestamp" json:"timestamp"`
	Metadata      string     `ch:"metadata" json:"metadata,omitempty"`
	CreatedAt     time.Time  `ch:"created_at" json:"createdAt"`
}

// MessageType constants
const (
	MessageTypeRequest      = "request"
	MessageTypeResponse     = "response"
	MessageTypeNotification = "notification"
	MessageTypeBroadcast    = "broadcast"
)

// MessageRole constants
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleSystem    = "system"
	MessageRoleFunction  = "function"
	MessageRoleTool      = "tool"
)

// MessageContentType constants
const (
	ContentTypeText       = "text"
	ContentTypeJSON       = "json"
	ContentTypeToolCall   = "tool_call"
	ContentTypeToolResult = "tool_result"
)

// AgentState represents a snapshot of an agent's state at a point in time
type AgentState struct {
	ID          uuid.UUID `ch:"id" json:"id"`
	ProjectID   uuid.UUID `ch:"project_id" json:"projectId"`
	TraceID     uuid.UUID `ch:"trace_id" json:"traceId"`
	AgentID     uuid.UUID `ch:"agent_id" json:"agentId"`
	SpanID      *uuid.UUID `ch:"span_id" json:"spanId,omitempty"`
	SequenceNum int       `ch:"sequence_num" json:"sequenceNum"` // Order of state snapshots
	State       string    `ch:"state" json:"state"`              // Current lifecycle state
	Variables   string    `ch:"variables" json:"variables"`      // JSON key-value store
	Memory      string    `ch:"memory" json:"memory"`            // JSON memory/context
	Plan        string    `ch:"plan" json:"plan,omitempty"`      // JSON current plan if any
	Reasoning   string    `ch:"reasoning" json:"reasoning,omitempty"` // Agent's reasoning at this point
	Timestamp   time.Time `ch:"timestamp" json:"timestamp"`
	Metadata    string    `ch:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time `ch:"created_at" json:"createdAt"`
}

// AgentLifecycleState constants
const (
	StateInitializing = "initializing"
	StatePlanning     = "planning"
	StateExecuting    = "executing"
	StateWaiting      = "waiting"
	StateThinking     = "thinking"
	StateCompleted    = "completed"
	StateFailed       = "failed"
)

// AgentHierarchy represents the hierarchical structure of agents in a trace
type AgentHierarchy struct {
	TraceID    uuid.UUID          `json:"traceId"`
	Agents     []*AgentNode       `json:"agents"`
	RootAgents []*AgentNode       `json:"rootAgents"`
	MaxDepth   int                `json:"maxDepth"`
}

// AgentNode represents an agent in the hierarchy tree
type AgentNode struct {
	Agent    *Agent       `json:"agent"`
	Children []*AgentNode `json:"children,omitempty"`
	Depth    int          `json:"depth"`
	Level    int          `json:"level"` // Same as depth but more intuitive
}

// BuildAgentHierarchy builds a hierarchy tree from a flat list of agents
func BuildAgentHierarchy(agents []*Agent) *AgentHierarchy {
	if len(agents) == 0 {
		return &AgentHierarchy{RootAgents: []*AgentNode{}, Agents: []*AgentNode{}}
	}

	hierarchy := &AgentHierarchy{
		TraceID: agents[0].TraceID,
	}

	// Create a map for quick lookup
	nodeMap := make(map[uuid.UUID]*AgentNode)
	for _, agent := range agents {
		node := &AgentNode{
			Agent:    agent,
			Children: []*AgentNode{},
		}
		nodeMap[agent.ID] = node
		hierarchy.Agents = append(hierarchy.Agents, node)
	}

	// Build tree relationships
	var rootAgents []*AgentNode
	for _, agent := range agents {
		node := nodeMap[agent.ID]
		if agent.ParentAgent == nil {
			// This is a root agent
			rootAgents = append(rootAgents, node)
		} else {
			// Find parent and add as child
			if parentNode, ok := nodeMap[*agent.ParentAgent]; ok {
				parentNode.Children = append(parentNode.Children, node)
			} else {
				// Parent not found, treat as root
				rootAgents = append(rootAgents, node)
			}
		}
	}

	hierarchy.RootAgents = rootAgents

	// Calculate depths
	hierarchy.MaxDepth = calculateAgentDepths(rootAgents, 0)

	return hierarchy
}

// calculateAgentDepths recursively sets depths and returns max depth
func calculateAgentDepths(nodes []*AgentNode, depth int) int {
	maxDepth := depth
	for _, node := range nodes {
		node.Depth = depth
		node.Level = depth
		if len(node.Children) > 0 {
			childMaxDepth := calculateAgentDepths(node.Children, depth+1)
			if childMaxDepth > maxDepth {
				maxDepth = childMaxDepth
			}
		}
	}
	return maxDepth
}

// GetAgentByID finds an agent in the hierarchy by ID
func (h *AgentHierarchy) GetAgentByID(agentID uuid.UUID) *AgentNode {
	for _, node := range h.Agents {
		if node.Agent.ID == agentID {
			return node
		}
	}
	return nil
}

// GetDescendantAgents returns all descendant agents for a given agent ID
func (h *AgentHierarchy) GetDescendantAgents(agentID uuid.UUID) []*Agent {
	node := h.GetAgentByID(agentID)
	if node == nil {
		return nil
	}

	var descendants []*Agent
	var collect func(nodes []*AgentNode)
	collect = func(nodes []*AgentNode) {
		for _, n := range nodes {
			descendants = append(descendants, n.Agent)
			if len(n.Children) > 0 {
				collect(n.Children)
			}
		}
	}
	collect(node.Children)
	return descendants
}

// DetectAgentHierarchyFromSpans analyzes spans to detect agent hierarchy
func DetectAgentHierarchyFromSpans(spans []*Span) []*Agent {
	var agents []*Agent
	spanToAgent := make(map[uuid.UUID]*Agent)

	// First pass: identify all agent spans
	for _, span := range spans {
		if span.Type == SpanTypeAgent {
			agent := &Agent{
				ID:        uuid.New(),
				ProjectID: span.ProjectID,
				TraceID:   span.TraceID,
				SpanID:    span.ID,
				Name:      span.Name,
				Type:      detectAgentType(span),
				StartTime: span.StartTime,
				EndTime:   span.EndTime,
				LatencyMs: span.LatencyMs,
				TotalTokens: span.Tokens,
				Cost:      span.Cost,
				Model:     span.Model,
				Status:    span.Status,
				ErrorMessage: span.ErrorMessage,
				Metadata:  span.Metadata,
			}
			agents = append(agents, agent)
			spanToAgent[span.ID] = agent
		}
	}

	// Second pass: establish parent-child relationships based on span hierarchy
	for _, span := range spans {
		if span.Type == SpanTypeAgent && span.ParentSpanID != nil {
			agent := spanToAgent[span.ID]
			// Walk up the span tree to find parent agent
			parentSpanID := span.ParentSpanID
			for parentSpanID != nil {
				if parentAgent, ok := spanToAgent[*parentSpanID]; ok {
					agent.ParentAgent = &parentAgent.ID
					break
				}
				// Find the parent span and continue up
				for _, s := range spans {
					if s.ID == *parentSpanID {
						parentSpanID = s.ParentSpanID
						break
					}
				}
			}
		}
	}

	return agents
}

// detectAgentType attempts to detect the agent type from span metadata
func detectAgentType(span *Span) string {
	if span.Metadata == "" {
		return AgentTypeCustom
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(span.Metadata), &meta); err != nil {
		return AgentTypeCustom
	}

	// Check for explicit agent type in metadata
	if agentType, ok := meta["agent_type"].(string); ok {
		return agentType
	}

	// Heuristics based on name patterns
	name := span.Name
	switch {
	case containsAny(name, "orchestrat", "coordinator", "supervisor"):
		return AgentTypeOrchestrator
	case containsAny(name, "planner", "planning"):
		return AgentTypePlanner
	case containsAny(name, "executor", "execute"):
		return AgentTypeExecutor
	case containsAny(name, "tool", "function"):
		return AgentTypeToolCaller
	case containsAny(name, "review", "validate", "check"):
		return AgentTypeReviewer
	case containsAny(name, "worker"):
		return AgentTypeWorker
	default:
		return AgentTypeCustom
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains is a simple case-insensitive contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > len(substr) &&
		 (stringContains(toLower(s), toLower(substr))))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ExtractToolCallsFromSpans extracts tool calls from spans
func ExtractToolCallsFromSpans(spans []*Span, agentMap map[uuid.UUID]*Agent) []*ToolCall {
	var toolCalls []*ToolCall

	for _, span := range spans {
		if span.Type == SpanTypeTool {
			toolCall := &ToolCall{
				ID:        uuid.New(),
				ProjectID: span.ProjectID,
				TraceID:   span.TraceID,
				SpanID:    span.ID,
				Name:      span.Name,
				Input:     span.Input,
				Output:    span.Output,
				StartTime: span.StartTime,
				EndTime:   span.EndTime,
				LatencyMs: span.LatencyMs,
				Status:    span.Status,
				ErrorMessage: span.ErrorMessage,
				Metadata:  span.Metadata,
			}

			// Try to find the parent agent
			if span.ParentSpanID != nil {
				for _, a := range agentMap {
					if a.SpanID == *span.ParentSpanID {
						toolCall.AgentID = &a.ID
						break
					}
				}
			}

			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}
