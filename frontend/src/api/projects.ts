import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface Organization {
  id: string;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
}

export interface Project {
  id: string;
  organizationId: string;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateOrganizationRequest {
  name: string;
}

export interface CreateProjectRequest {
  organizationId: string;
  name: string;
}

export interface UpdateProjectRequest {
  name?: string;
  settings?: Record<string, unknown>;
}

export interface ListOrganizationsParams {
  limit?: number;
  offset?: number;
}

export interface ListProjectsParams {
  limit?: number;
  offset?: number;
}

export interface ListOrganizationsResponse {
  data: Organization[];
  total: number;
  limit: number;
  offset: number;
}

export interface ListProjectsResponse {
  data: Project[];
  total: number;
  limit: number;
  offset: number;
}

// Query keys factory
export const orgKeys = {
  all: ['organizations'] as const,
  lists: () => [...orgKeys.all, 'list'] as const,
  list: (params: ListOrganizationsParams) => [...orgKeys.lists(), params] as const,
  details: () => [...orgKeys.all, 'detail'] as const,
  detail: (id: string) => [...orgKeys.details(), id] as const,
};

export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params: ListProjectsParams) => [...projectKeys.lists(), params] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectKeys.details(), id] as const,
};

// Hooks
export function useOrganizations(params: ListOrganizationsParams = {}) {
  return useQuery({
    queryKey: orgKeys.list(params),
    queryFn: () =>
      api.get<ListOrganizationsResponse>('/v1/organizations', { params }),
  });
}

export function useOrganization(id: string) {
  return useQuery({
    queryKey: orgKeys.detail(id),
    queryFn: () => api.get<Organization>(`/v1/organizations/${id}`),
    enabled: !!id,
  });
}

export function useProjects(params: ListProjectsParams = {}) {
  return useQuery({
    queryKey: projectKeys.list(params),
    queryFn: () =>
      api.get<ListProjectsResponse>('/v1/projects', { params }),
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: projectKeys.detail(id),
    queryFn: () => api.get<Project>(`/v1/projects/${id}`),
    enabled: !!id,
  });
}

export function useCreateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateOrganizationRequest) =>
      api.post<Organization>('/v1/organizations', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.lists() });
    },
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateProjectRequest) =>
      api.post<Project>('/v1/projects', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useUpdateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProjectRequest }) =>
      api.put<Project>(`/v1/projects/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
      queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) });
    },
  });
}

export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/projects/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useDeleteOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/organizations/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.lists() });
    },
  });
}
