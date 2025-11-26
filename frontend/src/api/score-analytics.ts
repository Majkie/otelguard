import { useQuery } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface ScoreDistribution {
  scoreName: string;
  mean: number;
  median: number;
  stdDev: number;
  min: number;
  max: number;
  count: number;
  histogram: HistogramBucket[];
  percentiles: Record<string, number>;
}

export interface HistogramBucket {
  min: number;
  max: number;
  count: number;
}

export interface CorrelationResult {
  score1: string;
  score2: string;
  pearson: number;
  spearman: number;
  sampleSize: number;
  pValue: number;
  isSignificant: boolean;
}

export interface ScoreBreakdown {
  dimension: string;
  values: Record<string, BreakdownStats>;
}

export interface BreakdownStats {
  count: number;
  mean: number;
  stdDev: number;
}

export interface CohenKappaResult {
  kappa: number;
  interpretation: string;
  agreement: number;
  chanceAgreement: number;
  confusionMatrix: Record<string, Record<string, number>>;
}

export interface F1ScoreResult {
  f1Score: number;
  precision: number;
  recall: number;
  accuracy: number;
  truePositives: number;
  trueNegatives: number;
  falsePositives: number;
  falseNegatives: number;
}

export interface ScoreTrend {
  scoreName: string;
  datapoints: TrendDatapoint[];
  trend: 'increasing' | 'decreasing' | 'stable';
  changeRate: number;
}

export interface TrendDatapoint {
  timestamp: string;
  mean: number;
  count: number;
}

export interface ScoreAnalyticsParams {
  scoreName?: string;
  startTime?: string;
  endTime?: string;
  dimension?: string;
  score1?: string;
  score2?: string;
  interval?: '1h' | '6h' | '12h' | '1d' | '1w';
  annotator1?: string;
  annotator2?: string;
  threshold?: number;
  groundTruthSource?: string;
}

// Query keys
export const scoreAnalyticsKeys = {
  all: ['score-analytics'] as const,
  distribution: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'distribution', params] as const,
  correlation: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'correlation', params] as const,
  breakdown: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'breakdown', params] as const,
  kappa: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'kappa', params] as const,
  f1: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'f1', params] as const,
  trend: (params: ScoreAnalyticsParams) =>
    [...scoreAnalyticsKeys.all, 'trend', params] as const,
};

// Hooks
export function useScoreDistribution(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.distribution(params),
    queryFn: () =>
      api.get<ScoreDistribution>('/v1/analytics/scores/distribution', {
        params: params as Record<string, string>,
      }),
    enabled: !!params.scoreName,
  });
}

export function useScoreCorrelation(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.correlation(params),
    queryFn: () =>
      api.get<CorrelationResult>('/v1/analytics/scores/correlation', {
        params: params as Record<string, string>,
      }),
    enabled: !!params.score1 && !!params.score2,
  });
}

export function useScoreBreakdown(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.breakdown(params),
    queryFn: () =>
      api.get<ScoreBreakdown>('/v1/analytics/scores/breakdown', {
        params: params as Record<string, string>,
      }),
    enabled: !!params.scoreName && !!params.dimension,
  });
}

export function useCohenKappa(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.kappa(params),
    queryFn: () =>
      api.get<CohenKappaResult>('/v1/analytics/scores/kappa', {
        params: params as Record<string, string>,
      }),
    enabled: !!params.scoreName && !!params.annotator1 && !!params.annotator2,
  });
}

export function useF1Score(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.f1(params),
    queryFn: () =>
      api.post<F1ScoreResult>('/v1/analytics/scores/f1', {
        threshold: params.threshold || 0.5,
      }, {
        params: {
          score_name: params.scoreName,
          ground_truth_source: params.groundTruthSource,
          start_time: params.startTime,
          end_time: params.endTime,
        } as Record<string, string>,
      }),
    enabled: !!params.scoreName,
  });
}

export function useScoreTrend(params: ScoreAnalyticsParams) {
  return useQuery({
    queryKey: scoreAnalyticsKeys.trend(params),
    queryFn: () =>
      api.get<ScoreTrend>('/v1/analytics/scores/trend', {
        params: params as Record<string, string>,
      }),
    enabled: !!params.scoreName,
  });
}
