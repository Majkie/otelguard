// Agent types based on backend domain models

export type AgentType =
  | 'orchestrator'
  | 'worker'
  | 'tool_caller'
  | 'planner'
  | 'executor'
  | 'reviewer'
  | 'custom';

export type AgentStatus = 'running' | 'success' | 'error' | 'timeout';

export interface Agent {
  id: string;
  projectId: string;
  traceId: string;
  spanId: string;
  parentAgentId?: string;
  name: string;
  agentType: AgentType;
  role: string;
  model?: string;
  systemPrompt?: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  totalTokens: number;
  cost: number;
  status: AgentStatus;
  errorMessage?: string;
  metadata?: string;
  tags?: string[];
  createdAt: string;
}

export type RelationType =
  | 'delegates_to'
  | 'calls'
  | 'responds_to'
  | 'supervises'
  | 'collaborates';

export interface AgentRelationship {
  id: string;
  projectId: string;
  traceId: string;
  sourceAgentId: string;
  targetAgentId: string;
  relationType: RelationType;
  timestamp: string;
  metadata?: string;
  createdAt: string;
}

export interface ToolCall {
  id: string;
  projectId: string;
  traceId: string;
  spanId: string;
  agentId?: string;
  name: string;
  description?: string;
  input: string;
  output: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  status: 'success' | 'error' | 'timeout' | 'pending';
  errorMessage?: string;
  retryCount: number;
  metadata?: string;
  createdAt: string;
}

export type MessageType = 'request' | 'response' | 'notification' | 'broadcast';
export type MessageRole = 'user' | 'assistant' | 'system' | 'function' | 'tool';
export type ContentType = 'text' | 'json' | 'tool_call' | 'tool_result';

export interface AgentMessage {
  id: string;
  projectId: string;
  traceId: string;
  spanId?: string;
  fromAgentId: string;
  toAgentId: string;
  messageType: MessageType;
  role: MessageRole;
  content: string;
  contentType: ContentType;
  sequenceNum: number;
  parentMsgId?: string;
  tokenCount: number;
  timestamp: string;
  metadata?: string;
  createdAt: string;
}

export interface AgentState {
  id: string;
  projectId: string;
  traceId: string;
  agentId: string;
  spanId?: string;
  sequenceNum: number;
  state: string;
  variables: string;
  memory: string;
  plan?: string;
  reasoning?: string;
  timestamp: string;
  metadata?: string;
  createdAt: string;
}

// Graph types
export type NodeType =
  | 'agent'
  | 'tool'
  | 'llm'
  | 'retrieval'
  | 'embedding'
  | 'message'
  | 'start'
  | 'end'
  | 'custom';

export type EdgeType =
  | 'delegation'
  | 'tool_call'
  | 'llm_call'
  | 'message'
  | 'sequence'
  | 'parallel'
  | 'return'
  | 'custom';

export interface NodePosition {
  x: number;
  y: number;
}

export interface GraphNode {
  id: string;
  type: NodeType;
  label: string;
  agentId?: string;
  spanId?: string;
  toolCallId?: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  status: string;
  tokens?: number;
  cost?: number;
  model?: string;
  depth: number;
  parallelGroup: number;
  metadata?: string;
  position?: NodePosition;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: EdgeType;
  label?: string;
  weight?: number;
  order: number;
  messageId?: string;
  latencyMs?: number;
  metadata?: string;
}

export interface Bottleneck {
  nodeId: string;
  latencyMs: number;
  percentage: number;
  reason: string;
}

export interface ExecutionLane {
  laneId: number;
  nodes: string[];
  startTime: string;
  endTime: string;
  latencyMs: number;
}

export interface GraphMetadata {
  totalNodes: number;
  totalEdges: number;
  maxDepth: number;
  maxParallelism: number;
  hasCycles: boolean;
  cycleNodes?: string[];
  parallelGroups: number;
  totalLatencyMs: number;
  criticalPath?: string[];
  criticalPathMs: number;
  bottlenecks?: Bottleneck[];
  executionLanes?: ExecutionLane[];
}

export interface AgentGraph {
  traceId: string;
  projectId: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: GraphMetadata;
  createdAt: string;
}

// Hierarchy types
export interface AgentNode {
  agent: Agent;
  children?: AgentNode[];
  depth: number;
  level: number;
}

export interface AgentHierarchy {
  traceId: string;
  agents: AgentNode[];
  rootAgents: AgentNode[];
  maxDepth: number;
}

// API response types
export interface ListAgentsParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  traceId?: string;
  agentType?: AgentType;
  status?: AgentStatus;
  sortBy?: 'start_time' | 'latency_ms' | 'cost' | 'total_tokens' | 'name';
  sortOrder?: 'ASC' | 'DESC';
}

export interface ListAgentsResponse {
  data: Agent[];
  total: number;
  limit: number;
  offset: number;
}

export interface ListToolCallsResponse {
  data: ToolCall[];
  total: number;
}

export interface ListAgentMessagesResponse {
  data: AgentMessage[];
  total: number;
}

export interface AgentStatistics {
  totalAgents: number;
  byType: Record<AgentType, number>;
  byStatus: Record<AgentStatus, number>;
  avgLatencyMs: number;
  avgTokens: number;
  avgCost: number;
  totalCost: number;
}
