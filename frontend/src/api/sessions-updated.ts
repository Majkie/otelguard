import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';
import type { Trace } from './traces';

// Types
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

// Query keys factory
export const sessionKeys = {
  all: ['sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (params: ListSessionsParams) => [...sessionKeys.lists(), params] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (id: string) => [...sessionKeys.details(), id] as const,
};

// Hooks with project context
export function useSessions(params: Omit<ListSessionsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: sessionKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListSessionsResponse>('/v1/sessions', { 
        params: { ...params, projectId } 
      }),
    enabled: !!projectId,
  });
}

export function useSession(id: string) {
  return useQuery({
    queryKey: sessionKeys.detail(id),
    queryFn: () => api.get<Session>(`/v1/sessions/${id}`),
    enabled: !!id,
  });
}
