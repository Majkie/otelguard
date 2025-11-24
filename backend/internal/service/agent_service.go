package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// AgentService handles agent-related business logic
type AgentService struct {
	agentRepo *clickhouse.AgentRepository
	traceRepo *clickhouse.TraceRepository
	logger    *zap.Logger
}

// NewAgentService creates a new agent service
func NewAgentService(
	agentRepo *clickhouse.AgentRepository,
	traceRepo *clickhouse.TraceRepository,
	logger *zap.Logger,
) *AgentService {
	return &AgentService{
		agentRepo: agentRepo,
		traceRepo: traceRepo,
		logger:    logger,
	}
}

// CreateAgent creates a new agent
func (s *AgentService) CreateAgent(ctx context.Context, agent *domain.Agent) error {
	if agent.ID == uuid.Nil {
		agent.ID = uuid.New()
	}
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = time.Now()
	}

	return s.agentRepo.InsertAgent(ctx, agent)
}

// CreateAgentsBatch creates multiple agents in a batch
func (s *AgentService) CreateAgentsBatch(ctx context.Context, agents []*domain.Agent) error {
	now := time.Now()
	for _, agent := range agents {
		if agent.ID == uuid.Nil {
			agent.ID = uuid.New()
		}
		if agent.CreatedAt.IsZero() {
			agent.CreatedAt = now
		}
	}

	return s.agentRepo.InsertAgents(ctx, agents)
}

// GetAgent retrieves an agent by ID
func (s *AgentService) GetAgent(ctx context.Context, projectID, agentID string) (*domain.Agent, error) {
	return s.agentRepo.GetAgentByID(ctx, projectID, agentID)
}

// GetAgentsByTrace retrieves all agents for a trace
func (s *AgentService) GetAgentsByTrace(ctx context.Context, projectID, traceID string) ([]*domain.Agent, error) {
	return s.agentRepo.GetAgentsByTraceID(ctx, projectID, traceID)
}

// ListAgentsOptions contains options for listing agents
type ListAgentsOptions struct {
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

// ListAgents lists agents with filtering and pagination
func (s *AgentService) ListAgents(ctx context.Context, opts *ListAgentsOptions) ([]*domain.Agent, int, error) {
	queryOpts := &clickhouse.AgentQueryOptions{
		ProjectID:   opts.ProjectID,
		TraceID:     opts.TraceID,
		AgentType:   opts.AgentType,
		Status:      opts.Status,
		ParentAgent: opts.ParentAgent,
		StartTime:   opts.StartTime,
		EndTime:     opts.EndTime,
		SortBy:      opts.SortBy,
		SortOrder:   opts.SortOrder,
		Limit:       opts.Limit,
		Offset:      opts.Offset,
	}

	return s.agentRepo.QueryAgents(ctx, queryOpts)
}

// GetAgentHierarchy returns the agent hierarchy for a trace
func (s *AgentService) GetAgentHierarchy(ctx context.Context, projectID, traceID string) (*domain.AgentHierarchy, error) {
	agents, err := s.agentRepo.GetAgentsByTraceID(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	return domain.BuildAgentHierarchy(agents), nil
}

// DetectAndStoreAgents detects agents from spans and stores them
func (s *AgentService) DetectAndStoreAgents(ctx context.Context, projectID, traceID string) ([]*domain.Agent, error) {
	// Get spans for the trace
	spans, err := s.traceRepo.GetSpans(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	// Detect agents from spans
	agents := domain.DetectAgentHierarchyFromSpans(spans)

	// Set project ID and store
	for _, agent := range agents {
		agent.ProjectID, _ = uuid.Parse(projectID)
		agent.CreatedAt = time.Now()
	}

	if len(agents) > 0 {
		if err := s.agentRepo.InsertAgents(ctx, agents); err != nil {
			return nil, err
		}
	}

	return agents, nil
}

// CreateAgentRelationship creates a relationship between agents
func (s *AgentService) CreateAgentRelationship(ctx context.Context, rel *domain.AgentRelationship) error {
	if rel.ID == uuid.Nil {
		rel.ID = uuid.New()
	}
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now()
	}

	return s.agentRepo.InsertAgentRelationship(ctx, rel)
}

// GetAgentRelationships retrieves relationships for a trace
func (s *AgentService) GetAgentRelationships(ctx context.Context, projectID, traceID string) ([]*domain.AgentRelationship, error) {
	return s.agentRepo.GetAgentRelationships(ctx, projectID, traceID)
}

// CreateToolCall creates a tool call record
func (s *AgentService) CreateToolCall(ctx context.Context, tc *domain.ToolCall) error {
	if tc.ID == uuid.Nil {
		tc.ID = uuid.New()
	}
	if tc.CreatedAt.IsZero() {
		tc.CreatedAt = time.Now()
	}

	return s.agentRepo.InsertToolCall(ctx, tc)
}

// CreateToolCallsBatch creates multiple tool calls in a batch
func (s *AgentService) CreateToolCallsBatch(ctx context.Context, toolCalls []*domain.ToolCall) error {
	now := time.Now()
	for _, tc := range toolCalls {
		if tc.ID == uuid.Nil {
			tc.ID = uuid.New()
		}
		if tc.CreatedAt.IsZero() {
			tc.CreatedAt = now
		}
	}

	return s.agentRepo.InsertToolCalls(ctx, toolCalls)
}

// GetToolCallsByTrace retrieves tool calls for a trace
func (s *AgentService) GetToolCallsByTrace(ctx context.Context, projectID, traceID string) ([]*domain.ToolCall, error) {
	return s.agentRepo.GetToolCallsByTraceID(ctx, projectID, traceID)
}

// GetToolCallsByAgent retrieves tool calls for an agent
func (s *AgentService) GetToolCallsByAgent(ctx context.Context, projectID, agentID string) ([]*domain.ToolCall, error) {
	return s.agentRepo.GetToolCallsByAgentID(ctx, projectID, agentID)
}

// DetectAndStoreToolCalls detects tool calls from spans and stores them
func (s *AgentService) DetectAndStoreToolCalls(ctx context.Context, projectID, traceID string) ([]*domain.ToolCall, error) {
	// Get spans for the trace
	spans, err := s.traceRepo.GetSpans(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	// Get agents for mapping
	agents, err := s.agentRepo.GetAgentsByTraceID(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	agentMap := make(map[uuid.UUID]*domain.Agent)
	for _, agent := range agents {
		agentMap[agent.SpanID] = agent
	}

	// Extract tool calls
	toolCalls := domain.ExtractToolCallsFromSpans(spans, agentMap)

	// Set project ID and store
	projectUUID, _ := uuid.Parse(projectID)
	for _, tc := range toolCalls {
		tc.ProjectID = projectUUID
		tc.CreatedAt = time.Now()
	}

	if len(toolCalls) > 0 {
		if err := s.agentRepo.InsertToolCalls(ctx, toolCalls); err != nil {
			return nil, err
		}
	}

	return toolCalls, nil
}

// CreateAgentMessage creates an agent message
func (s *AgentService) CreateAgentMessage(ctx context.Context, msg *domain.AgentMessage) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	return s.agentRepo.InsertAgentMessage(ctx, msg)
}

// GetAgentMessages retrieves messages for a trace
func (s *AgentService) GetAgentMessages(ctx context.Context, projectID, traceID string) ([]*domain.AgentMessage, error) {
	return s.agentRepo.GetAgentMessagesByTraceID(ctx, projectID, traceID)
}

// CreateAgentState creates an agent state snapshot
func (s *AgentService) CreateAgentState(ctx context.Context, state *domain.AgentState) error {
	if state.ID == uuid.Nil {
		state.ID = uuid.New()
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}

	return s.agentRepo.InsertAgentState(ctx, state)
}

// GetAgentStates retrieves state snapshots for an agent
func (s *AgentService) GetAgentStates(ctx context.Context, projectID, agentID string) ([]*domain.AgentState, error) {
	return s.agentRepo.GetAgentStatesByAgentID(ctx, projectID, agentID)
}

// GetAgentStatistics retrieves aggregated agent statistics
func (s *AgentService) GetAgentStatistics(ctx context.Context, projectID string, startTime, endTime time.Time) (*clickhouse.AgentStatistics, error) {
	return s.agentRepo.GetAgentStatistics(ctx, projectID, startTime, endTime)
}

// GetToolCallStatistics retrieves aggregated tool call statistics
func (s *AgentService) GetToolCallStatistics(ctx context.Context, projectID string, startTime, endTime time.Time) ([]*clickhouse.ToolCallStatistics, error) {
	return s.agentRepo.GetToolCallStatistics(ctx, projectID, startTime, endTime)
}

// BuildAgentGraph builds an agent graph for visualization
func (s *AgentService) BuildAgentGraph(ctx context.Context, projectID, traceID string) (*domain.AgentGraph, error) {
	// Get spans for the trace
	spans, err := s.traceRepo.GetSpans(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	projectUUID, _ := uuid.Parse(projectID)
	traceUUID, _ := uuid.Parse(traceID)

	// Build graph from spans
	graph := domain.BuildAgentGraph(spans, traceUUID, projectUUID)

	return graph, nil
}

// BuildAgentGraphFromAgents builds a graph from stored agents
func (s *AgentService) BuildAgentGraphFromAgents(ctx context.Context, projectID, traceID string) (*domain.AgentGraph, error) {
	// Get agents
	agents, err := s.agentRepo.GetAgentsByTraceID(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	// Get relationships
	relationships, err := s.agentRepo.GetAgentRelationships(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	// Get tool calls
	toolCalls, err := s.agentRepo.GetToolCallsByTraceID(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	projectUUID, _ := uuid.Parse(projectID)
	traceUUID, _ := uuid.Parse(traceID)

	// Build graph
	graph := &domain.AgentGraph{
		TraceID:   traceUUID,
		ProjectID: projectUUID,
		Nodes:     make([]*domain.GraphNode, 0),
		Edges:     make([]*domain.GraphEdge, 0),
		NodeMap:   make(map[uuid.UUID]*domain.GraphNode),
		EdgeMap:   make(map[string]*domain.GraphEdge),
		CreatedAt: time.Now(),
	}

	// Add agent nodes
	for _, agent := range agents {
		node := &domain.GraphNode{
			ID:        agent.ID,
			Type:      domain.NodeTypeAgent,
			Label:     agent.Name,
			AgentID:   &agent.ID,
			SpanID:    &agent.SpanID,
			StartTime: agent.StartTime,
			EndTime:   agent.EndTime,
			LatencyMs: agent.LatencyMs,
			Status:    agent.Status,
			Tokens:    agent.TotalTokens,
			Cost:      agent.Cost,
			Model:     agent.Model,
			Metadata:  agent.Metadata,
		}
		graph.Nodes = append(graph.Nodes, node)
		graph.NodeMap[node.ID] = node
	}

	// Add tool call nodes
	for _, tc := range toolCalls {
		node := &domain.GraphNode{
			ID:         tc.ID,
			Type:       domain.NodeTypeTool,
			Label:      tc.Name,
			ToolCallID: &tc.ID,
			SpanID:     &tc.SpanID,
			StartTime:  tc.StartTime,
			EndTime:    tc.EndTime,
			LatencyMs:  tc.LatencyMs,
			Status:     tc.Status,
			Metadata:   tc.Metadata,
		}
		graph.Nodes = append(graph.Nodes, node)
		graph.NodeMap[node.ID] = node

		// Add edge from agent to tool
		if tc.AgentID != nil {
			edge := &domain.GraphEdge{
				ID:     uuid.New(),
				Source: *tc.AgentID,
				Target: tc.ID,
				Type:   domain.EdgeTypeToolCall,
				Label:  tc.Name,
			}
			graph.Edges = append(graph.Edges, edge)
			graph.EdgeMap[edge.Source.String()+"-"+edge.Target.String()] = edge
		}
	}

	// Add relationship edges
	for _, rel := range relationships {
		edgeType := domain.EdgeTypeSequence
		switch rel.RelationType {
		case domain.RelationTypeDelegatesTo:
			edgeType = domain.EdgeTypeDelegation
		case domain.RelationTypeCalls:
			edgeType = domain.EdgeTypeSequence
		case domain.RelationTypeRespondsTo:
			edgeType = domain.EdgeTypeReturn
		}

		edge := &domain.GraphEdge{
			ID:     rel.ID,
			Source: rel.SourceAgentID,
			Target: rel.TargetAgentID,
			Type:   edgeType,
		}
		edgeKey := edge.Source.String() + "-" + edge.Target.String()
		if _, exists := graph.EdgeMap[edgeKey]; !exists {
			graph.Edges = append(graph.Edges, edge)
			graph.EdgeMap[edgeKey] = edge
		}
	}

	// Calculate metadata
	graph.Metadata = &domain.GraphMetadata{
		TotalNodes: len(graph.Nodes),
		TotalEdges: len(graph.Edges),
	}

	return graph, nil
}

// GetSimplifiedGraph returns a simplified version of the agent graph
func (s *AgentService) GetSimplifiedGraph(ctx context.Context, projectID, traceID string, maxNodes int) (*domain.AgentGraph, error) {
	graph, err := s.BuildAgentGraph(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	return graph.SimplifyGraph(maxNodes), nil
}

// GetSubgraph returns a subgraph starting from a specific node
func (s *AgentService) GetSubgraph(ctx context.Context, projectID, traceID, nodeID string, maxDepth int) (*domain.AgentGraph, error) {
	graph, err := s.BuildAgentGraph(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	nodeUUID, err := uuid.Parse(nodeID)
	if err != nil {
		return nil, err
	}

	return graph.GetSubgraph(nodeUUID, maxDepth), nil
}
