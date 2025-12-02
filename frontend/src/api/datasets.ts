import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import {useProjectContext} from "@/contexts/project-context.tsx";

// ============================================================================
// Types
// ============================================================================

export interface Dataset {
  id: string;
  projectId: string;
  name: string;
  description?: string;
  itemCount?: number;
  createdAt: string;
  updatedAt: string;
}

export interface DatasetItem {
  id: string;
  datasetId: string;
  input: Record<string, any>;
  expectedOutput?: Record<string, any>;
  metadata?: Record<string, any>;
  createdAt: string;
}

export interface CreateDatasetInput {
  projectId: string;
  name: string;
  description?: string;
}

export interface UpdateDatasetInput {
  name?: string;
  description?: string;
}

export interface CreateDatasetItemInput {
  datasetId: string;
  input: Record<string, any>;
  expectedOutput?: Record<string, any>;
  metadata?: Record<string, any>;
}

export interface UpdateDatasetItemInput {
  input?: Record<string, any>;
  expectedOutput?: Record<string, any>;
  metadata?: Record<string, any>;
}

export interface DatasetImportInput {
  datasetId: string;
  format: 'json' | 'csv';
  data: string;
}

export interface ListDatasetsParams {
  projectId?: string;
  limit?: number;
  offset?: number;
}

export interface ListDatasetItemsParams {
  limit?: number;
  offset?: number;
}

// ============================================================================
// Query Keys
// ============================================================================

export const datasetKeys = {
  all: ['datasets'] as const,
  lists: () => [...datasetKeys.all, 'list'] as const,
  list: (params: ListDatasetsParams) => [...datasetKeys.lists(), params] as const,
  details: () => [...datasetKeys.all, 'detail'] as const,
  detail: (id: string) => [...datasetKeys.details(), id] as const,
  items: (datasetId: string, params?: ListDatasetItemsParams) =>
    [...datasetKeys.all, 'items', datasetId, params] as const,
  item: (itemId: string) => [...datasetKeys.all, 'item', itemId] as const,
};

// ============================================================================
// Hooks
// ============================================================================

/**
 * List datasets for a project
 */
export function useDatasets(params: Omit<ListDatasetsParams, 'projectId'> = {}) {
  const { selectedProject } = useProjectContext();
  const projectId = selectedProject?.id;

  return useQuery({
    queryKey: datasetKeys.list({ ...params, projectId }),
    queryFn: () =>
      api.get<{
        data: Dataset[];
        total: number;
        limit: number;
        offset: number;
      }>('/v1/datasets', { params: { ...params, projectId }}),
    enabled: !!projectId,
  });
}

/**
 * Get a single dataset by ID
 */
export function useDataset(id: string) {
  return useQuery({
    queryKey: datasetKeys.detail(id),
    queryFn: () => api.get<Dataset>(`/v1/datasets/${id}`),
    enabled: !!id,
  });
}

/**
 * List items in a dataset
 */
export function useDatasetItems(datasetId: string, params?: ListDatasetItemsParams) {
  return useQuery({
    queryKey: datasetKeys.items(datasetId, params),
    queryFn: () =>
      api.get<{
        data: DatasetItem[];
        total: number;
        limit: number;
        offset: number;
      }>(`/v1/datasets/${datasetId}/items`, { params }),
    enabled: !!datasetId,
  });
}

/**
 * Get a single dataset item by ID
 */
export function useDatasetItem(itemId: string) {
  return useQuery({
    queryKey: datasetKeys.item(itemId),
    queryFn: () => api.get<DatasetItem>(`/v1/datasets/items/${itemId}`),
    enabled: !!itemId,
  });
}

/**
 * Create a new dataset
 */
export function useCreateDataset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateDatasetInput) =>
      api.post<Dataset>('/v1/datasets', input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: datasetKeys.lists() });
    },
  });
}

/**
 * Update a dataset
 */
export function useUpdateDataset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateDatasetInput }) =>
      api.put<Dataset>(`/v1/datasets/${id}`, input),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: datasetKeys.detail(variables.id) });
      queryClient.invalidateQueries({ queryKey: datasetKeys.lists() });
    },
  });
}

/**
 * Delete a dataset
 */
export function useDeleteDataset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/datasets/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: datasetKeys.lists() });
    },
  });
}

/**
 * Create a dataset item
 */
export function useCreateDatasetItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateDatasetItemInput) =>
      api.post<DatasetItem>('/v1/datasets/items', input),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: datasetKeys.items(variables.datasetId),
      });
      queryClient.invalidateQueries({
        queryKey: datasetKeys.detail(variables.datasetId),
      });
    },
  });
}

/**
 * Update a dataset item
 */
export function useUpdateDatasetItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      itemId,
      datasetId,
      input,
    }: {
      itemId: string;
      datasetId: string;
      input: UpdateDatasetItemInput;
    }) => api.put<DatasetItem>(`/v1/datasets/items/${itemId}`, input),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: datasetKeys.item(variables.itemId),
      });
      void queryClient.invalidateQueries({
        queryKey: datasetKeys.items(variables.datasetId),
      });
    },
  });
}

/**
 * Delete a dataset item
 */
export function useDeleteDatasetItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ itemId, datasetId }: { itemId: string; datasetId: string }) =>
      api.delete(`/v1/datasets/items/${itemId}`),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: datasetKeys.items(variables.datasetId),
      });
      queryClient.invalidateQueries({
        queryKey: datasetKeys.detail(variables.datasetId),
      });
    },
  });
}

/**
 * Import dataset items from JSON or CSV
 */
export function useImportDataset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: DatasetImportInput) =>
      api.post<{ message: string; count: number }>('/v1/datasets/import', input),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: datasetKeys.items(variables.datasetId),
      });
      queryClient.invalidateQueries({
        queryKey: datasetKeys.detail(variables.datasetId),
      });
    },
  });
}
