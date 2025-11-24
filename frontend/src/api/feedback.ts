import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import { useProjectContext } from '@/contexts/project-context';

// Types
export interface Feedback {
  id: string;
  projectId: string;
  userId?: string;
  sessionId?: string;
  traceId?: string;
  spanId?: string;
  itemType: 'trace' | 'session' | 'span' | 'prompt';
  itemId: string;
  thumbsUp?: boolean;
  rating?: number;
  comment?: string;
  metadata?: Record<string, any>;
  userAgent?: string;
  ipAddress?: string;
  createdAt: string;
  updatedAt: string;
}

export interface FeedbackAnalytics {
  projectId: string;
  itemType: string;
  totalFeedback: number;
  thumbsUpCount: number;
  thumbsDownCount: number;
  averageRating?: number;
  ratingCounts: Record<number, number>;
  commentCount: number;
  dateRange: string;
  trends?: FeedbackTrend[];
}

export interface FeedbackTrend {
  date: string;
  totalFeedback: number;
  thumbsUpRate: number;
  averageRating: number;
  commentCount: number;
}

export interface FeedbackFilter {
  projectId?: string;
  userId?: string;
  itemType?: string;
  itemId?: string;
  traceId?: string;
  sessionId?: string;
  thumbsUp?: boolean;
  rating?: number;
  startDate?: string;
  endDate?: string;
  orderBy?: string;
  orderDesc?: boolean;
  limit?: number;
  offset?: number;
}

export interface CreateFeedbackRequest {
  projectId: string;
  userId?: string;
  sessionId?: string;
  traceId?: string;
  spanId?: string;
  itemType: 'trace' | 'session' | 'span' | 'prompt';
  itemId: string;
  thumbsUp?: boolean;
  rating?: number;
  comment?: string;
  metadata?: Record<string, any>;
}

export interface UpdateFeedbackRequest {
  thumbsUp?: boolean;
  rating?: number;
  comment?: string;
  metadata?: Record<string, any>;
}

export interface FeedbackListResponse {
  feedback: Feedback[];
  total: number;
  limit: number;
  offset: number;
}

// Query Keys
export const feedbackKeys = {
  all: ['feedback'] as const,
  lists: () => [...feedbackKeys.all, 'list'] as const,
  list: (filters: FeedbackFilter) => [...feedbackKeys.lists(), filters] as const,
  details: () => [...feedbackKeys.all, 'detail'] as const,
  detail: (id: string) => [...feedbackKeys.details(), id] as const,
  analytics: () => [...feedbackKeys.all, 'analytics'] as const,
  analyticsDetail: (params: { projectId: string; itemType: string; startDate: string; endDate: string }) =>
    [...feedbackKeys.analytics(), params] as const,
  trends: () => [...feedbackKeys.all, 'trends'] as const,
  trendsDetail: (params: { projectId: string; itemType: string; startDate: string; endDate: string; interval: string }) =>
    [...feedbackKeys.trends(), params] as const,
};

// Hooks
export function useFeedbackList(filters: FeedbackFilter) {
  const { selectedProject } = useProjectContext();

  return useQuery({
    queryKey: feedbackKeys.list({ ...filters, projectId: selectedProject?.id }),
    queryFn: async () => {
      const params = new URLSearchParams();

      // Add filters to query params
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          if (typeof value === 'boolean') {
            params.append(key, value.toString());
          } else {
            params.append(key, value.toString());
          }
        }
      });

      // Always filter by current project
      if (selectedProject?.id) {
        params.set('projectId', selectedProject.id);
      }

      const response = await api.get<FeedbackListResponse>(`/v1/feedback?${params.toString()}`);
      return response;
    },
    enabled: !!selectedProject?.id,
  });
}

export function useFeedbackDetail(id: string) {
  return useQuery({
    queryKey: feedbackKeys.detail(id),
    queryFn: async () => {
      const response = await api.get<Feedback>(`/v1/feedback/${id}`);
      return response;
    },
    enabled: !!id,
  });
}

export function useFeedbackAnalytics(
  itemType: string,
  startDate: string,
  endDate: string
) {
  const { selectedProject } = useProjectContext();

  return useQuery({
    queryKey: feedbackKeys.analyticsDetail({
      projectId: selectedProject?.id || '',
      itemType,
      startDate,
      endDate,
    }),
    queryFn: async () => {
      const params = new URLSearchParams({
        projectId: selectedProject?.id || '',
        itemType,
        startDate,
        endDate,
      });

      const response = await api.get<FeedbackAnalytics>(`/v1/feedback/analytics?${params.toString()}`);
      return response;
    },
    enabled: !!selectedProject?.id && !!itemType,
  });
}

export function useFeedbackTrends(
  itemType: string,
  startDate: string,
  endDate: string,
  interval: 'hour' | 'day' | 'week' | 'month' = 'day'
) {
  const { selectedProject } = useProjectContext();

  return useQuery({
    queryKey: feedbackKeys.trendsDetail({
      projectId: selectedProject?.id || '',
      itemType,
      startDate,
      endDate,
      interval,
    }),
    queryFn: async () => {
      const params = new URLSearchParams({
        projectId: selectedProject?.id || '',
        itemType,
        startDate,
        endDate,
        interval,
      });

      const response = await api.get<FeedbackTrend[]>(`/v1/feedback/trends?${params.toString()}`);
      return response;
    },
    enabled: !!selectedProject?.id && !!itemType,
  });
}

export function useCreateFeedback() {
  const queryClient = useQueryClient();
  const { selectedProject } = useProjectContext();

  return useMutation({
    mutationFn: async (feedback: CreateFeedbackRequest) => {
      // Ensure project ID is set
      feedback.projectId = selectedProject?.id || feedback.projectId;

      const response = await api.post<Feedback>('/v1/feedback', feedback);
      return response;
    },
    onSuccess: () => {
      // Invalidate feedback lists
      queryClient.invalidateQueries({ queryKey: feedbackKeys.lists() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.analytics() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.trends() });
    },
  });
}

export function useUpdateFeedback() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, updates }: { id: string; updates: UpdateFeedbackRequest }) => {
      const response = await api.put<Feedback>(`/v1/feedback/${id}`, updates);
      return response;
    },
    onSuccess: (data) => {
      // Update the specific feedback item
      queryClient.setQueryData(feedbackKeys.detail(data.id), data);
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: feedbackKeys.lists() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.analytics() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.trends() });
    },
  });
}

export function useDeleteFeedback() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/v1/feedback/${id}`);
      return id;
    },
    onSuccess: (id) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: feedbackKeys.detail(id) });
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: feedbackKeys.lists() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.analytics() });
      queryClient.invalidateQueries({ queryKey: feedbackKeys.trends() });
    },
  });
}
