import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// ============================================================================
// Types
// ============================================================================

export interface Experiment {
  id: string;
  projectId: string;
  datasetId: string;
  name: string;
  description?: string;
  config: ExperimentConfig;
  status: 'pending' | 'running' | 'completed' | 'failed';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExperimentConfig {
  promptId?: string;
  promptVersion?: number;
  model: string;
  provider: string;
  parameters?: Record<string, any>;
  evaluators?: string[];
}

export interface ExperimentRun {
  id: string;
  experimentId: string;
  runNumber: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  startedAt: string;
  completedAt?: string;
  totalItems: number;
  completedItems: number;
  failedItems: number;
  totalCost: number;
  avgLatencyMs: number;
  error?: string;
  createdAt: string;
}

export interface ExperimentResult {
  id: string;
  runId: string;
  datasetItemId: string;
  status: 'success' | 'error';
  output?: Record<string, any>;
  error?: string;
  latencyMs: number;
  tokensUsed: number;
  cost: number;
  scores?: Record<string, any>;
  createdAt: string;
}

export interface ComparisonMetrics {
  mean: number;
  median: number;
  stdDev: number;
  min: number;
  max: number;
  n: number;
}

export interface ExperimentComparison {
  runIds: string[];
  runs: ExperimentRun[];
  metrics: Record<string, ComparisonMetrics>;
}

export interface PairwiseComparison {
  run1Id: string;
  run2Id: string;
  run1Name: string;
  run2Name: string;
  metricName: string;
  tStatistic: number;
  pValue: number;
  degreesOfFreedom: number;
  significantAt05: boolean;
  significantAt01: boolean;
  meanDifference: number;
  effectSize: number;
}

export interface StatisticalComparison extends ExperimentComparison {
  pairwiseTests: Record<string, PairwiseComparison[]>;
}

export interface CreateExperimentInput {
  projectId: string;
  datasetId: string;
  name: string;
  description?: string;
  config: ExperimentConfig;
  createdBy: string;
}

export interface ExecuteExperimentInput {
  experimentId: string;
  async?: boolean;
}

export interface ListExperimentsParams {
  projectId: string;
  limit?: number;
  offset?: number;
}

// ============================================================================
// Query Keys
// ============================================================================

export const experimentKeys = {
  all: ['experiments'] as const,
  lists: () => [...experimentKeys.all, 'list'] as const,
  list: (params: ListExperimentsParams) => [...experimentKeys.lists(), params] as const,
  details: () => [...experimentKeys.all, 'detail'] as const,
  detail: (id: string) => [...experimentKeys.details(), id] as const,
  runs: (experimentId: string) => [...experimentKeys.all, 'runs', experimentId] as const,
  run: (runId: string) => [...experimentKeys.all, 'run', runId] as const,
  results: (runId: string) => [...experimentKeys.all, 'results', runId] as const,
  comparison: (runIds: string[]) => [...experimentKeys.all, 'comparison', runIds.sort().join(',')] as const,
  statisticalComparison: (runIds: string[]) => [...experimentKeys.all, 'statistical-comparison', runIds.sort().join(',')] as const,
};

// ============================================================================
// Hooks
// ============================================================================

/**
 * List experiments for a project
 */
export function useExperiments(params: ListExperimentsParams) {
  return useQuery({
    queryKey: experimentKeys.list(params),
    queryFn: () =>
      api.get<{
        data: Experiment[];
        total: number;
        limit: number;
        offset: number;
      }>('/v1/experiments', { params }),
  });
}

/**
 * Get a single experiment by ID
 */
export function useExperiment(id: string) {
  return useQuery({
    queryKey: experimentKeys.detail(id),
    queryFn: () => api.get<Experiment>(`/v1/experiments/${id}`),
    enabled: !!id,
  });
}

/**
 * List all runs for an experiment
 */
export function useExperimentRuns(experimentId: string) {
  return useQuery({
    queryKey: experimentKeys.runs(experimentId),
    queryFn: () =>
      api.get<{ data: ExperimentRun[] }>(`/v1/experiments/${experimentId}/runs`),
    enabled: !!experimentId,
  });
}

/**
 * Get a specific experiment run
 */
export function useExperimentRun(runId: string) {
  return useQuery({
    queryKey: experimentKeys.run(runId),
    queryFn: () => api.get<ExperimentRun>(`/v1/experiments/runs/${runId}`),
    enabled: !!runId,
  });
}

/**
 * Get results for a specific run
 */
export function useExperimentResults(runId: string) {
  return useQuery({
    queryKey: experimentKeys.results(runId),
    queryFn: () =>
      api.get<{ data: ExperimentResult[] }>(`/v1/experiments/runs/${runId}/results`),
    enabled: !!runId,
  });
}

/**
 * Compare multiple experiment runs
 */
export function useExperimentComparison(runIds: string[]) {
  return useQuery({
    queryKey: experimentKeys.comparison(runIds),
    queryFn: () =>
      api.post<ExperimentComparison>('/v1/experiments/compare', { runIds }),
    enabled: runIds.length >= 2,
  });
}

/**
 * Statistical comparison of multiple experiment runs
 */
export function useStatisticalComparison(runIds: string[]) {
  return useQuery({
    queryKey: experimentKeys.statisticalComparison(runIds),
    queryFn: () =>
      api.post<StatisticalComparison>('/v1/experiments/statistical-comparison', {
        runIds,
      }),
    enabled: runIds.length >= 2,
  });
}

/**
 * Create a new experiment
 */
export function useCreateExperiment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateExperimentInput) =>
      api.post<Experiment>('/v1/experiments', input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: experimentKeys.lists() });
    },
  });
}

/**
 * Execute an experiment
 */
export function useExecuteExperiment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: ExecuteExperimentInput) =>
      api.post<{ message: string; run: ExperimentRun }>(
        '/v1/experiments/execute',
        input
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: experimentKeys.runs(variables.experimentId),
      });
    },
  });
}

/**
 * Delete an experiment
 */
export function useDeleteExperiment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/experiments/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: experimentKeys.lists() });
    },
  });
}
