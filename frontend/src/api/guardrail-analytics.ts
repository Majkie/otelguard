import { useQuery } from '@tanstack/react-query';
import { api } from './client';

// ============================================================================
// Types
// ============================================================================

export interface PolicyStats {
  policyId: string;
  policyName: string;
  evaluationCount: number;
  triggerCount: number;
  actionCount: number;
  triggerRate: number;
  actionRate: number;
  avgLatencyMs: number;
  totalLatencyMs: number;
}

export interface RuleTypeStats {
  ruleType: string;
  triggerCount: number;
  actionCount: number;
  triggerRate: number;
  avgLatencyMs: number;
}

export interface ActionStats {
  actionType: string;
  actionCount: number;
  successCount: number;
  successRate: number;
}

export interface TriggerStats {
  projectId: string;
  startTime: string;
  endTime: string;
  totalEvaluations: number;
  totalTriggered: number;
  totalActioned: number;
  triggerRate: number;
  actionRate: number;
  byPolicy: Record<string, PolicyStats>;
  byRuleType: Record<string, RuleTypeStats>;
  byAction: Record<string, ActionStats>;
}

export interface ViolationTrend {
  timestamp: string;
  evaluationCount: number;
  triggerCount: number;
  actionCount: number;
  triggerRate: number;
}

export interface RemediationSuccessRate {
  actionType: string;
  totalAttempts: number;
  successfulCount: number;
  successRate: number;
  avgLatencyMs: number;
}

export interface PolicyCostImpact {
  policyId: string;
  policyName: string;
  evaluationCount: number;
  blockedCount: number;
  latencyImpactMs: number;
  avgLatencyMs: number;
  estimatedCostSavings: number;
}

export interface CostImpactAnalysis {
  projectId: string;
  startTime: string;
  endTime: string;
  totalEvaluations: number;
  totalLatencyMs: number;
  avgLatencyMs: number;
  estimatedCostSavings: number;
  byPolicy: Record<string, PolicyCostImpact>;
}

export interface LatencyImpact {
  totalEvaluations: number;
  totalLatencyMs: number;
  avgLatencyMs: number;
  medianLatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  byRuleType: Record<
    string,
    {
      avgLatencyMs: number;
      count: number;
    }
  >;
  impactPercentage: number;
}

export interface GuardrailAnalyticsParams {
  projectId: string;
  startTime?: string;
  endTime?: string;
  interval?: '1h' | '6h' | '12h' | '1d' | '1w';
}

// ============================================================================
// Query Keys
// ============================================================================

export const guardrailAnalyticsKeys = {
  all: ['guardrail-analytics'] as const,
  triggers: (params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'triggers', params] as const,
  trends: (params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'trends', params] as const,
  remediationSuccess: (params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'remediation-success', params] as const,
  policyAnalytics: (policyId: string, params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'policy', policyId, params] as const,
  costImpact: (params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'cost-impact', params] as const,
  latencyImpact: (params: GuardrailAnalyticsParams) =>
    [...guardrailAnalyticsKeys.all, 'latency-impact', params] as const,
};

// ============================================================================
// Hooks
// ============================================================================

/**
 * Get trigger statistics
 */
export function useTriggerStats(params: GuardrailAnalyticsParams) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.triggers(params),
    queryFn: () =>
      api.get<TriggerStats>('/v1/guardrails/analytics/triggers', { params }),
    enabled: !!params.projectId,
  });
}

/**
 * Get violation trends over time
 */
export function useViolationTrends(params: GuardrailAnalyticsParams) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.trends(params),
    queryFn: () =>
      api.get<{ data: ViolationTrend[] }>('/v1/guardrails/analytics/trends', {
        params,
      }),
    enabled: !!params.projectId,
  });
}

/**
 * Get remediation success rates
 */
export function useRemediationSuccessRates(params: GuardrailAnalyticsParams) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.remediationSuccess(params),
    queryFn: () =>
      api.get<{ data: RemediationSuccessRate[] }>(
        '/v1/guardrails/analytics/remediation-success',
        { params }
      ),
    enabled: !!params.projectId,
  });
}

/**
 * Get analytics for a specific policy
 */
export function usePolicyAnalytics(
  policyId: string,
  params: GuardrailAnalyticsParams
) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.policyAnalytics(policyId, params),
    queryFn: () =>
      api.get<PolicyStats>(
        `/v1/guardrails/analytics/policies/${policyId}`,
        { params }
      ),
    enabled: !!policyId && !!params.projectId,
  });
}

/**
 * Get cost impact analysis
 */
export function useCostImpactAnalysis(params: GuardrailAnalyticsParams) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.costImpact(params),
    queryFn: () =>
      api.get<CostImpactAnalysis>(
        '/v1/guardrails/analytics/cost-impact',
        { params }
      ),
    enabled: !!params.projectId,
  });
}

/**
 * Get latency impact analysis
 */
export function useLatencyImpact(params: GuardrailAnalyticsParams) {
  return useQuery({
    queryKey: guardrailAnalyticsKeys.latencyImpact(params),
    queryFn: () =>
      api.get<LatencyImpact>('/v1/guardrails/analytics/latency-impact', {
        params,
      }),
    enabled: !!params.projectId,
  });
}
