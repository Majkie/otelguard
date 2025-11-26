import { useQuery } from '@tanstack/react-query';
import { api } from './client';

// ============================================================================
// Types
// ============================================================================

export interface CoreMetrics {
  totalTraces: number;
  totalSpans: number;
  avgLatencyMs: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  totalCost: number;
  avgCost: number;
  totalTokens: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  errorCount: number;
  errorRate: number;
}

export interface TimeSeriesPoint {
  timestamp: string;
  value: number;
  count?: number;
}

export interface TimeSeriesData {
  metricName: string;
  points: TimeSeriesPoint[];
}

export interface ModelMetrics {
  model: string;
  traceCount: number;
  avgLatencyMs: number;
  totalCost: number;
  totalTokens: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  errorCount: number;
  errorRate: number;
}

export interface UserMetrics {
  userId: string;
  traceCount: number;
  totalCost: number;
  totalTokens: number;
  avgLatency: number;
  errorCount: number;
  lastActivity: string;
}

export interface ModelCostSummary {
  model: string;
  totalCost: number;
  traceCount: number;
  avgCost: number;
  totalTokens: number;
}

export interface UserCostSummary {
  userId: string;
  totalCost: number;
  traceCount: number;
  avgCost: number;
  totalTokens: number;
}

export interface CostBreakdown {
  totalCost: number;
  costByModel: Record<string, number>;
  costByUser: Record<string, number>;
  costOverTime: TimeSeriesPoint[];
  topCostModels: ModelCostSummary[];
  topCostUsers: UserCostSummary[];
}

export interface QualityMetrics {
  totalScores: number;
  avgScore: number;
  scoresByName: Record<string, number>;
  feedbackCount: number;
  positiveFeedback: number;
  negativeFeedback: number;
  feedbackRate: number;
}

export interface MetricsFilter {
  projectId: string;
  startTime?: string;
  endTime?: string;
  model?: string;
  userId?: string;
  sessionId?: string;
}

export interface TimeSeriesFilter extends MetricsFilter {
  metric: string; // traces, latency, cost, tokens, errors, error_rate
  interval?: 'hour' | 'day' | 'week' | 'month';
}

// ============================================================================
// Query Keys
// ============================================================================

export const metricsKeys = {
  all: ['metrics'] as const,
  core: (filter: MetricsFilter) => [...metricsKeys.all, 'core', filter] as const,
  timeseries: (filter: TimeSeriesFilter) => [...metricsKeys.all, 'timeseries', filter] as const,
  models: (filter: MetricsFilter) => [...metricsKeys.all, 'models', filter] as const,
  users: (filter: MetricsFilter, limit?: number) => [...metricsKeys.all, 'users', filter, limit] as const,
  cost: (filter: MetricsFilter) => [...metricsKeys.all, 'cost', filter] as const,
  quality: (filter: MetricsFilter) => [...metricsKeys.all, 'quality', filter] as const,
};

// ============================================================================
// Hooks
// ============================================================================

export function useCoreMetrics(filter: MetricsFilter) {
  return useQuery({
    queryKey: metricsKeys.core(filter),
    queryFn: () => api.get<CoreMetrics>('/v1/metrics/core', { params: filter as any }),
    enabled: !!filter.projectId,
  });
}

export function useTimeSeriesMetrics(filter: TimeSeriesFilter) {
  return useQuery({
    queryKey: metricsKeys.timeseries(filter),
    queryFn: () => api.get<TimeSeriesData>('/v1/metrics/timeseries', { params: filter as any }),
    enabled: !!filter.projectId && !!filter.metric,
  });
}

export function useModelBreakdown(filter: MetricsFilter) {
  return useQuery({
    queryKey: metricsKeys.models(filter),
    queryFn: () => api.get<{ data: ModelMetrics[] }>('/v1/metrics/models', { params: filter as any }).then(res => res.data),
    enabled: !!filter.projectId,
  });
}

export function useUserBreakdown(filter: MetricsFilter, limit?: number) {
  return useQuery({
    queryKey: metricsKeys.users(filter, limit),
    queryFn: () => api.get<{ data: UserMetrics[] }>('/v1/metrics/users', { params: { ...filter, limit } as any }).then(res => res.data),
    enabled: !!filter.projectId,
  });
}

export function useCostBreakdown(filter: MetricsFilter) {
  return useQuery({
    queryKey: metricsKeys.cost(filter),
    queryFn: () => api.get<CostBreakdown>('/v1/metrics/cost', { params: filter as any }),
    enabled: !!filter.projectId,
  });
}

export function useQualityMetrics(filter: MetricsFilter) {
  return useQuery({
    queryKey: metricsKeys.quality(filter),
    queryFn: () => api.get<QualityMetrics>('/v1/metrics/quality', { params: filter as any }),
    enabled: !!filter.projectId,
  });
}
