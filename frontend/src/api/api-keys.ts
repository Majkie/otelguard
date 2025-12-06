import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface APIKey {
  id: string;
  projectId: string;
  name: string;
  keyPrefix: string;
  scopes: string[];
  lastUsedAt?: string;
  expiresAt?: string;
  createdAt: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  scopes?: string[];
  expiresAt?: string;
}

export interface CreateAPIKeyResponse {
  id: string;
  key: string; // Only returned once
  keyPrefix: string;
  message: string;
}

export interface ListAPIKeysResponse {
  data: APIKey[];
  total: number;
}

// Query keys factory
export const apiKeyKeys = {
  all: ['api-keys'] as const,
  lists: () => [...apiKeyKeys.all, 'list'] as const,
  list: (projectId: string) => [...apiKeyKeys.lists(), projectId] as const,
};

// Hooks
export function useAPIKeys(projectId: string) {
  return useQuery({
    queryKey: apiKeyKeys.list(projectId),
    queryFn: () =>
      api.get<ListAPIKeysResponse>(`/v1/projects/${projectId}/api-keys`),
    enabled: !!projectId,
  });
}

export function useCreateAPIKey(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateAPIKeyRequest) =>
      api.post<CreateAPIKeyResponse>(`/v1/projects/${projectId}/api-keys`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list(projectId) });
    },
  });
}

export function useRevokeAPIKey(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (keyId: string) =>
      api.delete(`/v1/projects/${projectId}/api-keys/${keyId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list(projectId) });
    },
  });
}
