import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

// Types
export interface Evaluator {
  id: string;
  projectId: string;
  name: string;
  description: string;
  type: 'llm_judge' | 'rule_based' | 'custom';
  provider: 'openai' | 'anthropic' | 'google' | 'ollama';
  model: string;
  template: string;
  config: Record<string, any>;
  outputType: 'numeric' | 'boolean' | 'categorical';
  minValue?: number;
  maxValue?: number;
  categories?: string[];
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface EvaluatorTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  template: string;
  variables: string[];
  outputType: 'numeric' | 'boolean' | 'categorical';
  minValue?: number;
  maxValue?: number;
  categories?: string[];
}

export interface EvaluationJob {
  id: string;
  projectId: string;
  evaluatorId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  targetType: 'trace' | 'span';
  targetIds: string[];
  totalItems: number;
  completed: number;
  failed: number;
  startedAt?: string;
  completedAt?: string;
  totalCost: number;
  totalTokens: number;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export interface EvaluationResult {
  id: string;
  jobId?: string;
  evaluatorId: string;
  projectId: string;
  traceId: string;
  spanId?: string;
  score: number;
  stringValue?: string;
  reasoning?: string;
  rawResponse: string;
  promptTokens: number;
  completionTokens: number;
  cost: number;
  latencyMs: number;
  status: 'success' | 'error';
  errorMessage?: string;
  createdAt: string;
}

export interface EvaluationStats {
  evaluatorId: string;
  totalEvaluations: number;
  successCount: number;
  errorCount: number;
  avgScore: number;
  minScore: number;
  maxScore: number;
  totalCost: number;
  totalTokens: number;
  avgLatencyMs: number;
}

export interface CostSummary {
  evaluatorId: string;
  evaluatorName: string;
  totalCost: number;
  totalTokens: number;
  evaluationCount: number;
  avgCostPerEval: number;
}

export interface ListEvaluatorsParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  type?: string;
  provider?: string;
  outputType?: string;
  enabled?: boolean;
  search?: string;
}

export interface ListEvaluatorsResponse {
  evaluators: Evaluator[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface ListJobsParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  evaluatorId?: string;
  status?: string;
}

export interface ListJobsResponse {
  jobs: EvaluationJob[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface ListResultsParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  evaluatorId?: string;
  jobId?: string;
  traceId?: string;
  status?: string;
  startTime?: string;
  endTime?: string;
}

export interface ListResultsResponse {
  results: EvaluationResult[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface CreateEvaluatorRequest {
  name: string;
  description?: string;
  type: 'llm_judge' | 'rule_based' | 'custom';
  provider: 'openai' | 'anthropic' | 'google' | 'ollama';
  model: string;
  template: string;
  config?: Record<string, any>;
  outputType: 'numeric' | 'boolean' | 'categorical';
  minValue?: number;
  maxValue?: number;
  categories?: string[];
  enabled?: boolean;
}

export interface UpdateEvaluatorRequest {
  name?: string;
  description?: string;
  provider?: string;
  model?: string;
  template?: string;
  config?: Record<string, any>;
  outputType?: string;
  minValue?: number;
  maxValue?: number;
  categories?: string[];
  enabled?: boolean;
}

export interface RunEvaluationRequest {
  evaluatorId: string;
  traceId: string;
  spanId?: string;
}

export interface BatchEvaluationRequest {
  evaluatorId: string;
  targetType: 'trace' | 'span';
  targetIds: string[];
}

// Query keys
export const evaluatorKeys = {
  all: ['evaluators'] as const,
  lists: () => [...evaluatorKeys.all, 'list'] as const,
  list: (params: ListEvaluatorsParams) => [...evaluatorKeys.lists(), params] as const,
  details: () => [...evaluatorKeys.all, 'detail'] as const,
  detail: (id: string) => [...evaluatorKeys.details(), id] as const,
  templates: () => [...evaluatorKeys.all, 'templates'] as const,
  template: (id: string) => [...evaluatorKeys.templates(), id] as const,
  jobs: () => [...evaluatorKeys.all, 'jobs'] as const,
  jobList: (params: ListJobsParams) => [...evaluatorKeys.jobs(), params] as const,
  job: (id: string) => [...evaluatorKeys.jobs(), 'detail', id] as const,
  results: () => [...evaluatorKeys.all, 'results'] as const,
  resultList: (params: ListResultsParams) => [...evaluatorKeys.results(), params] as const,
  stats: (params: { projectId?: string; evaluatorId?: string }) => [...evaluatorKeys.all, 'stats', params] as const,
  costs: (params: { projectId?: string; startTime?: string; endTime?: string }) => [...evaluatorKeys.all, 'costs', params] as const,
};

// Hooks

// Evaluators
export function useEvaluators(params: Omit<ListEvaluatorsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListEvaluatorsResponse>('/v1/evaluators', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useEvaluator(id: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.detail(id),
    queryFn: () =>
      api.get<Evaluator>(`/v1/evaluators/${id}`, {
        params: { projectId }
      }),
    enabled: !!id && !!projectId,
  });
}

export function useCreateEvaluator() {
  const queryClient = useQueryClient();
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: (data: CreateEvaluatorRequest) =>
      api.post<Evaluator>('/v1/evaluators', {
        ...data,
        projectId: selectedProject?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.lists() });
    },
  });
}

export function useUpdateEvaluator() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateEvaluatorRequest }) =>
      api.put<Evaluator>(`/v1/evaluators/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.lists() });
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.detail(id) });
    },
  });
}

export function useDeleteEvaluator() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      api.delete(`/v1/evaluators/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.lists() });
    },
  });
}

// Templates
export function useEvaluatorTemplates() {
  return useQuery({
    queryKey: evaluatorKeys.templates(),
    queryFn: () =>
      api.get<{ templates: EvaluatorTemplate[] }>('/v1/evaluators/templates'),
  });
}

export function useEvaluatorTemplate(id: string) {
  return useQuery({
    queryKey: evaluatorKeys.template(id),
    queryFn: () =>
      api.get<EvaluatorTemplate>(`/v1/evaluators/templates/${id}`),
    enabled: !!id,
  });
}

// Evaluations
export function useRunEvaluation() {
  const queryClient = useQueryClient();
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: (data: RunEvaluationRequest) =>
      api.post<EvaluationResult>('/v1/evaluations/run', {
        ...data,
        projectId: selectedProject?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.results() });
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.stats({}) });
    },
  });
}

export function useBatchEvaluation() {
  const queryClient = useQueryClient();
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: (data: BatchEvaluationRequest) =>
      api.post<EvaluationJob>('/v1/evaluations/batch', {
        ...data,
        projectId: selectedProject?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: evaluatorKeys.jobs() });
    },
  });
}

// Jobs
export function useEvaluationJobs(params: Omit<ListJobsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.jobList({ ...params, projectId }),
    queryFn: () =>
      api.get<ListJobsResponse>('/v1/evaluations/jobs', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useEvaluationJob(id: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.job(id),
    queryFn: () =>
      api.get<EvaluationJob>(`/v1/evaluations/jobs/${id}`, {
        params: { projectId }
      }),
    enabled: !!id && !!projectId,
    refetchInterval: (data) => {
      // Refetch every 2 seconds if job is still running
      if (data?.status === 'pending' || data?.status === 'running') {
        return 2000;
      }
      return false;
    },
  });
}

// Results
export function useEvaluationResults(params: Omit<ListResultsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.resultList({ ...params, projectId }),
    queryFn: () =>
      api.get<ListResultsResponse>('/v1/evaluations/results', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

// Stats
export function useEvaluationStats(evaluatorId?: string) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.stats({ projectId, evaluatorId }),
    queryFn: () =>
      api.get<EvaluationStats>('/v1/evaluations/stats', {
        params: { projectId, evaluatorId }
      }),
    enabled: !!projectId,
  });
}

// Costs
export function useEvaluationCosts(params: { startTime?: string; endTime?: string } = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: evaluatorKeys.costs({ ...params, projectId }),
    queryFn: () =>
      api.get<{ costs: CostSummary[] }>('/v1/evaluations/costs', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}
