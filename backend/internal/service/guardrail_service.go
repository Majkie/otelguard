package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// GuardrailService handles guardrail business logic
type GuardrailService struct {
	policyRepo         *postgres.GuardrailRepository
	eventRepo          *clickhouse.GuardrailEventRepository
	validatorService   *ValidatorService
	remediationService *RemediationService
	cache              *EvaluationCache
	logger             *zap.Logger
}

// NewGuardrailService creates a new guardrail service
func NewGuardrailService(
	policyRepo *postgres.GuardrailRepository,
	eventRepo *clickhouse.GuardrailEventRepository,
	validatorService *ValidatorService,
	remediationService *RemediationService,
	logger *zap.Logger,
) *GuardrailService {
	// Create cache with 5 minute TTL and max 10000 entries
	cache := NewEvaluationCache(CacheConfig{
		TTL:             5 * time.Minute,
		MaxSize:         10000,
		CleanupInterval: 1 * time.Minute,
	}, logger)

	return &GuardrailService{
		policyRepo:         policyRepo,
		eventRepo:          eventRepo,
		validatorService:   validatorService,
		remediationService: remediationService,
		cache:              cache,
		logger:             logger,
	}
}

// EvaluationInput represents the input for guardrail evaluation
type EvaluationInput struct {
	ProjectID   uuid.UUID
	TraceID     *uuid.UUID
	SpanID      *uuid.UUID
	PolicyID    *uuid.UUID
	Input       string
	Output      string
	Context     map[string]interface{}
	Model       string   // LLM model being used
	Environment string   // Environment (production, staging, etc.)
	Tags        []string // Tags for matching
	UserID      string   // User ID for matching
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

// PolicyTriggers represents the trigger conditions for a guardrail policy
type PolicyTriggers struct {
	Models       []string          `json:"models,omitempty"`       // Model patterns (supports wildcards like "gpt-4*")
	Environments []string          `json:"environments,omitempty"` // Environments to match
	Tags         []string          `json:"tags,omitempty"`         // Required tags
	UserIDs      []string          `json:"userIds,omitempty"`      // Specific user IDs
	Conditions   map[string]string `json:"conditions,omitempty"`   // Custom conditions
}

// Evaluate evaluates content against guardrail policies
func (s *GuardrailService) Evaluate(ctx context.Context, input *EvaluationInput) (*EvaluationResult, error) {
	// Check cache first (if not testing mode)
	if cached, found := s.cache.Get(ctx, input); found {
		s.logger.Debug("returning cached evaluation result")
		return cached, nil
	}

	start := time.Now()

	// Get all enabled policies
	allPolicies, err := s.policyRepo.GetEnabledPolicies(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Filter policies based on triggers (policy matching logic)
	matchedPolicies := s.filterPoliciesByTriggers(allPolicies, input)

	// Sort policies by priority (higher priority first)
	sort.Slice(matchedPolicies, func(i, j int) bool {
		return matchedPolicies[i].Priority > matchedPolicies[j].Priority
	})

	s.logger.Debug("evaluating guardrails",
		zap.Int("total_policies", len(allPolicies)),
		zap.Int("matched_policies", len(matchedPolicies)),
		zap.String("model", input.Model),
		zap.String("environment", input.Environment),
	)

	result := &EvaluationResult{
		Passed:     true,
		Violations: []Violation{},
		Output:     input.Output,
	}

	// Evaluate matched policies in priority order
	for _, policy := range matchedPolicies {
		// Get rules for policy
		rules, err := s.policyRepo.GetRules(ctx, policy.ID)
		if err != nil {
			s.logger.Error("failed to get rules", zap.Error(err), zap.String("policy_id", policy.ID.String()))
			continue
		}

		// Sort rules by order_index
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].OrderIndex < rules[j].OrderIndex
		})

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

				// Parse action configuration
				var remediationConfig RemediationConfig
				if len(rule.ActionConfig) > 0 {
					if err := json.Unmarshal(rule.ActionConfig, &remediationConfig); err != nil {
						s.logger.Warn("failed to parse action config",
							zap.Error(err),
							zap.String("rule_id", rule.ID.String()),
						)
					}
				}
				remediationConfig.Action = rule.Action

				// Execute remediation action
				remediationResult, err := s.remediationService.ExecuteRemediation(
					ctx,
					result.Output,
					rule.Type,
					remediationConfig,
				)

				if err != nil {
					s.logger.Error("remediation failed",
						zap.Error(err),
						zap.String("action", rule.Action),
					)
				} else if remediationResult.Success {
					violation.ActionTaken = true
					result.Output = remediationResult.ModifiedText
					result.Remediated = true
				}

				result.Violations = append(result.Violations, violation)
				s.logEvent(ctx, input, policy, rule, true, message)

				// Block action stops all further evaluation
				if rule.Action == "block" {
					result.LatencyMs = time.Since(start).Milliseconds()
					return result, nil
				}
			}
		}
	}

	result.LatencyMs = time.Since(start).Milliseconds()

	// Cache the result
	s.cache.Set(ctx, input, result)

	return result, nil
}

// filterPoliciesByTriggers filters policies based on trigger conditions
func (s *GuardrailService) filterPoliciesByTriggers(policies []*domain.GuardrailPolicy, input *EvaluationInput) []*domain.GuardrailPolicy {
	var matched []*domain.GuardrailPolicy

	for _, policy := range policies {
		// If specific policy requested, only match that one
		if input.PolicyID != nil && policy.ID != *input.PolicyID {
			continue
		}

		// Parse triggers
		var triggers PolicyTriggers
		if len(policy.Triggers) > 0 {
			if err := json.Unmarshal(policy.Triggers, &triggers); err != nil {
				s.logger.Warn("failed to parse policy triggers",
					zap.Error(err),
					zap.String("policy_id", policy.ID.String()),
				)
				// If triggers are invalid, include the policy (fail open)
				matched = append(matched, policy)
				continue
			}
		}

		// If no triggers defined, match all
		if len(triggers.Models) == 0 && len(triggers.Environments) == 0 &&
			len(triggers.Tags) == 0 && len(triggers.UserIDs) == 0 {
			matched = append(matched, policy)
			continue
		}

		// Check model matching (supports wildcards)
		if len(triggers.Models) > 0 && input.Model != "" {
			modelMatched := false
			for _, pattern := range triggers.Models {
				if matchPattern(pattern, input.Model) {
					modelMatched = true
					break
				}
			}
			if !modelMatched {
				continue
			}
		}

		// Check environment matching
		if len(triggers.Environments) > 0 && input.Environment != "" {
			envMatched := false
			for _, env := range triggers.Environments {
				if strings.EqualFold(env, input.Environment) {
					envMatched = true
					break
				}
			}
			if !envMatched {
				continue
			}
		}

		// Check tag matching (any tag must match)
		if len(triggers.Tags) > 0 && len(input.Tags) > 0 {
			tagMatched := false
			for _, requiredTag := range triggers.Tags {
				for _, inputTag := range input.Tags {
					if requiredTag == inputTag {
						tagMatched = true
						break
					}
				}
				if tagMatched {
					break
				}
			}
			if !tagMatched {
				continue
			}
		}

		// Check user ID matching
		if len(triggers.UserIDs) > 0 && input.UserID != "" {
			userMatched := false
			for _, userID := range triggers.UserIDs {
				if userID == input.UserID {
					userMatched = true
					break
				}
			}
			if !userMatched {
				continue
			}
		}

		// Policy matched all conditions
		matched = append(matched, policy)
	}

	return matched
}

// matchPattern matches a string against a pattern with wildcard support
// Supports * as wildcard (e.g., "gpt-4*" matches "gpt-4", "gpt-4-turbo", etc.)
func matchPattern(pattern, str string) bool {
	if pattern == "*" {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return strings.EqualFold(pattern, str)
	}

	// Simple wildcard matching
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return true
	}

	// Check prefix
	if len(parts[0]) > 0 {
		if !strings.HasPrefix(strings.ToLower(str), strings.ToLower(parts[0])) {
			return false
		}
		str = str[len(parts[0]):]
	}

	// Check suffix
	if len(parts) > 1 && len(parts[len(parts)-1]) > 0 {
		suffix := parts[len(parts)-1]
		if !strings.HasSuffix(strings.ToLower(str), strings.ToLower(suffix)) {
			return false
		}
	}

	return true
}

// evaluateRule evaluates a single rule against input/output
func (s *GuardrailService) evaluateRule(rule *domain.GuardrailRule, input, output string) (bool, string) {
	ctx := context.Background()

	// Parse rule configuration
	var config ValidatorConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &config); err != nil {
			s.logger.Warn("failed to parse rule config",
				zap.Error(err),
				zap.String("rule_id", rule.ID.String()),
			)
			// Continue with empty config
		}
	}

	var result ValidationResult

	switch rule.Type {
	// Input validators
	case "pii_detection":
		result = s.validatorService.ValidatePII(ctx, input, config)
	case "prompt_injection":
		result = s.validatorService.ValidatePromptInjection(ctx, input, config)
	case "secrets_detection":
		result = s.validatorService.ValidateSecrets(ctx, input, config)
	case "length_limit":
		result = s.validatorService.ValidateLengthLimit(ctx, input, config)
	case "regex_pattern":
		result = s.validatorService.ValidateRegexPattern(ctx, input, config)
	case "keyword_blocker":
		result = s.validatorService.ValidateKeywordBlocker(ctx, input, config)
	case "language_detection":
		result = s.validatorService.ValidateLanguage(ctx, input, config)

	// Output validators
	case "toxicity":
		result = s.validatorService.ValidateToxicity(ctx, output, config)
	case "json_schema":
		result = s.validatorService.ValidateJSONSchema(ctx, output, config)
	case "format_validator":
		result = s.validatorService.ValidateFormat(ctx, output, config)
	case "completeness":
		result = s.validatorService.ValidateCompleteness(ctx, output, config)
	case "relevance":
		result = s.validatorService.ValidateRelevance(ctx, input, output, config)

	default:
		s.logger.Warn("unknown rule type", zap.String("type", rule.Type))
		return false, ""
	}

	return result.Triggered, result.Message
}

// List returns guardrail policies for a project
func (s *GuardrailService) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.GuardrailPolicy, int, error) {
	return s.policyRepo.List(ctx, projectID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// GetByID retrieves a policy by ID
func (s *GuardrailService) GetByID(ctx context.Context, id string) (*domain.GuardrailPolicy, error) {
	return s.policyRepo.GetByID(ctx, id)
}

// Create creates a new guardrail policy
func (s *GuardrailService) Create(ctx context.Context, policy *domain.GuardrailPolicy) error {
	return s.policyRepo.Create(ctx, policy)
}

// Update updates a guardrail policy
func (s *GuardrailService) Update(ctx context.Context, policy *domain.GuardrailPolicy) error {
	return s.policyRepo.Update(ctx, policy)
}

// Delete deletes a guardrail policy
func (s *GuardrailService) Delete(ctx context.Context, id string) error {
	return s.policyRepo.Delete(ctx, id)
}

// GetRules returns all rules for a policy
func (s *GuardrailService) GetRules(ctx context.Context, policyID uuid.UUID) ([]*domain.GuardrailRule, error) {
	return s.policyRepo.GetRules(ctx, policyID)
}

// AddRule adds a rule to a policy
func (s *GuardrailService) AddRule(ctx context.Context, rule *domain.GuardrailRule) error {
	return s.policyRepo.AddRule(ctx, rule)
}

// DeleteRule deletes a rule
func (s *GuardrailService) DeleteRule(ctx context.Context, id string) error {
	return s.policyRepo.DeleteRule(ctx, id)
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

// CreateVersion creates a new version snapshot of a policy
func (s *GuardrailService) CreateVersion(ctx context.Context, policyID string, changeNotes string, createdBy uuid.UUID) (*domain.GuardrailPolicyVersion, error) {
	// Get current policy
	policy, err := s.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return nil, err
	}

	// Get current rules
	rules, err := s.policyRepo.GetRules(ctx, policy.ID)
	if err != nil {
		return nil, err
	}

	// Serialize rules to JSON
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize rules: %w", err)
	}

	// Get next version number
	nextVersion, err := s.policyRepo.GetNextVersionNumber(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	// Create version snapshot
	version := &domain.GuardrailPolicyVersion{
		ID:          uuid.New(),
		PolicyID:    policy.ID,
		Version:     nextVersion,
		Name:        policy.Name,
		Description: policy.Description,
		Enabled:     policy.Enabled,
		Priority:    policy.Priority,
		Triggers:    policy.Triggers,
		Rules:       rulesJSON,
		ChangeNotes: changeNotes,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := s.policyRepo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// Update current_version on policy
	if err := s.policyRepo.UpdateCurrentVersion(ctx, policyID, nextVersion); err != nil {
		s.logger.Warn("failed to update current version", zap.Error(err))
	}

	return version, nil
}

// GetVersion retrieves a specific version of a policy
func (s *GuardrailService) GetVersion(ctx context.Context, policyID string, version int) (*domain.GuardrailPolicyVersion, error) {
	return s.policyRepo.GetVersion(ctx, policyID, version)
}

// GetLatestVersion retrieves the latest version of a policy
func (s *GuardrailService) GetLatestVersion(ctx context.Context, policyID string) (*domain.GuardrailPolicyVersion, error) {
	return s.policyRepo.GetLatestVersion(ctx, policyID)
}

// ListVersions retrieves all versions of a policy
func (s *GuardrailService) ListVersions(ctx context.Context, policyID string) ([]*domain.GuardrailPolicyVersion, error) {
	return s.policyRepo.ListVersions(ctx, policyID)
}

// RestoreVersion restores a policy to a previous version
func (s *GuardrailService) RestoreVersion(ctx context.Context, policyID string, version int, createdBy uuid.UUID) error {
	// Get the version to restore
	oldVersion, err := s.policyRepo.GetVersion(ctx, policyID, version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Get current policy
	policy, err := s.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return err
	}

	// Update policy with old version data
	policy.Name = oldVersion.Name
	policy.Description = oldVersion.Description
	policy.Enabled = oldVersion.Enabled
	policy.Priority = oldVersion.Priority
	policy.Triggers = oldVersion.Triggers
	policy.UpdatedAt = time.Now()

	if err := s.policyRepo.Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	// Restore rules
	var rules []*domain.GuardrailRule
	if err := json.Unmarshal(oldVersion.Rules, &rules); err != nil {
		return fmt.Errorf("failed to parse rules: %w", err)
	}

	// Delete existing rules
	currentRules, err := s.policyRepo.GetRules(ctx, policy.ID)
	if err != nil {
		return fmt.Errorf("failed to get current rules: %w", err)
	}

	for _, rule := range currentRules {
		if err := s.policyRepo.DeleteRule(ctx, rule.ID.String()); err != nil {
			s.logger.Warn("failed to delete rule", zap.Error(err))
		}
	}

	// Add restored rules
	for _, rule := range rules {
		rule.ID = uuid.New() // Generate new IDs
		rule.CreatedAt = time.Now()
		if err := s.policyRepo.AddRule(ctx, rule); err != nil {
			s.logger.Error("failed to add restored rule", zap.Error(err))
		}
	}

	// Create a new version snapshot with restore note
	changeNotes := fmt.Sprintf("Restored from version %d", version)
	_, err = s.CreateVersion(ctx, policyID, changeNotes, createdBy)
	return err
}

// GetCacheStats returns cache statistics
func (s *GuardrailService) GetCacheStats() *CacheStats {
	return s.cache.GetStats()
}

// ClearCache clears the evaluation cache
func (s *GuardrailService) ClearCache() {
	s.cache.InvalidateAll()
}

// InvalidateCache invalidates cache entries for a specific project or policy
func (s *GuardrailService) InvalidateCache(ctx context.Context, projectID string, policyID *string) int {
	return s.cache.Invalidate(projectID, policyID)
}
