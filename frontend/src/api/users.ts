import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import type { Trace } from './traces';
import type { Session } from './sessions';

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

export interface UserDetailResponse {
  user: User;
  traces: {
    data: Trace[];
    total: number;
    limit: number;
    offset: number;
  };
  sessions: {
    data: Session[];
    total: number;
    limit: number;
    offset: number;
  };
}

export interface SearchTracesParams {
  q: string;
  limit?: number;
  offset?: number;
  projectId?: string;
  startTime?: string;
  endTime?: string;
}

export interface SearchTracesResponse {
  data: Trace[];
  total: number;
  limit: number;
  offset: number;
  query: string;
}

// Query keys
export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (params: ListUsersParams) => [...userKeys.lists(), params] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

export const searchKeys = {
  traces: (params: SearchTracesParams) => ['search', 'traces', params] as const,
};

// Hooks
export function useUsers(params: ListUsersParams = {}) {
  return useQuery({
    queryKey: userKeys.list(params),
    queryFn: () => api.get<ListUsersResponse>('/v1/users', { params }),
  });
}

export function useUser(
  id: string,
  traceLimit?: number,
  traceOffset?: number,
  sessionLimit?: number,
  sessionOffset?: number
) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () =>
      api.get<UserDetailResponse>(`/v1/users/${encodeURIComponent(id)}`, {
        params: { traceLimit, traceOffset, sessionLimit, sessionOffset },
      }),
    enabled: !!id,
  });
}

export function useSearchTraces(params: SearchTracesParams) {
  return useQuery({
    queryKey: searchKeys.traces(params),
    queryFn: () => api.get<SearchTracesResponse>('/v1/search/traces', { params }),
    enabled: !!params.q && params.q.length > 0,
  });
}
