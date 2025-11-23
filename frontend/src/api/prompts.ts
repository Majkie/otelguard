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

export interface CreateVersionRequest {
  content: string;
  config?: Record<string, unknown>;
  labels?: string[];
}

export interface UpdateVersionLabelsRequest {
  labels: string[];
}

export interface CompilePromptRequest {
  variables?: Record<string, unknown>;
  version?: number;
}

export interface CompilePromptResponse {
  id: string;
  compiled: string;
  variables: string[];
  missing?: string[];
  errors?: string[];
}

export interface CompareVersionsResponse {
  promptId: string;
  v1: {
    version: number;
    content: string;
    labels?: string[];
    createdAt: string;
  };
  v2: {
    version: number;
    content: string;
    labels?: string[];
    createdAt: string;
  };
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
    mutationFn: ({ promptId, data }: { promptId: string; data: CreateVersionRequest }) =>
      api.post<PromptVersion>(`/v1/prompts/${promptId}/versions`, data),
    onSuccess: (_, { promptId }) => {
      queryClient.invalidateQueries({ queryKey: promptKeys.versions(promptId) });
      queryClient.invalidateQueries({ queryKey: promptKeys.detail(promptId) });
    },
  });
}

export function useUpdateVersionLabels() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      promptId,
      version,
      data,
    }: {
      promptId: string;
      version: number;
      data: UpdateVersionLabelsRequest;
    }) =>
      api.put<PromptVersion>(
        `/v1/prompts/${promptId}/versions/${version}/labels`,
        data
      ),
    onSuccess: (_, { promptId }) => {
      queryClient.invalidateQueries({ queryKey: promptKeys.versions(promptId) });
    },
  });
}

// Duplication hook
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

// Compare versions hook
export function useCompareVersions(promptId: string, v1: number, v2: number) {
  return useQuery({
    queryKey: promptKeys.compare(promptId, v1, v2),
    queryFn: () =>
      api.get<CompareVersionsResponse>(
        `/v1/prompts/${promptId}/compare?v1=${v1}&v2=${v2}`
      ),
    enabled: !!promptId && v1 > 0 && v2 > 0,
  });
}

// Compile template hook
export function useCompilePrompt() {
  return useMutation({
    mutationFn: ({
      promptId,
      data,
    }: {
      promptId: string;
      data: CompilePromptRequest;
    }) => api.post<CompilePromptResponse>(`/v1/prompts/${promptId}/compile`, data),
  });
}

// Extract variables hook
export function useExtractVariables() {
  return useMutation({
    mutationFn: (content: string) =>
      api.post<{ variables: string[] }>('/v1/prompts/extract-variables', {
        content,
      }),
  });
}

// Version promotion hook
export function usePromoteVersion() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      promptId,
      version,
      target,
    }: {
      promptId: string;
      version: number;
      target: 'production' | 'staging' | 'development';
    }) =>
      api.post<PromptVersion>(
        `/v1/prompts/${promptId}/versions/${version}/promote`,
        { target }
      ),
    onSuccess: (_, { promptId }) => {
      queryClient.invalidateQueries({ queryKey: promptKeys.versions(promptId) });
    },
  });
}

// Get version by label hook
export function useVersionByLabel(promptId: string, label: string) {
  return useQuery({
    queryKey: [...promptKeys.versions(promptId), 'label', label],
    queryFn: () =>
      api.get<PromptVersion>(`/v1/prompts/${promptId}/versions/by-label/${label}`),
    enabled: !!promptId && !!label,
  });
}

// Prompt analytics types
export interface PromptAnalytics {
  promptId: string;
  promptName: string;
  totalVersions: number;
  latestVersion: number;
  productionVersion?: number;
  stagingVersion?: number;
  developmentVersion?: number;
  versions: {
    version: number;
    labels?: string[];
    createdAt: string;
  }[];
  createdAt: string;
  updatedAt: string;
}

// Prompt analytics hook
export function usePromptAnalytics(promptId: string) {
  return useQuery({
    queryKey: [...promptKeys.detail(promptId), 'analytics'],
    queryFn: () => api.get<PromptAnalytics>(`/v1/prompts/${promptId}/analytics`),
    enabled: !!promptId,
  });
}

// Linked traces types
export interface LinkedTrace {
  id: string;
  name: string;
  startTime: string;
  latencyMs: number;
  status: string;
  promptVersion?: number;
}

export interface LinkedTracesResponse {
  promptId: string;
  traces: LinkedTrace[];
  total: number;
  message?: string;
}

// Get traces linked to a prompt
export function useLinkedTraces(promptId: string) {
  return useQuery({
    queryKey: [...promptKeys.detail(promptId), 'traces'],
    queryFn: () => api.get<LinkedTracesResponse>(`/v1/prompts/${promptId}/traces`),
    enabled: !!promptId,
  });
}
