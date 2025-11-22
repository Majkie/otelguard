import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface Prompt {
  id: string;
  projectId: string;
  name: string;
  description?: string;
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface PromptVersion {
  id: string;
  promptId: string;
  version: number;
  content: string;
  config?: Record<string, unknown>;
  labels?: string[];
  createdBy?: string;
  createdAt: string;
}

export interface ListPromptsParams {
  limit?: number;
  offset?: number;
}

export interface ListPromptsResponse {
  data: Prompt[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreatePromptRequest {
  name: string;
  description?: string;
  content?: string;
  tags?: string[];
}

export interface UpdatePromptRequest {
  name?: string;
  description?: string;
  tags?: string[];
}

// Query keys factory
export const promptKeys = {
  all: ['prompts'] as const,
  lists: () => [...promptKeys.all, 'list'] as const,
  list: (params: ListPromptsParams) => [...promptKeys.lists(), params] as const,
  details: () => [...promptKeys.all, 'detail'] as const,
  detail: (id: string) => [...promptKeys.details(), id] as const,
  versions: (id: string) => [...promptKeys.detail(id), 'versions'] as const,
};

// Hooks
export function usePrompts(params: ListPromptsParams = {}) {
  return useQuery({
    queryKey: promptKeys.list(params),
    queryFn: () =>
      api.get<ListPromptsResponse>('/v1/prompts', { params }),
  });
}

export function usePrompt(id: string) {
  return useQuery({
    queryKey: promptKeys.detail(id),
    queryFn: () => api.get<Prompt>(`/v1/prompts/${id}`),
    enabled: !!id,
  });
}

export function usePromptVersions(promptId: string) {
  return useQuery({
    queryKey: promptKeys.versions(promptId),
    queryFn: () =>
      api.get<{ data: PromptVersion[]; total: number }>(
        `/v1/prompts/${promptId}/versions`
      ),
    enabled: !!promptId,
  });
}

export function useCreatePrompt() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreatePromptRequest) =>
      api.post<Prompt>('/v1/prompts', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: promptKeys.lists() });
    },
  });
}

export function useUpdatePrompt() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdatePromptRequest }) =>
      api.put<Prompt>(`/v1/prompts/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: promptKeys.lists() });
      queryClient.invalidateQueries({ queryKey: promptKeys.detail(id) });
    },
  });
}

export function useDeletePrompt() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/prompts/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: promptKeys.lists() });
    },
  });
}
