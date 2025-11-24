package domain

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// AgentGraph represents a directed graph of agent interactions
type AgentGraph struct {
	TraceID   uuid.UUID                `json:"traceId"`
	ProjectID uuid.UUID                `json:"projectId"`
	Nodes     []*GraphNode             `json:"nodes"`
	Edges     []*GraphEdge             `json:"edges"`
	NodeMap   map[uuid.UUID]*GraphNode `json:"-"` // Internal lookup
	EdgeMap   map[string]*GraphEdge    `json:"-"` // Internal lookup (source-target key)
	Metadata  *GraphMetadata           `json:"metadata"`
	CreatedAt time.Time                `json:"createdAt"`
}

// GraphNode represents a node in the agent graph
type GraphNode struct {
	ID            uuid.UUID     `json:"id"`
	Type          NodeType      `json:"type"`
	Label         string        `json:"label"`
	AgentID       *uuid.UUID    `json:"agentId,omitempty"`
	SpanID        *uuid.UUID    `json:"spanId,omitempty"`
	ToolCallID    *uuid.UUID    `json:"toolCallId,omitempty"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`
	LatencyMs     uint32        `json:"latencyMs"`
	Status        string        `json:"status"`
	Tokens        uint32        `json:"tokens,omitempty"`
	Cost          float64       `json:"cost,omitempty"`
	Model         *string       `json:"model,omitempty"`
	Depth         int           `json:"depth"`         // Depth in the graph
	ParallelGroup int           `json:"parallelGroup"` // Group of parallel nodes
	Metadata      string        `json:"metadata,omitempty"`
	Position      *NodePosition `json:"position,omitempty"` // For visualization
}

// NodePosition represents the visual position of a node
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeType represents the type of graph node
type NodeType string

const (
	NodeTypeAgent     NodeType = "agent"
	NodeTypeTool      NodeType = "tool"
	NodeTypeLLM       NodeType = "llm"
	NodeTypeRetrieval NodeType = "retrieval"
	NodeTypeEmbedding NodeType = "embedding"
	NodeTypeMessage   NodeType = "message"
	NodeTypeStart     NodeType = "start"
	NodeTypeEnd       NodeType = "end"
	NodeTypeCustom    NodeType = "custom"
)

// GraphEdge represents an edge connecting two nodes
type GraphEdge struct {
	ID        uuid.UUID  `json:"id"`
	Source    uuid.UUID  `json:"source"` // Source node ID
	Target    uuid.UUID  `json:"target"` // Target node ID
	Type      EdgeType   `json:"type"`
	Label     string     `json:"label,omitempty"`
	Weight    float64    `json:"weight,omitempty"` // For weighted graphs
	Order     int        `json:"order"`            // Temporal order
	MessageID *uuid.UUID `json:"messageId,omitempty"`
	LatencyMs uint32     `json:"latencyMs,omitempty"`
	Metadata  string     `json:"metadata,omitempty"`
}

// EdgeType represents the type of graph edge
type EdgeType string

const (
	EdgeTypeDelegation EdgeType = "delegation" // Agent delegates to another
	EdgeTypeToolCall   EdgeType = "tool_call"  // Agent calls a tool
	EdgeTypeLLMCall    EdgeType = "llm_call"   // Agent makes an LLM call
	EdgeTypeMessage    EdgeType = "message"    // Message between agents
	EdgeTypeSequence   EdgeType = "sequence"   // Sequential execution
	EdgeTypeParallel   EdgeType = "parallel"   // Parallel execution
	EdgeTypeReturn     EdgeType = "return"     // Return from delegation
	EdgeTypeCustom     EdgeType = "custom"
)

// GraphMetadata contains metadata about the graph
type GraphMetadata struct {
	TotalNodes     int              `json:"totalNodes"`
	TotalEdges     int              `json:"totalEdges"`
	MaxDepth       int              `json:"maxDepth"`
	MaxParallelism int              `json:"maxParallelism"`
	HasCycles      bool             `json:"hasCycles"`
	CycleNodes     []uuid.UUID      `json:"cycleNodes,omitempty"`
	ParallelGroups int              `json:"parallelGroups"`
	TotalLatencyMs uint32           `json:"totalLatencyMs"`
	CriticalPath   []uuid.UUID      `json:"criticalPath,omitempty"`
	CriticalPathMs uint32           `json:"criticalPathMs"`
	Bottlenecks    []*Bottleneck    `json:"bottlenecks,omitempty"`
	ExecutionLanes []*ExecutionLane `json:"executionLanes,omitempty"`
}

// Bottleneck represents a performance bottleneck in the graph
type Bottleneck struct {
	NodeID     uuid.UUID `json:"nodeId"`
	LatencyMs  uint32    `json:"latencyMs"`
	Percentage float64   `json:"percentage"` // Percentage of total latency
	Reason     string    `json:"reason"`
}

// ExecutionLane represents a lane of parallel execution
type ExecutionLane struct {
	LaneID    int         `json:"laneId"`
	Nodes     []uuid.UUID `json:"nodes"`
	StartTime time.Time   `json:"startTime"`
	EndTime   time.Time   `json:"endTime"`
	LatencyMs uint32      `json:"latencyMs"`
}

// BuildAgentGraph constructs a graph from spans
func BuildAgentGraph(spans []*Span, traceID, projectID uuid.UUID) *AgentGraph {
	graph := &AgentGraph{
		TraceID:   traceID,
		ProjectID: projectID,
		Nodes:     make([]*GraphNode, 0),
		Edges:     make([]*GraphEdge, 0),
		NodeMap:   make(map[uuid.UUID]*GraphNode),
		EdgeMap:   make(map[string]*GraphEdge),
		CreatedAt: time.Now(),
	}

	if len(spans) == 0 {
		graph.Metadata = &GraphMetadata{}
		return graph
	}

	// Extract nodes from spans
	graph.extractNodes(spans)

	// Extract edges based on relationships
	graph.extractEdges(spans)

	// Build temporal ordering
	graph.buildTemporalOrdering()

	// Detect parallel execution
	graph.detectParallelExecution(spans)

	// Detect cycles
	graph.detectCycles()

	// Calculate metadata
	graph.calculateMetadata(spans)

	return graph
}

// extractNodes creates graph nodes from spans
func (g *AgentGraph) extractNodes(spans []*Span) {
	for _, span := range spans {
		node := &GraphNode{
			ID:        span.ID,
			Type:      spanTypeToNodeType(span.Type),
			Label:     span.Name,
			SpanID:    &span.ID,
			StartTime: span.StartTime,
			EndTime:   span.EndTime,
			LatencyMs: span.LatencyMs,
			Status:    span.Status,
			Tokens:    span.Tokens,
			Cost:      span.Cost,
			Model:     span.Model,
			Metadata:  span.Metadata,
		}
		g.Nodes = append(g.Nodes, node)
		g.NodeMap[node.ID] = node
	}
}

// spanTypeToNodeType converts span type to node type
func spanTypeToNodeType(spanType string) NodeType {
	switch spanType {
	case SpanTypeAgent:
		return NodeTypeAgent
	case SpanTypeTool:
		return NodeTypeTool
	case SpanTypeLLM:
		return NodeTypeLLM
	case SpanTypeRetrieval:
		return NodeTypeRetrieval
	case SpanTypeEmbedding:
		return NodeTypeEmbedding
	default:
		return NodeTypeCustom
	}
}

// extractEdges creates edges based on span relationships
func (g *AgentGraph) extractEdges(spans []*Span) {
	spanMap := make(map[uuid.UUID]*Span)
	for _, span := range spans {
		spanMap[span.ID] = span
	}

	// Create parent-child edges
	for _, span := range spans {
		if span.ParentSpanID != nil {
			parentSpan, exists := spanMap[*span.ParentSpanID]
			if exists {
				edge := &GraphEdge{
					ID:     uuid.New(),
					Source: *span.ParentSpanID,
					Target: span.ID,
					Type:   determineEdgeType(parentSpan, span),
					Label:  span.Name,
				}
				edgeKey := edge.Source.String() + "-" + edge.Target.String()
				if _, exists := g.EdgeMap[edgeKey]; !exists {
					g.Edges = append(g.Edges, edge)
					g.EdgeMap[edgeKey] = edge
				}
			}
		}
	}

	// Create sibling sequence edges (spans with same parent, ordered by time)
	parentChildren := make(map[uuid.UUID][]*Span)
	for _, span := range spans {
		if span.ParentSpanID != nil {
			parentChildren[*span.ParentSpanID] = append(parentChildren[*span.ParentSpanID], span)
		}
	}

	for _, children := range parentChildren {
		if len(children) > 1 {
			// Sort by start time
			sort.Slice(children, func(i, j int) bool {
				return children[i].StartTime.Before(children[j].StartTime)
			})

			// Create sequence edges for non-overlapping spans
			for i := 0; i < len(children)-1; i++ {
				current := children[i]
				next := children[i+1]

				// Check if they don't overlap (sequential)
				if current.EndTime.Before(next.StartTime) || current.EndTime.Equal(next.StartTime) {
					edge := &GraphEdge{
						ID:        uuid.New(),
						Source:    current.ID,
						Target:    next.ID,
						Type:      EdgeTypeSequence,
						Order:     i,
						LatencyMs: uint32(next.StartTime.Sub(current.EndTime).Milliseconds()),
					}
					edgeKey := edge.Source.String() + "-" + edge.Target.String()
					if _, exists := g.EdgeMap[edgeKey]; !exists {
						g.Edges = append(g.Edges, edge)
						g.EdgeMap[edgeKey] = edge
					}
				}
			}
		}
	}
}

// determineEdgeType determines the edge type based on span types
func determineEdgeType(parent, child *Span) EdgeType {
	if parent.Type == SpanTypeAgent && child.Type == SpanTypeAgent {
		return EdgeTypeDelegation
	}
	if child.Type == SpanTypeTool {
		return EdgeTypeToolCall
	}
	if child.Type == SpanTypeLLM {
		return EdgeTypeLLMCall
	}
	return EdgeTypeSequence
}

// buildTemporalOrdering assigns order to edges based on time
func (g *AgentGraph) buildTemporalOrdering() {
	// Sort edges by source node's start time, then target node's start time
	sort.Slice(g.Edges, func(i, j int) bool {
		srcI, okI := g.NodeMap[g.Edges[i].Source]
		srcJ, okJ := g.NodeMap[g.Edges[j].Source]
		if !okI || !okJ {
			return false
		}
		if srcI.StartTime.Equal(srcJ.StartTime) {
			tgtI, okTI := g.NodeMap[g.Edges[i].Target]
			tgtJ, okTJ := g.NodeMap[g.Edges[j].Target]
			if !okTI || !okTJ {
				return false
			}
			return tgtI.StartTime.Before(tgtJ.StartTime)
		}
		return srcI.StartTime.Before(srcJ.StartTime)
	})

	// Assign order numbers
	for i, edge := range g.Edges {
		edge.Order = i
	}
}

// detectParallelExecution identifies nodes that execute in parallel
func (g *AgentGraph) detectParallelExecution(spans []*Span) {
	if len(spans) == 0 {
		return
	}

	// Group spans by parent
	parentChildren := make(map[uuid.UUID][]*Span)
	for _, span := range spans {
		if span.ParentSpanID != nil {
			parentChildren[*span.ParentSpanID] = append(parentChildren[*span.ParentSpanID], span)
		}
	}

	parallelGroupCounter := 0

	for _, children := range parentChildren {
		if len(children) < 2 {
			continue
		}

		// Sort by start time
		sort.Slice(children, func(i, j int) bool {
			return children[i].StartTime.Before(children[j].StartTime)
		})

		// Find overlapping groups
		groups := findOverlappingGroups(children)

		for _, group := range groups {
			if len(group) > 1 {
				parallelGroupCounter++
				for _, span := range group {
					if node, ok := g.NodeMap[span.ID]; ok {
						node.ParallelGroup = parallelGroupCounter
					}
				}

				// Add parallel edges between nodes in the same group
				for i := 0; i < len(group); i++ {
					for j := i + 1; j < len(group); j++ {
						edgeKey := group[i].ID.String() + "-" + group[j].ID.String() + "-parallel"
						if _, exists := g.EdgeMap[edgeKey]; !exists {
							edge := &GraphEdge{
								ID:     uuid.New(),
								Source: group[i].ID,
								Target: group[j].ID,
								Type:   EdgeTypeParallel,
								Label:  "parallel",
							}
							// Note: parallel edges are informational, stored with special key
							g.EdgeMap[edgeKey] = edge
						}
					}
				}
			}
		}
	}
}

// findOverlappingGroups groups spans that have overlapping execution times
func findOverlappingGroups(spans []*Span) [][]*Span {
	if len(spans) == 0 {
		return nil
	}

	var groups [][]*Span
	used := make(map[uuid.UUID]bool)

	for i, span := range spans {
		if used[span.ID] {
			continue
		}

		group := []*Span{span}
		used[span.ID] = true

		for j := i + 1; j < len(spans); j++ {
			other := spans[j]
			if used[other.ID] {
				continue
			}

			// Check if any span in the group overlaps with this span
			overlaps := false
			for _, g := range group {
				if spansOverlap(g, other) {
					overlaps = true
					break
				}
			}

			if overlaps {
				group = append(group, other)
				used[other.ID] = true
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// spansOverlap checks if two spans have overlapping execution times
func spansOverlap(a, b *Span) bool {
	// Two spans overlap if one starts before the other ends
	return a.StartTime.Before(b.EndTime) && b.StartTime.Before(a.EndTime)
}

// detectCycles uses DFS to detect cycles in the graph
func (g *AgentGraph) detectCycles() {
	if len(g.Nodes) == 0 {
		return
	}

	visited := make(map[uuid.UUID]bool)
	recStack := make(map[uuid.UUID]bool)
	var cycleNodes []uuid.UUID

	// Build adjacency list
	adjacency := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range g.Edges {
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	var dfs func(nodeID uuid.UUID) bool
	dfs = func(nodeID uuid.UUID) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, neighbor := range adjacency[nodeID] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					cycleNodes = append(cycleNodes, nodeID)
					return true
				}
			} else if recStack[neighbor] {
				cycleNodes = append(cycleNodes, nodeID)
				cycleNodes = append(cycleNodes, neighbor)
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	hasCycles := false
	for _, node := range g.Nodes {
		if !visited[node.ID] {
			if dfs(node.ID) {
				hasCycles = true
			}
		}
	}

	if g.Metadata == nil {
		g.Metadata = &GraphMetadata{}
	}
	g.Metadata.HasCycles = hasCycles
	g.Metadata.CycleNodes = cycleNodes
}

// calculateMetadata computes graph metadata
func (g *AgentGraph) calculateMetadata(spans []*Span) {
	if g.Metadata == nil {
		g.Metadata = &GraphMetadata{}
	}

	g.Metadata.TotalNodes = len(g.Nodes)
	g.Metadata.TotalEdges = len(g.Edges)

	// Calculate max depth using BFS from root nodes
	g.calculateDepths()
	g.Metadata.MaxDepth = 0
	for _, node := range g.Nodes {
		if node.Depth > g.Metadata.MaxDepth {
			g.Metadata.MaxDepth = node.Depth
		}
	}

	// Count parallel groups
	parallelGroups := make(map[int]bool)
	for _, node := range g.Nodes {
		if node.ParallelGroup > 0 {
			parallelGroups[node.ParallelGroup] = true
		}
	}
	g.Metadata.ParallelGroups = len(parallelGroups)

	// Calculate max parallelism
	g.calculateMaxParallelism(spans)

	// Calculate total latency (from first start to last end)
	g.calculateTotalLatency()

	// Find critical path
	g.findCriticalPath()

	// Identify bottlenecks
	g.identifyBottlenecks()

	// Build execution lanes
	g.buildExecutionLanes(spans)
}

// calculateDepths calculates the depth of each node in the graph
func (g *AgentGraph) calculateDepths() {
	// Find root nodes (nodes with no incoming edges)
	hasIncoming := make(map[uuid.UUID]bool)
	for _, edge := range g.Edges {
		hasIncoming[edge.Target] = true
	}

	// Build adjacency list
	adjacency := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range g.Edges {
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	// BFS from roots
	queue := make([]uuid.UUID, 0)
	for _, node := range g.Nodes {
		if !hasIncoming[node.ID] {
			node.Depth = 0
			queue = append(queue, node.ID)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentNode := g.NodeMap[current]

		for _, neighborID := range adjacency[current] {
			neighbor := g.NodeMap[neighborID]
			newDepth := currentNode.Depth + 1
			if neighbor.Depth == 0 || newDepth > neighbor.Depth {
				neighbor.Depth = newDepth
				queue = append(queue, neighborID)
			}
		}
	}
}

// calculateMaxParallelism finds the maximum number of concurrent executions
func (g *AgentGraph) calculateMaxParallelism(spans []*Span) {
	if len(spans) == 0 {
		g.Metadata.MaxParallelism = 0
		return
	}

	// Create events for start and end times
	type event struct {
		time  time.Time
		delta int // +1 for start, -1 for end
	}

	events := make([]event, 0, len(spans)*2)
	for _, span := range spans {
		events = append(events, event{span.StartTime, 1})
		events = append(events, event{span.EndTime, -1})
	}

	// Sort events by time
	sort.Slice(events, func(i, j int) bool {
		if events[i].time.Equal(events[j].time) {
			// Process ends before starts at the same time
			return events[i].delta < events[j].delta
		}
		return events[i].time.Before(events[j].time)
	})

	maxParallel := 0
	current := 0
	for _, e := range events {
		current += e.delta
		if current > maxParallel {
			maxParallel = current
		}
	}

	g.Metadata.MaxParallelism = maxParallel
}

// calculateTotalLatency calculates the total wall-clock latency
func (g *AgentGraph) calculateTotalLatency() {
	if len(g.Nodes) == 0 {
		g.Metadata.TotalLatencyMs = 0
		return
	}

	var minStart, maxEnd time.Time
	first := true

	for _, node := range g.Nodes {
		if first {
			minStart = node.StartTime
			maxEnd = node.EndTime
			first = false
		} else {
			if node.StartTime.Before(minStart) {
				minStart = node.StartTime
			}
			if node.EndTime.After(maxEnd) {
				maxEnd = node.EndTime
			}
		}
	}

	g.Metadata.TotalLatencyMs = uint32(maxEnd.Sub(minStart).Milliseconds())
}

// findCriticalPath finds the longest path through the graph
func (g *AgentGraph) findCriticalPath() {
	if len(g.Nodes) == 0 {
		return
	}

	// Build adjacency list
	adjacency := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range g.Edges {
		if edge.Type != EdgeTypeParallel {
			adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
		}
	}

	// Find nodes with no incoming edges (starts)
	hasIncoming := make(map[uuid.UUID]bool)
	for _, edge := range g.Edges {
		if edge.Type != EdgeTypeParallel {
			hasIncoming[edge.Target] = true
		}
	}

	var startNodes []uuid.UUID
	for _, node := range g.Nodes {
		if !hasIncoming[node.ID] {
			startNodes = append(startNodes, node.ID)
		}
	}

	// DFS to find longest path
	dist := make(map[uuid.UUID]uint32)
	parent := make(map[uuid.UUID]uuid.UUID)

	var dfs func(nodeID uuid.UUID, currentDist uint32)
	dfs = func(nodeID uuid.UUID, currentDist uint32) {
		node := g.NodeMap[nodeID]
		newDist := currentDist + node.LatencyMs

		if newDist > dist[nodeID] {
			dist[nodeID] = newDist
		}

		for _, neighborID := range adjacency[nodeID] {
			if newDist > dist[neighborID] {
				dist[neighborID] = newDist
				parent[neighborID] = nodeID
				dfs(neighborID, newDist)
			}
		}
	}

	for _, startNode := range startNodes {
		node := g.NodeMap[startNode]
		dist[startNode] = node.LatencyMs
		dfs(startNode, node.LatencyMs)
	}

	// Find the end node with maximum distance
	var maxDist uint32
	var endNode uuid.UUID
	for nodeID, d := range dist {
		if d > maxDist {
			maxDist = d
			endNode = nodeID
		}
	}

	// Reconstruct path
	var path []uuid.UUID
	current := endNode
	for {
		path = append([]uuid.UUID{current}, path...)
		p, exists := parent[current]
		if !exists {
			break
		}
		current = p
	}

	g.Metadata.CriticalPath = path
	g.Metadata.CriticalPathMs = maxDist
}

// identifyBottlenecks finds nodes that contribute most to latency
func (g *AgentGraph) identifyBottlenecks() {
	if len(g.Nodes) == 0 || g.Metadata.TotalLatencyMs == 0 {
		return
	}

	// Sort nodes by latency
	sortedNodes := make([]*GraphNode, len(g.Nodes))
	copy(sortedNodes, g.Nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].LatencyMs > sortedNodes[j].LatencyMs
	})

	// Take top bottlenecks (nodes contributing > 10% of latency or top 5)
	var bottlenecks []*Bottleneck
	for i, node := range sortedNodes {
		percentage := float64(node.LatencyMs) / float64(g.Metadata.TotalLatencyMs) * 100
		if percentage > 10 || i < 5 {
			reason := "High latency"
			if node.Type == NodeTypeLLM {
				reason = "LLM call latency"
			} else if node.Type == NodeTypeTool {
				reason = "Tool call latency"
			} else if node.Type == NodeTypeAgent {
				reason = "Agent processing time"
			}

			bottlenecks = append(bottlenecks, &Bottleneck{
				NodeID:     node.ID,
				LatencyMs:  node.LatencyMs,
				Percentage: percentage,
				Reason:     reason,
			})
		}
		if i >= 10 {
			break
		}
	}

	g.Metadata.Bottlenecks = bottlenecks
}

// buildExecutionLanes creates lanes for parallel execution visualization
func (g *AgentGraph) buildExecutionLanes(spans []*Span) {
	if len(spans) == 0 {
		return
	}

	// Sort spans by start time
	sortedSpans := make([]*Span, len(spans))
	copy(sortedSpans, spans)
	sort.Slice(sortedSpans, func(i, j int) bool {
		return sortedSpans[i].StartTime.Before(sortedSpans[j].StartTime)
	})

	// Assign spans to lanes (first-fit decreasing)
	var lanes []*ExecutionLane
	spanToLane := make(map[uuid.UUID]int)

	for _, span := range sortedSpans {
		assigned := false
		for i, lane := range lanes {
			// Check if span can fit in this lane (no overlap with last span)
			if span.StartTime.After(lane.EndTime) || span.StartTime.Equal(lane.EndTime) {
				lane.Nodes = append(lane.Nodes, span.ID)
				if span.EndTime.After(lane.EndTime) {
					lane.EndTime = span.EndTime
				}
				lane.LatencyMs = uint32(lane.EndTime.Sub(lane.StartTime).Milliseconds())
				spanToLane[span.ID] = i
				assigned = true
				break
			}
		}

		if !assigned {
			// Create new lane
			lane := &ExecutionLane{
				LaneID:    len(lanes),
				Nodes:     []uuid.UUID{span.ID},
				StartTime: span.StartTime,
				EndTime:   span.EndTime,
				LatencyMs: span.LatencyMs,
			}
			lanes = append(lanes, lane)
			spanToLane[span.ID] = len(lanes) - 1
		}
	}

	g.Metadata.ExecutionLanes = lanes
}

// SimplifyGraph creates a simplified version of the graph for complex flows
func (g *AgentGraph) SimplifyGraph(maxNodes int) *AgentGraph {
	if len(g.Nodes) <= maxNodes {
		return g
	}

	simplified := &AgentGraph{
		TraceID:   g.TraceID,
		ProjectID: g.ProjectID,
		Nodes:     make([]*GraphNode, 0),
		Edges:     make([]*GraphEdge, 0),
		NodeMap:   make(map[uuid.UUID]*GraphNode),
		EdgeMap:   make(map[string]*GraphEdge),
		Metadata:  g.Metadata,
		CreatedAt: g.CreatedAt,
	}

	// Strategy 1: Keep only agent nodes and critical path nodes
	criticalPathSet := make(map[uuid.UUID]bool)
	for _, nodeID := range g.Metadata.CriticalPath {
		criticalPathSet[nodeID] = true
	}

	// Keep agents and critical path nodes
	for _, node := range g.Nodes {
		if node.Type == NodeTypeAgent || criticalPathSet[node.ID] {
			simplified.Nodes = append(simplified.Nodes, node)
			simplified.NodeMap[node.ID] = node
		}
	}

	// If still too many, aggregate by parent
	if len(simplified.Nodes) > maxNodes {
		simplified = g.aggregateNodes(maxNodes)
	}

	// Rebuild edges for remaining nodes
	for _, edge := range g.Edges {
		_, sourceExists := simplified.NodeMap[edge.Source]
		_, targetExists := simplified.NodeMap[edge.Target]
		if sourceExists && targetExists {
			simplified.Edges = append(simplified.Edges, edge)
			edgeKey := edge.Source.String() + "-" + edge.Target.String()
			simplified.EdgeMap[edgeKey] = edge
		}
	}

	return simplified
}

// aggregateNodes aggregates nodes to reduce complexity
func (g *AgentGraph) aggregateNodes(maxNodes int) *AgentGraph {
	// Group nodes by parent and type
	type nodeGroup struct {
		nodes    []*GraphNode
		combined *GraphNode
	}

	groups := make(map[string]*nodeGroup)

	// Find parent for each node
	parentOf := make(map[uuid.UUID]uuid.UUID)
	for _, edge := range g.Edges {
		if edge.Type == EdgeTypeDelegation || edge.Type == EdgeTypeToolCall || edge.Type == EdgeTypeLLMCall {
			parentOf[edge.Target] = edge.Source
		}
	}

	for _, node := range g.Nodes {
		parent := parentOf[node.ID]
		key := parent.String() + "-" + string(node.Type)
		if groups[key] == nil {
			groups[key] = &nodeGroup{nodes: []*GraphNode{}}
		}
		groups[key].nodes = append(groups[key].nodes, node)
	}

	// Create combined nodes for each group
	aggregated := &AgentGraph{
		TraceID:   g.TraceID,
		ProjectID: g.ProjectID,
		Nodes:     make([]*GraphNode, 0),
		Edges:     make([]*GraphEdge, 0),
		NodeMap:   make(map[uuid.UUID]*GraphNode),
		EdgeMap:   make(map[string]*GraphEdge),
		Metadata:  g.Metadata,
		CreatedAt: g.CreatedAt,
	}

	for _, group := range groups {
		if len(group.nodes) == 1 {
			aggregated.Nodes = append(aggregated.Nodes, group.nodes[0])
			aggregated.NodeMap[group.nodes[0].ID] = group.nodes[0]
		} else {
			// Combine nodes
			combined := &GraphNode{
				ID:    uuid.New(),
				Type:  group.nodes[0].Type,
				Label: group.nodes[0].Label + " (+" + string(rune('0'+len(group.nodes)-1)) + " more)",
			}

			// Aggregate metrics
			var totalLatency uint32
			var totalTokens uint32
			var totalCost float64
			minStart := group.nodes[0].StartTime
			maxEnd := group.nodes[0].EndTime

			for _, n := range group.nodes {
				totalLatency += n.LatencyMs
				totalTokens += n.Tokens
				totalCost += n.Cost
				if n.StartTime.Before(minStart) {
					minStart = n.StartTime
				}
				if n.EndTime.After(maxEnd) {
					maxEnd = n.EndTime
				}
			}

			combined.LatencyMs = totalLatency
			combined.Tokens = totalTokens
			combined.Cost = totalCost
			combined.StartTime = minStart
			combined.EndTime = maxEnd

			aggregated.Nodes = append(aggregated.Nodes, combined)
			aggregated.NodeMap[combined.ID] = combined

			// Map old node IDs to new combined ID
			group.combined = combined
		}
	}

	return aggregated
}

// GetSubgraph returns a subgraph starting from a specific node
func (g *AgentGraph) GetSubgraph(rootNodeID uuid.UUID, maxDepth int) *AgentGraph {
	subgraph := &AgentGraph{
		TraceID:   g.TraceID,
		ProjectID: g.ProjectID,
		Nodes:     make([]*GraphNode, 0),
		Edges:     make([]*GraphEdge, 0),
		NodeMap:   make(map[uuid.UUID]*GraphNode),
		EdgeMap:   make(map[string]*GraphEdge),
		CreatedAt: g.CreatedAt,
	}

	// BFS from root node
	adjacency := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range g.Edges {
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	visited := make(map[uuid.UUID]bool)
	type queueItem struct {
		nodeID uuid.UUID
		depth  int
	}

	queue := []queueItem{{rootNodeID, 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.nodeID] || item.depth > maxDepth {
			continue
		}
		visited[item.nodeID] = true

		if node, ok := g.NodeMap[item.nodeID]; ok {
			subgraph.Nodes = append(subgraph.Nodes, node)
			subgraph.NodeMap[node.ID] = node
		}

		for _, neighbor := range adjacency[item.nodeID] {
			if !visited[neighbor] {
				queue = append(queue, queueItem{neighbor, item.depth + 1})
			}
		}
	}

	// Copy relevant edges
	for _, edge := range g.Edges {
		if visited[edge.Source] && visited[edge.Target] {
			subgraph.Edges = append(subgraph.Edges, edge)
			edgeKey := edge.Source.String() + "-" + edge.Target.String()
			subgraph.EdgeMap[edgeKey] = edge
		}
	}

	// Recalculate metadata for subgraph
	subgraph.Metadata = &GraphMetadata{
		TotalNodes: len(subgraph.Nodes),
		TotalEdges: len(subgraph.Edges),
	}

	return subgraph
}

// ToJSON returns a JSON-serializable representation of the graph
func (g *AgentGraph) ToJSON() map[string]interface{} {
	nodes := make([]map[string]interface{}, len(g.Nodes))
	for i, node := range g.Nodes {
		nodes[i] = map[string]interface{}{
			"id":        node.ID,
			"type":      node.Type,
			"label":     node.Label,
			"startTime": node.StartTime,
			"endTime":   node.EndTime,
			"latencyMs": node.LatencyMs,
			"status":    node.Status,
			"depth":     node.Depth,
		}
		if node.Tokens > 0 {
			nodes[i]["tokens"] = node.Tokens
		}
		if node.Cost > 0 {
			nodes[i]["cost"] = node.Cost
		}
		if node.Position != nil {
			nodes[i]["position"] = node.Position
		}
	}

	edges := make([]map[string]interface{}, len(g.Edges))
	for i, edge := range g.Edges {
		edges[i] = map[string]interface{}{
			"id":     edge.ID,
			"source": edge.Source,
			"target": edge.Target,
			"type":   edge.Type,
			"order":  edge.Order,
		}
		if edge.Label != "" {
			edges[i]["label"] = edge.Label
		}
	}

	return map[string]interface{}{
		"traceId":   g.TraceID,
		"projectId": g.ProjectID,
		"nodes":     nodes,
		"edges":     edges,
		"metadata":  g.Metadata,
		"createdAt": g.CreatedAt,
	}
}
