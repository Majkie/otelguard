import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import type { Trace } from './traces';

export interface Session {
  sessionId: string;
  projectId: string;
  userId?: string;
  traceCount: number;
  totalLatencyMs: number;
  totalTokens: number;
  totalCost: number;
  successCount: number;
  errorCount: number;
  firstTraceTime: string;
  lastTraceTime: string;
  models?: string[];
}

export interface ListSessionsParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  userId?: string;
  startTime?: string;
  endTime?: string;
}

export interface ListSessionsResponse {
  data: Session[];
  total: number;
  limit: number;
  offset: number;
}

export interface SessionDetailResponse {
  session: Session;
  traces: {
    data: Trace[];
    total: number;
    limit: number;
    offset: number;
  };
}

// Query keys
export const sessionKeys = {
  all: ['sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (params: ListSessionsParams) => [...sessionKeys.lists(), params] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (id: string) => [...sessionKeys.details(), id] as const,
};

// Hooks
export function useSessions(params: ListSessionsParams = {}) {
  return useQuery({
    queryKey: sessionKeys.list(params),
    queryFn: () => api.get<ListSessionsResponse>('/v1/sessions', { params }),
  });
}

export function useSession(id: string, traceLimit?: number, traceOffset?: number) {
  return useQuery({
    queryKey: sessionKeys.detail(id),
    queryFn: () =>
      api.get<SessionDetailResponse>(`/v1/sessions/${id}`, {
        params: { traceLimit, traceOffset },
      }),
    enabled: !!id,
  });
}
