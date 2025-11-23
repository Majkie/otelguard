import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

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
  projectId?: string;
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
  projectId?: string;
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
  version: (id: string, version: number) => [...promptKeys.versions(id), version] as const,
  compare: (id: string, v1: number, v2: number) => [...promptKeys.detail(id), 'compare', v1, v2] as const,
};

// Hooks with project context
export function usePrompts(params: Omit<ListPromptsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: promptKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<ListPromptsResponse>('/v1/prompts', { 
        params: { ...params, projectId } 
      }),
    enabled: !!projectId,
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
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: (data: Omit<CreatePromptRequest, 'projectId'>) => {
      const projectId = selectedProject?.id;
      if (!projectId) {
        throw new Error('No project selected');
      }
      return api.post<Prompt>('/v1/prompts', { ...data, projectId });
    },
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

export function useDuplicatePrompt() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api.post<Prompt>(`/v1/prompts/${id}/duplicate`, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: promptKeys.lists() });
    },
  });
}

// Version management hooks
export function usePromptVersion(promptId: string, version: number) {
  return useQuery({
    queryKey: promptKeys.version(promptId, version),
    queryFn: () =>
      api.get<PromptVersion>(`/v1/prompts/${promptId}/versions/${version}`),
    enabled: !!promptId && version > 0,
  });
}

export function useCreateVersion() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ promptId, data }: { promptId: string; data: any }) =>
      api.post<PromptVersion>(`/v1/prompts/${promptId}/versions`, data),
    onSuccess: (_, { promptId }) => {
      queryClient.invalidateQueries({ queryKey: promptKeys.versions(promptId) });
      queryClient.invalidateQueries({ queryKey: promptKeys.detail(promptId) });
    },
  });
}

// Other hooks remain the same but will use project context when needed
export function useCompilePrompt() {
  return useMutation({
    mutationFn: ({
      promptId,
      data,
    }: {
      promptId: string;
      data: any;
    }) => api.post<any>(`/v1/prompts/${promptId}/compile`, data),
  });
}
