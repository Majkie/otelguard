package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// GuardrailAnalyticsService provides analytics for guardrail policies
type GuardrailAnalyticsService struct {
	policyRepo          *postgres.GuardrailRepository
	guardrailEventsRepo *clickhouse.GuardrailEventRepository
	logger              *zap.Logger
}

// NewGuardrailAnalyticsService creates a new guardrail analytics service
func NewGuardrailAnalyticsService(
	policyRepo *postgres.GuardrailRepository,
	guardrailEventsRepo *clickhouse.GuardrailEventRepository,
	logger *zap.Logger,
) *GuardrailAnalyticsService {
	return &GuardrailAnalyticsService{
		policyRepo:          policyRepo,
		guardrailEventsRepo: guardrailEventsRepo,
		logger:              logger,
	}
}

// ============================================================================
// Data Types
// ============================================================================

// TriggerStats represents aggregated trigger statistics
type TriggerStats struct {
	ProjectID       uuid.UUID                 `json:"projectId"`
	StartTime       time.Time                 `json:"startTime"`
	EndTime         time.Time                 `json:"endTime"`
	TotalEvaluations int64                    `json:"totalEvaluations"`
	TotalTriggered  int64                     `json:"totalTriggered"`
	TotalActioned   int64                     `json:"totalActioned"`
	TriggerRate     float64                   `json:"triggerRate"`
	ActionRate      float64                   `json:"actionRate"`
	ByPolicy        map[string]*PolicyStats   `json:"byPolicy"`
	ByRuleType      map[string]*RuleTypeStats `json:"byRuleType"`
	ByAction        map[string]*ActionStats   `json:"byAction"`
}

// PolicyStats represents statistics for a specific policy
type PolicyStats struct {
	PolicyID         uuid.UUID `json:"policyId"`
	PolicyName       string    `json:"policyName"`
	EvaluationCount  int64     `json:"evaluationCount"`
	TriggerCount     int64     `json:"triggerCount"`
	ActionCount      int64     `json:"actionCount"`
	TriggerRate      float64   `json:"triggerRate"`
	ActionRate       float64   `json:"actionRate"`
	AvgLatencyMs     float64   `json:"avgLatencyMs"`
	TotalLatencyMs   int64     `json:"totalLatencyMs"`
}

// RuleTypeStats represents statistics for a specific rule type
type RuleTypeStats struct {
	RuleType        string  `json:"ruleType"`
	TriggerCount    int64   `json:"triggerCount"`
	ActionCount     int64   `json:"actionCount"`
	TriggerRate     float64 `json:"triggerRate"`
	AvgLatencyMs    float64 `json:"avgLatencyMs"`
}

// ActionStats represents statistics for a specific action type
type ActionStats struct {
	ActionType      string  `json:"actionType"`
	ActionCount     int64   `json:"actionCount"`
	SuccessCount    int64   `json:"successCount"`
	SuccessRate     float64 `json:"successRate"`
}

// ViolationTrend represents violation trends over time
type ViolationTrend struct {
	Timestamp       time.Time `json:"timestamp"`
	EvaluationCount int64     `json:"evaluationCount"`
	TriggerCount    int64     `json:"triggerCount"`
	ActionCount     int64     `json:"actionCount"`
	TriggerRate     float64   `json:"triggerRate"`
}

// RemediationSuccessRate represents success rates for remediation actions
type RemediationSuccessRate struct {
	ActionType      string  `json:"actionType"`
	TotalAttempts   int64   `json:"totalAttempts"`
	SuccessfulCount int64   `json:"successfulCount"`
	SuccessRate     float64 `json:"successRate"`
	AvgLatencyMs    float64 `json:"avgLatencyMs"`
}

// CostImpactAnalysis represents the cost impact of guardrails
type CostImpactAnalysis struct {
	ProjectID            uuid.UUID                    `json:"projectId"`
	StartTime            time.Time                    `json:"startTime"`
	EndTime              time.Time                    `json:"endTime"`
	TotalEvaluations     int64                        `json:"totalEvaluations"`
	TotalLatencyMs       int64                        `json:"totalLatencyMs"`
	AvgLatencyMs         float64                      `json:"avgLatencyMs"`
	EstimatedCostSavings float64                      `json:"estimatedCostSavings"` // From blocked requests
	ByPolicy             map[string]*PolicyCostImpact `json:"byPolicy"`
}

// PolicyCostImpact represents cost impact for a specific policy
type PolicyCostImpact struct {
	PolicyID            uuid.UUID `json:"policyId"`
	PolicyName          string    `json:"policyName"`
	EvaluationCount     int64     `json:"evaluationCount"`
	BlockedCount        int64     `json:"blockedCount"`
	LatencyImpactMs     int64     `json:"latencyImpactMs"`
	AvgLatencyMs        float64   `json:"avgLatencyMs"`
	EstimatedCostSavings float64  `json:"estimatedCostSavings"`
}

// ============================================================================
// Methods
// ============================================================================

// GetTriggerStats returns trigger statistics for a project within a time range
func (s *GuardrailAnalyticsService) GetTriggerStats(
	ctx context.Context,
	projectID uuid.UUID,
	startTime, endTime time.Time,
) (*TriggerStats, error) {
	// Mock implementation - in production, this would query ClickHouse
	stats := &TriggerStats{
		ProjectID:        projectID,
		StartTime:        startTime,
		EndTime:          endTime,
		TotalEvaluations: 10000,
		TotalTriggered:   250,
		TotalActioned:    200,
		TriggerRate:      0.025,
		ActionRate:       0.02,
		ByPolicy:         make(map[string]*PolicyStats),
		ByRuleType:       make(map[string]*RuleTypeStats),
		ByAction:         make(map[string]*ActionStats),
	}

	// Mock policy stats
	stats.ByPolicy["policy-1"] = &PolicyStats{
		PolicyID:        uuid.New(),
		PolicyName:      "PII Protection",
		EvaluationCount: 5000,
		TriggerCount:    150,
		ActionCount:     120,
		TriggerRate:     0.03,
		ActionRate:      0.024,
		AvgLatencyMs:    12.5,
		TotalLatencyMs:  1500,
	}

	stats.ByPolicy["policy-2"] = &PolicyStats{
		PolicyID:        uuid.New(),
		PolicyName:      "Content Moderation",
		EvaluationCount: 5000,
		TriggerCount:    100,
		ActionCount:     80,
		TriggerRate:     0.02,
		ActionRate:      0.016,
		AvgLatencyMs:    15.0,
		TotalLatencyMs:  1500,
	}

	// Mock rule type stats
	stats.ByRuleType["pii_detection"] = &RuleTypeStats{
		RuleType:     "pii_detection",
		TriggerCount: 150,
		ActionCount:  120,
		TriggerRate:  0.015,
		AvgLatencyMs: 10.0,
	}

	stats.ByRuleType["toxicity"] = &RuleTypeStats{
		RuleType:     "toxicity",
		TriggerCount: 100,
		ActionCount:  80,
		TriggerRate:  0.01,
		AvgLatencyMs: 20.0,
	}

	// Mock action stats
	stats.ByAction["block"] = &ActionStats{
		ActionType:   "block",
		ActionCount:  150,
		SuccessCount: 150,
		SuccessRate:  1.0,
	}

	stats.ByAction["sanitize"] = &ActionStats{
		ActionType:   "sanitize",
		ActionCount:  50,
		SuccessCount: 48,
		SuccessRate:  0.96,
	}

	return stats, nil
}

// GetViolationTrend returns violation trends over time
func (s *GuardrailAnalyticsService) GetViolationTrend(
	ctx context.Context,
	projectID uuid.UUID,
	startTime, endTime time.Time,
	interval string, // "1h", "1d", "1w"
) ([]ViolationTrend, error) {
	// Mock implementation
	trends := []ViolationTrend{}

	// Generate hourly data points
	duration := endTime.Sub(startTime)
	numPoints := 24
	if duration > 7*24*time.Hour {
		numPoints = 7 // Daily for week+
	}

	for i := 0; i < numPoints; i++ {
		timestamp := startTime.Add(time.Duration(i) * (duration / time.Duration(numPoints)))

		// Mock data with some variation
		evaluations := int64(400 + i*10)
		triggers := int64(10 + i)
		actions := int64(8 + i)

		trends = append(trends, ViolationTrend{
			Timestamp:       timestamp,
			EvaluationCount: evaluations,
			TriggerCount:    triggers,
			ActionCount:     actions,
			TriggerRate:     float64(triggers) / float64(evaluations),
		})
	}

	return trends, nil
}

// GetRemediationSuccessRates returns success rates for remediation actions
func (s *GuardrailAnalyticsService) GetRemediationSuccessRates(
	ctx context.Context,
	projectID uuid.UUID,
	startTime, endTime time.Time,
) ([]RemediationSuccessRate, error) {
	// Mock implementation
	rates := []RemediationSuccessRate{
		{
			ActionType:      "block",
			TotalAttempts:   1500,
			SuccessfulCount: 1500,
			SuccessRate:     1.0,
			AvgLatencyMs:    5.0,
		},
		{
			ActionType:      "sanitize",
			TotalAttempts:   500,
			SuccessfulCount: 480,
			SuccessRate:     0.96,
			AvgLatencyMs:    15.0,
		},
		{
			ActionType:      "retry",
			TotalAttempts:   200,
			SuccessfulCount: 180,
			SuccessRate:     0.90,
			AvgLatencyMs:    250.0,
		},
		{
			ActionType:      "fallback",
			TotalAttempts:   100,
			SuccessfulCount: 95,
			SuccessRate:     0.95,
			AvgLatencyMs:    50.0,
		},
		{
			ActionType:      "alert",
			TotalAttempts:   300,
			SuccessfulCount: 300,
			SuccessRate:     1.0,
			AvgLatencyMs:    2.0,
		},
		{
			ActionType:      "transform",
			TotalAttempts:   150,
			SuccessfulCount: 145,
			SuccessRate:     0.967,
			AvgLatencyMs:    25.0,
		},
	}

	return rates, nil
}

// GetPolicyAnalytics returns detailed analytics for a specific policy
func (s *GuardrailAnalyticsService) GetPolicyAnalytics(
	ctx context.Context,
	policyID uuid.UUID,
	startTime, endTime time.Time,
) (*PolicyStats, error) {
	// Get policy details
	policy, err := s.policyRepo.GetByID(ctx, policyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// Mock analytics - in production, query ClickHouse
	stats := &PolicyStats{
		PolicyID:        policy.ID,
		PolicyName:      policy.Name,
		EvaluationCount: 5000,
		TriggerCount:    150,
		ActionCount:     120,
		TriggerRate:     0.03,
		ActionRate:      0.024,
		AvgLatencyMs:    12.5,
		TotalLatencyMs:  62500,
	}

	return stats, nil
}

// GetCostImpactAnalysis returns cost impact analysis
func (s *GuardrailAnalyticsService) GetCostImpactAnalysis(
	ctx context.Context,
	projectID uuid.UUID,
	startTime, endTime time.Time,
) (*CostImpactAnalysis, error) {
	// Mock implementation
	analysis := &CostImpactAnalysis{
		ProjectID:            projectID,
		StartTime:            startTime,
		EndTime:              endTime,
		TotalEvaluations:     10000,
		TotalLatencyMs:       125000,
		AvgLatencyMs:         12.5,
		EstimatedCostSavings: 45.50, // Estimated savings from blocked requests
		ByPolicy:             make(map[string]*PolicyCostImpact),
	}

	// Mock per-policy cost impact
	analysis.ByPolicy["policy-1"] = &PolicyCostImpact{
		PolicyID:             uuid.New(),
		PolicyName:           "PII Protection",
		EvaluationCount:      5000,
		BlockedCount:         120,
		LatencyImpactMs:      62500,
		AvgLatencyMs:         12.5,
		EstimatedCostSavings: 25.00, // From blocked requests
	}

	analysis.ByPolicy["policy-2"] = &PolicyCostImpact{
		PolicyID:             uuid.New(),
		PolicyName:           "Content Moderation",
		EvaluationCount:      5000,
		BlockedCount:         80,
		LatencyImpactMs:      62500,
		AvgLatencyMs:         12.5,
		EstimatedCostSavings: 20.50,
	}

	return analysis, nil
}

// GetLatencyImpact returns latency impact analysis
func (s *GuardrailAnalyticsService) GetLatencyImpact(
	ctx context.Context,
	projectID uuid.UUID,
	startTime, endTime time.Time,
) (map[string]interface{}, error) {
	// Mock implementation
	impact := map[string]interface{}{
		"totalEvaluations":  10000,
		"totalLatencyMs":    125000,
		"avgLatencyMs":      12.5,
		"medianLatencyMs":   10.0,
		"p95LatencyMs":      25.0,
		"p99LatencyMs":      50.0,
		"byRuleType": map[string]interface{}{
			"pii_detection": map[string]interface{}{
				"avgLatencyMs": 10.0,
				"count":        5000,
			},
			"toxicity": map[string]interface{}{
				"avgLatencyMs": 20.0,
				"count":        3000,
			},
			"secrets_detection": map[string]interface{}{
				"avgLatencyMs": 8.0,
				"count":        2000,
			},
		},
		"impactPercentage": 2.5, // Percentage of total request latency
	}

	return impact, nil
}
