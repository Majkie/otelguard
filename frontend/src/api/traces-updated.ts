import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface Trace {
  id: string;
  projectId: string;
  sessionId?: string;
  userId?: string;
  name: string;
  input: string;
  output: string;
  metadata?: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  totalTokens: number;
  promptTokens: number;
  completionTokens: number;
  cost: number;
  model: string;
  tags: string[];
  status: string;
  errorMessage?: string;
}

export interface Span {
  id: string;
  traceId: string;
  parentSpanId?: string;
  projectId: string;
  name: string;
  type: string;
  input: string;
  output: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  tokens: number;
  cost: number;
  model?: string;
  status: string;
}

// Update ListTracesParams to make projectId required
export interface ListTracesParams {
  // Pagination
  limit?: number;
  offset?: number;
  // Required project filter
  projectId: string;
  // Basic filters
  sessionId?: string;
  userId?: string;
  model?: string;
  name?: string;
  status?: 'success' | 'error' | 'pending';
  tags?: string;
  // Time filters
  startTime?: string;
  endTime?: string;
  // Numeric filters
  minLatency?: number;
  maxLatency?: number;
  minCost?: number;
  maxCost?: number;
  // Sorting
  sortBy?: 'start_time' | 'latency_ms' | 'cost' | 'total_tokens' | 'name' | 'model';
  sortOrder?: 'ASC' | 'DESC';
}

export interface ListTracesResponse {
  data: Trace[];
  total: number;
  limit: number;
  offset: number;
}

// Query keys
export const traceKeys = {
  all: ['traces'] as const,
  lists: () => [...traceKeys.all, 'list'] as const,
  list: (params: ListTracesParams) => [...traceKeys.lists(), params] as const,
  details: () => [...traceKeys.all, 'detail'] as const,
  detail: (id: string) => [...traceKeys.details(), id] as const,
  spans: (traceId: string) => [...traceKeys.detail(traceId), 'spans'] as const,
};

// Hooks
// Update useTraces hook to require projectId
export function useTraces(params: ListTracesParams) {
  return useQuery({
    queryKey: traceKeys.list(params),
    queryFn: () => api.get<ListTracesResponse>('/v1/traces', { params }),
    enabled: !!params.projectId,
  });
}

export function useTrace(id: string) {
  return useQuery({
    queryKey: traceKeys.detail(id),
    queryFn: () => api.get<Trace>(`/v1/traces/${id}`),
    enabled: !!id,
  });
}

// Update useTraceSpans to include projectId
export function useTraceSpans(traceId: string, projectId: string) {
  return useQuery({
    queryKey: traceKeys.spans(traceId),
    queryFn: () => 
      api.get<{ data: Span[] }>(`/v1/traces/${traceId}/spans`, {
        params: { projectId }
      }),
    enabled: !!traceId && !!projectId,
  });
}

export function useDeleteTrace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/traces/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: traceKeys.lists() });
    },
  });
}
