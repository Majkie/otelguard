import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

// Types
export interface OverviewMetrics {
  totalTraces: number;
  totalTokens: number;
  totalCost: number;
  avgLatencyMs: number;
  errorRate: number;
  successCount: number;
  errorCount: number;
  uniqueUsers: number;
  uniqueSessions: number;
}

export interface TimeSeriesPoint {
  timestamp: string;
  value: number;
  count: number;
}

export interface CostByModel {
  model: string;
  totalCost: number;
  totalTokens: number;
  traceCount: number;
}

export interface AnalyticsOptions {
  projectId?: string;
  startTime?: string;
  endTime?: string;
  granularity?: 'hour' | 'day' | 'week';
}

// Response types
export interface OverviewResponse extends OverviewMetrics {}

export interface CostAnalyticsResponse {
  data: TimeSeriesPoint[];
  byModel: CostByModel[];
  totalCost: number;
}

export interface UsageAnalyticsResponse {
  data: TimeSeriesPoint[];
  totalTokens: number;
}

// Query keys
export const analyticsKeys = {
  all: ['analytics'] as const,
  overview: (params: AnalyticsOptions) => [...analyticsKeys.all, 'overview', params] as const,
  costs: (params: AnalyticsOptions) => [...analyticsKeys.all, 'costs', params] as const,
  usage: (params: AnalyticsOptions) => [...analyticsKeys.all, 'usage', params] as const,
};

// Hooks
export function useOverviewMetrics(params: Omit<AnalyticsOptions, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: analyticsKeys.overview({ ...params, projectId }),
    queryFn: () =>
      api.get<OverviewResponse>('/v1/analytics/overview', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useCostAnalytics(params: Omit<AnalyticsOptions, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: analyticsKeys.costs({ ...params, projectId }),
    queryFn: () =>
      api.get<CostAnalyticsResponse>('/v1/analytics/costs', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useUsageAnalytics(params: Omit<AnalyticsOptions, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: analyticsKeys.usage({ ...params, projectId }),
    queryFn: () =>
      api.get<UsageAnalyticsResponse>('/v1/analytics/usage', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}
