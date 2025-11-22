package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// GuardrailService handles guardrail business logic
type GuardrailService struct {
	policyRepo *postgres.GuardrailRepository
	eventRepo  *clickhouse.GuardrailEventRepository
	logger     *zap.Logger
}

// NewGuardrailService creates a new guardrail service
func NewGuardrailService(
	policyRepo *postgres.GuardrailRepository,
	eventRepo *clickhouse.GuardrailEventRepository,
	logger *zap.Logger,
) *GuardrailService {
	return &GuardrailService{
		policyRepo: policyRepo,
		eventRepo:  eventRepo,
		logger:     logger,
	}
}

// EvaluationInput represents the input for guardrail evaluation
type EvaluationInput struct {
	ProjectID uuid.UUID
	TraceID   *uuid.UUID
	SpanID    *uuid.UUID
	PolicyID  *uuid.UUID
	Input     string
	Output    string
	Context   map[string]interface{}
}

// EvaluationResult represents the result of guardrail evaluation
type EvaluationResult struct {
	Passed     bool
	Violations []Violation
	Remediated bool
	Output     string
	LatencyMs  int64
}

// Violation represents a single rule violation
type Violation struct {
	RuleID      uuid.UUID
	RuleType    string
	Message     string
	Action      string
	ActionTaken bool
}

// Evaluate evaluates content against guardrail policies
func (s *GuardrailService) Evaluate(ctx context.Context, input *EvaluationInput) (*EvaluationResult, error) {
	start := time.Now()

	// Get applicable policies
	policies, err := s.policyRepo.GetEnabledPolicies(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	result := &EvaluationResult{
		Passed:     true,
		Violations: []Violation{},
		Output:     input.Output,
	}

	for _, policy := range policies {
		// Get rules for policy
		rules, err := s.policyRepo.GetRules(ctx, policy.ID)
		if err != nil {
			s.logger.Error("failed to get rules", zap.Error(err), zap.String("policy_id", policy.ID.String()))
			continue
		}

		for _, rule := range rules {
			// Evaluate rule
			triggered, message := s.evaluateRule(rule, input.Input, input.Output)

			if triggered {
				result.Passed = false
				violation := Violation{
					RuleID:   rule.ID,
					RuleType: rule.Type,
					Message:  message,
					Action:   rule.Action,
				}

				// Execute action
				if rule.Action == "block" {
					violation.ActionTaken = true
					result.Violations = append(result.Violations, violation)

					// Log guardrail event
					s.logEvent(ctx, input, policy, rule, true, message)
					break // Stop evaluation on block
				}

				result.Violations = append(result.Violations, violation)
				s.logEvent(ctx, input, policy, rule, true, message)
			}
		}
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	return result, nil
}

// evaluateRule evaluates a single rule against input/output
func (s *GuardrailService) evaluateRule(rule *domain.GuardrailRule, input, output string) (bool, string) {
	// TODO: Implement actual rule evaluation logic
	// This would include:
	// - PII detection
	// - Prompt injection detection
	// - Toxicity detection
	// - Custom regex patterns
	// - Length limits
	// etc.

	switch rule.Type {
	case "pii_detection":
		// Placeholder for PII detection
		return false, ""
	case "prompt_injection":
		// Placeholder for prompt injection detection
		return false, ""
	case "toxicity":
		// Placeholder for toxicity detection
		return false, ""
	case "length_limit":
		// Placeholder for length limit check
		return false, ""
	default:
		return false, ""
	}
}

// logEvent logs a guardrail evaluation event to ClickHouse
func (s *GuardrailService) logEvent(
	ctx context.Context,
	input *EvaluationInput,
	policy *domain.GuardrailPolicy,
	rule *domain.GuardrailRule,
	triggered bool,
	message string,
) {
	event := &domain.GuardrailEvent{
		ID:              uuid.New(),
		ProjectID:       input.ProjectID,
		TraceID:         input.TraceID,
		SpanID:          input.SpanID,
		PolicyID:        policy.ID,
		RuleID:          rule.ID,
		RuleType:        rule.Type,
		Triggered:       triggered,
		Action:          rule.Action,
		ActionTaken:     triggered && rule.Action == "block",
		InputText:       input.Input,
		DetectionResult: message,
		CreatedAt:       time.Now(),
	}

	if input.Output != "" {
		event.OutputText = &input.Output
	}

	if err := s.eventRepo.Insert(ctx, event); err != nil {
		s.logger.Error("failed to log guardrail event", zap.Error(err))
	}
}
