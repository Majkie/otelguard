import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';
import type {
  Agent,
  AgentGraph,
  AgentHierarchy,
  ToolCall,
  AgentState,
  ListAgentsParams,
  ListAgentsResponse,
  ListToolCallsResponse,
  ListAgentMessagesResponse,
  AgentStatistics,
} from '@/types/agent';

// Query keys
export const agentKeys = {
  all: ['agents'] as const,
  lists: () => [...agentKeys.all, 'list'] as const,
  list: (params: ListAgentsParams) => [...agentKeys.lists(), params] as const,
  details: () => [...agentKeys.all, 'detail'] as const,
  detail: (id: string) => [...agentKeys.details(), id] as const,
  traceAgents: (traceId: string) => [...agentKeys.all, 'trace', traceId] as const,
  hierarchy: (traceId: string) => [...agentKeys.all, 'hierarchy', traceId] as const,
  graph: (traceId: string) => [...agentKeys.all, 'graph', traceId] as const,
  subgraph: (traceId: string, nodeId: string, depth: number) =>
    [...agentKeys.graph(traceId), 'subgraph', nodeId, depth] as const,
  toolCalls: (traceId: string) => [...agentKeys.all, 'toolCalls', traceId] as const,
  agentToolCalls: (agentId: string) => [...agentKeys.all, 'agentToolCalls', agentId] as const,
  states: (agentId: string) => [...agentKeys.all, 'states', agentId] as const,
  messages: (traceId: string) => [...agentKeys.all, 'messages', traceId] as const,
  statistics: () => [...agentKeys.all, 'statistics'] as const,
  toolCallStatistics: () => [...agentKeys.all, 'toolCallStatistics'] as const,
};

// List agents with optional filtering
export function useAgents(params: Omit<ListAgentsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListAgentsResponse>('/v1/agents', {
        params: { ...params, projectId },
      }),
    enabled: !!projectId,
  });
}

// Get a single agent
export function useAgent(id: string) {
  return useQuery({
    queryKey: agentKeys.detail(id),
    queryFn: () => api.get<Agent>(`/v1/agents/${id}`),
    enabled: !!id,
  });
}

// Get all agents for a trace
export function useTraceAgents(traceId: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.traceAgents(traceId),
    queryFn: () =>
      api.get<{ data: Agent[] }>(`/v1/traces/${traceId}/agents`, {
        params: { projectId },
      }),
    enabled: !!traceId && !!projectId,
  });
}

// Get agent hierarchy for a trace
export function useAgentHierarchy(traceId: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.hierarchy(traceId),
    queryFn: () =>
      api.get<AgentHierarchy>(`/v1/traces/${traceId}/agents/hierarchy`, {
        params: { projectId },
      }),
    enabled: !!traceId && !!projectId,
  });
}

// Get agent graph visualization data
export function useAgentGraph(traceId: string, options?: { simplified?: boolean; maxNodes?: number }) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.graph(traceId),
    queryFn: () =>
      api.get<AgentGraph>(`/v1/traces/${traceId}/graph`, {
        params: {
          projectId,
          simplified: options?.simplified,
          maxNodes: options?.maxNodes,
        },
      }),
    enabled: !!traceId && !!projectId,
  });
}

// Get subgraph from a specific node
export function useAgentSubgraph(traceId: string, nodeId: string, depth: number = 3) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.subgraph(traceId, nodeId, depth),
    queryFn: () =>
      api.get<AgentGraph>(`/v1/traces/${traceId}/graph/${nodeId}/subgraph`, {
        params: { projectId, depth },
      }),
    enabled: !!traceId && !!nodeId && !!projectId,
  });
}

// Get tool calls for a trace
export function useTraceToolCalls(traceId: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.toolCalls(traceId),
    queryFn: () =>
      api.get<ListToolCallsResponse>(`/v1/traces/${traceId}/tool-calls`, {
        params: { projectId },
      }),
    enabled: !!traceId && !!projectId,
  });
}

// Get tool calls for an agent
export function useAgentToolCalls(agentId: string) {
  return useQuery({
    queryKey: agentKeys.agentToolCalls(agentId),
    queryFn: () => api.get<{ data: ToolCall[] }>(`/v1/agents/${agentId}/tool-calls`),
    enabled: !!agentId,
  });
}

// Get agent state snapshots
export function useAgentStates(agentId: string) {
  return useQuery({
    queryKey: agentKeys.states(agentId),
    queryFn: () => api.get<{ data: AgentState[] }>(`/v1/agents/${agentId}/states`),
    enabled: !!agentId,
  });
}

// Get messages between agents in a trace
export function useAgentMessages(traceId: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.messages(traceId),
    queryFn: () =>
      api.get<ListAgentMessagesResponse>(`/v1/traces/${traceId}/agent-messages`, {
        params: { projectId },
      }),
    enabled: !!traceId && !!projectId,
  });
}

// Get agent statistics
export function useAgentStatistics() {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: agentKeys.statistics(),
    queryFn: () =>
      api.get<AgentStatistics>('/v1/analytics/agents', {
        params: { projectId },
      }),
    enabled: !!projectId,
  });
}
