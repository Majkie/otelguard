import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface AlertRule {
  id: string;
  project_id: string;
  name: string;
  description?: string;
  enabled: boolean;
  metric_type: string;
  metric_field?: string;
  condition_type: string;
  operator: string;
  threshold_value: number;
  window_duration: number;
  evaluation_frequency: number;
  filters: Record<string, any>;
  notification_channels: string[];
  notification_message?: string;
  escalation_policy_id?: string;
  group_by: string[];
  group_wait: number;
  repeat_interval: number;
  severity: string;
  tags: string[];
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface AlertHistory {
  id: string;
  alert_rule_id: string;
  project_id: string;
  status: string;
  severity: string;
  metric_value?: number;
  threshold_value?: number;
  fired_at: string;
  resolved_at?: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  fingerprint: string;
  group_labels: Record<string, any>;
  notification_sent: boolean;
  notification_channels: string[];
  notification_error?: string;
  message?: string;
  annotations: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertRuleRequest {
  name: string;
  description?: string;
  enabled: boolean;
  metric_type: string;
  metric_field?: string;
  condition_type: string;
  operator: string;
  threshold_value: number;
  window_duration?: number;
  evaluation_frequency?: number;
  filters?: Record<string, any>;
  notification_channels: string[];
  notification_message?: string;
  escalation_policy_id?: string;
  group_by?: string[];
  group_wait?: number;
  repeat_interval?: number;
  severity?: string;
  tags?: string[];
}

export interface ListAlertsParams {
  limit?: number;
  offset?: number;
}

export interface ListResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

// Query keys
export const alertKeys = {
  all: ['alerts'] as const,
  rules: (projectId: string) => [...alertKeys.all, 'rules', projectId] as const,
  rule: (projectId: string, ruleId: string) => [...alertKeys.all, 'rules', projectId, ruleId] as const,
  history: (projectId: string) => [...alertKeys.all, 'history', projectId] as const,
  historyItem: (projectId: string, alertId: string) => [...alertKeys.all, 'history', projectId, alertId] as const,
};

// Hooks
export function useAlertRules(projectId: string, params: ListAlertsParams = {}) {
  return useQuery({
    queryKey: [...alertKeys.rules(projectId), params],
    queryFn: () =>
      api.get<ListResponse<AlertRule>>(`/v1/projects/${projectId}/alerts/rules`, { params }),
    enabled: !!projectId,
  });
}

export function useAlertRule(projectId: string, ruleId: string) {
  return useQuery({
    queryKey: alertKeys.rule(projectId, ruleId),
    queryFn: () => api.get<AlertRule>(`/v1/projects/${projectId}/alerts/rules/${ruleId}`),
    enabled: !!projectId && !!ruleId,
  });
}

export function useCreateAlertRule(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateAlertRuleRequest) =>
      api.post<AlertRule>(`/v1/projects/${projectId}/alerts/rules`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules(projectId) });
    },
  });
}

export function useUpdateAlertRule(projectId: string, ruleId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: Partial<CreateAlertRuleRequest>) =>
      api.put<AlertRule>(`/v1/projects/${projectId}/alerts/rules/${ruleId}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules(projectId) });
      queryClient.invalidateQueries({ queryKey: alertKeys.rule(projectId, ruleId) });
    },
  });
}

export function useDeleteAlertRule(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ruleId: string) =>
      api.delete(`/v1/projects/${projectId}/alerts/rules/${ruleId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules(projectId) });
    },
  });
}

export function useAlertHistory(projectId: string, params: ListAlertsParams = {}) {
  return useQuery({
    queryKey: [...alertKeys.history(projectId), params],
    queryFn: () =>
      api.get<ListResponse<AlertHistory>>(`/v1/projects/${projectId}/alerts/history`, { params }),
    enabled: !!projectId,
  });
}

export function useAlertHistoryItem(projectId: string, alertId: string) {
  return useQuery({
    queryKey: alertKeys.historyItem(projectId, alertId),
    queryFn: () => api.get<AlertHistory>(`/v1/projects/${projectId}/alerts/history/${alertId}`),
    enabled: !!projectId && !!alertId,
  });
}

export function useAcknowledgeAlert(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ alertId, userId }: { alertId: string; userId: string }) =>
      api.post<AlertHistory>(`/v1/projects/${projectId}/alerts/history/${alertId}/acknowledge`, {
        user_id: userId,
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: alertKeys.history(projectId) });
      queryClient.invalidateQueries({ queryKey: alertKeys.historyItem(projectId, variables.alertId) });
    },
  });
}

export function useEvaluateAlerts(projectId: string) {
  return useMutation({
    mutationFn: () => api.post(`/v1/projects/${projectId}/alerts/evaluate`),
  });
}
