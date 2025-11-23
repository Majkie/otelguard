import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

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
  name: string;
  type: string;
  config: Record<string, unknown>;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ListPoliciesParams {
  limit?: number;
  offset?: number;
  projectId?: string;
  enabled?: boolean;
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
  priority?: number;
  triggers?: Record<string, unknown>;
  projectId?: string;
}

export interface UpdatePolicyRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  triggers?: Record<string, unknown>;
}

// Query keys factory
export const guardrailKeys = {
  all: ['guardrails'] as const,
  policies: () => [...guardrailKeys.all, 'policies'] as const,
  policyLists: () => [...guardrailKeys.policies(), 'list'] as const,
  policyList: (params: ListPoliciesParams) => [...guardrailKeys.policyLists(), params] as const,
  policyDetails: () => [...guardrailKeys.policies(), 'detail'] as const,
  policyDetail: (id: string) => [...guardrailKeys.policyDetails(), id] as const,
  rules: (policyId: string) => [...guardrailKeys.policyDetail(policyId), 'rules'] as const,
};

// Hooks with project context
export function useGuardrailPolicies(params: Omit<ListPoliciesParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: guardrailKeys.policyList({ ...params, projectId }),
    queryFn: () =>
      api.get<ListPoliciesResponse>('/v1/guardrails/policies', {
        params: { ...params, projectId }
      }),
    enabled: !!projectId,
  });
}

export function useGuardrailPolicy(id: string) {
  return useQuery({
    queryKey: guardrailKeys.policyDetail(id),
    queryFn: () => api.get<GuardrailPolicy>(`/v1/guardrails/policies/${id}`),
    enabled: !!id,
  });
}

export function useGuardrailRules(policyId: string) {
  return useQuery({
    queryKey: guardrailKeys.rules(policyId),
    queryFn: () =>
      api.get<{ data: GuardrailRule[] }>(`/v1/guardrails/policies/${policyId}/rules`),
    enabled: !!policyId,
  });
}

export function useCreateGuardrailPolicy() {
  const queryClient = useQueryClient();
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: (data: Omit<CreatePolicyRequest, 'projectId'>) => {
      const projectId = selectedProject?.id;
      if (!projectId) {
        throw new Error('No project selected');
      }
      return api.post<GuardrailPolicy>('/v1/guardrails/policies', { ...data, projectId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.policyLists() });
    },
  });
}

export function useUpdateGuardrailPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdatePolicyRequest }) =>
      api.put<GuardrailPolicy>(`/v1/guardrails/policies/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.policyLists() });
      queryClient.invalidateQueries({ queryKey: guardrailKeys.policyDetail(id) });
    },
  });
}

export function useDeleteGuardrailPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/guardrails/policies/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: guardrailKeys.policyLists() });
    },
  });
}

