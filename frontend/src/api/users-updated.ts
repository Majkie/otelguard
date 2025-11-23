import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';
import type { Trace } from './traces';
import type { Session } from './sessions';

// Types
export interface User {
  userId: string;
  projectId: string;
  traceCount: number;
  sessionCount: number;
  totalLatencyMs: number;
  avgLatencyMs: number;
  totalTokens: number;
  totalCost: number;
  successCount: number;
  errorCount: number;
  successRate: number;
  firstSeenTime: string;
  lastSeenTime: string;
  models?: string[];
}

export interface ListUsersParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  startTime?: string;
  endTime?: string;
}

export interface ListUsersResponse {
  data: User[];
  total: number;
  limit: number;
  offset: number;
}

// Query keys factory
export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (params: ListUsersParams) => [...userKeys.lists(), params] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

// Hooks with project context
export function useUsers(params: Omit<ListUsersParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: userKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListUsersResponse>('/v1/users', { 
        params: { ...params, projectId } 
      }),
    enabled: !!projectId,
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => api.get<User>(`/v1/users/${id}`),
    enabled: !!id,
  });
}
