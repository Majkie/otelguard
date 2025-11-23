import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

// Types
export interface Score {
  id: string;
  projectId: string;
  traceId: string;
  spanId?: string;
  name: string;
  value: number;
  stringValue?: string;
  dataType: 'numeric' | 'boolean' | 'categorical';
  source: 'api' | 'llm_judge' | 'human' | 'user_feedback';
  configId?: string;
  comment?: string;
  createdAt: string;
}

export interface ScoreAggregation {
  name: string;
  dataType: string;
  count: number;
  avgValue?: number;
  minValue?: number;
  maxValue?: number;
  sumValue?: number;
  categories?: Record<string, number>; // For categorical scores
}

export interface ScoreTrend {
  timePeriod: string;
  name: string;
  dataType: string;
  count: number;
  avgValue?: number;
  categories?: Record<string, number>;
}

export interface ScoreComparison {
  dimension: string;
  value: string;
  name: string;
  dataType: string;
  count: number;
  avgValue?: number;
  categories?: Record<string, number>;
}

export interface ListScoresParams {
  // Pagination
  limit?: number;
  offset?: number;
  // Project filter (injected by context)
  projectId?: string;
  // Filters
  traceId?: string;
  spanId?: string;
  name?: string;
  source?: 'api' | 'llm_judge' | 'human' | 'user_feedback';
  dataType?: 'numeric' | 'boolean' | 'categorical';
  startTime?: string;
  endTime?: string;
}

export interface ScoreAnalyticsParams {
  projectId?: string;
  traceId?: string;
  spanId?: string;
  name?: string;
  source?: 'api' | 'llm_judge' | 'human' | 'user_feedback';
  startTime?: string;
  endTime?: string;
}

export interface ScoreTrendsParams extends ScoreAnalyticsParams {
  groupBy?: 'hour' | 'day' | 'week' | 'month';
}

export interface ScoreComparisonsParams extends ScoreAnalyticsParams {
  dimension: 'model' | 'user' | 'session' | 'prompt';
}

// Response types
export interface ListScoresResponse {
  scores: Score[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface ScoreAggregationsResponse {
  aggregations: ScoreAggregation[];
}

export interface ScoreTrendsResponse {
  trends: ScoreTrend[];
  groupBy: string;
}

export interface ScoreComparisonsResponse {
  comparisons: ScoreComparison[];
  dimension: string;
}

// Query keys
export const scoreKeys = {
  all: ['scores'] as const,
  lists: () => [...scoreKeys.all, 'list'] as const,
  list: (params: ListScoresParams) => [...scoreKeys.lists(), params] as const,
  details: () => [...scoreKeys.all, 'detail'] as const,
  detail: (id: string) => [...scoreKeys.details(), id] as const,
  aggregations: (params: ScoreAnalyticsParams) => [...scoreKeys.all, 'aggregations', params] as const,
  trends: (params: ScoreTrendsParams) => [...scoreKeys.all, 'trends', params] as const,
  comparisons: (params: ScoreComparisonsParams) => [...scoreKeys.all, 'comparisons', params] as const,
};

// Hooks
export function useScores(params: Omit<ListScoresParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: scoreKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListScoresResponse>('/v1/scores', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useScore(id: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: scoreKeys.detail(id),
    queryFn: () =>
      api.get<Score>(`/v1/scores/${id}`, {
        params: { projectId }
      }),
    enabled: !!id && !!projectId,
  });
}

export function useScoreAggregations(params: Omit<ScoreAnalyticsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: scoreKeys.aggregations({ ...params, projectId }),
    queryFn: () =>
      api.get<ScoreAggregationsResponse>('/v1/analytics/scores/aggregations', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useScoreTrends(params: Omit<ScoreTrendsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: scoreKeys.trends({ ...params, projectId }),
    queryFn: () =>
      api.get<ScoreTrendsResponse>('/v1/analytics/scores/trends', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useScoreComparisons(params: Omit<ScoreComparisonsParams, 'projectId'>) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: scoreKeys.comparisons({ ...params, projectId }),
    queryFn: () =>
      api.get<ScoreComparisonsResponse>('/v1/analytics/scores/comparisons', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId && !!params.dimension,
  });
}
