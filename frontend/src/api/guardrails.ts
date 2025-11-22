import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface GuardrailPolicy {
  id: string;
  projectId: string;
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  triggers?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface GuardrailRule {
  id: string;
  policyId: string;
  type: string;
  config?: Record<string, unknown>;
  action: string;
  actionConfig?: Record<string, unknown>;
  orderIndex: number;
  createdAt: string;
}

export interface ListPoliciesParams {
  limit?: number;
  offset?: number;
}

export interface ListPoliciesResponse {
  data: GuardrailPolicy[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreatePolicyRequest {
  name: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  triggers?: Record<string, unknown>;
}

export interface UpdatePolicyRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  triggers?: Record<string, unknown>;
}

export interface EvaluateRequest {
  input: string;
  output?: string;
  context?: Record<string, unknown>;
  traceId?: string;
  policyId?: string;
}

export interface EvaluateResponse {
  passed: boolean;
  violations: {
    ruleId: string;
    ruleType: string;
    message: string;
    action: string;
    actionTaken: boolean;
  }[];
  remediated: boolean;
  output?: string;
  latencyMs: number;
  evaluationId: string;
}

// Query keys factory
export const guardrailKeys = {
  all: ['guardrails'] as const,
  lists: () => [...guardrailKeys.all, 'list'] as const,
  list: (params: ListPoliciesParams) => [...guardrailKeys.lists(), params] as const,
  details: () => [...guardrailKeys.all, 'detail'] as const,
  detail: (id: string) => [...guardrailKeys.details(), id] as const,
  rules: (id: string) => [...guardrailKeys.detail(id), 'rules'] as const,
};

// Hooks
export function useGuardrailPolicies(params: ListPoliciesParams = {}) {
  return useQuery({
    queryKey: guardrailKeys.list(params),
    queryFn: () =>
      api.get<ListPoliciesResponse>('/v1/guardrails', { params }),
  });
}

export function useGuardrailPolicy(id: string) {
  return useQuery({
    queryKey: guardrailKeys.detail(id),
    queryFn: () => api.get<GuardrailPolicy>(`/v1/guardrails/${id}`),
    enabled: !!id,
  });
}

export function useGuardrailRules(policyId: string) {
  return useQuery({
    queryKey: guardrailKeys.rules(policyId),
    queryFn: () =>
      api.get<{ data: GuardrailRule[]; total: number }>(
        `/v1/guardrails/${policyId}/rules`
      ),
    enabled: !!policyId,
  });
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreatePolicyRequest) =>
      api.post<GuardrailPolicy>('/v1/guardrails', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.lists() });
    },
  });
}

export function useUpdatePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdatePolicyRequest }) =>
      api.put<GuardrailPolicy>(`/v1/guardrails/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.lists() });
      queryClient.invalidateQueries({ queryKey: guardrailKeys.detail(id) });
    },
  });
}

export function useDeletePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/guardrails/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.lists() });
    },
  });
}

export function useEvaluateGuardrail() {
  return useMutation({
    mutationFn: (data: EvaluateRequest) =>
      api.post<EvaluateResponse>('/v1/guardrails/evaluate', data),
  });
}
