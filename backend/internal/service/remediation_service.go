package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// RemediationService handles auto-remediation of guardrail violations
type RemediationService struct {
	logger *zap.Logger
}

// NewRemediationService creates a new remediation service
func NewRemediationService(logger *zap.Logger) *RemediationService {
	return &RemediationService{
		logger: logger,
	}
}

// RemediationConfig holds configuration for remediation actions
type RemediationConfig struct {
	// Common
	Action string `json:"action"` // block, sanitize, retry, fallback, alert, transform

	// Block action
	BlockResponse string `json:"block_response,omitempty"`

	// Sanitize action
	SanitizeTypes []string `json:"sanitize_types,omitempty"` // pii, secrets, etc.
	RedactText    string   `json:"redact_text,omitempty"`    // Text to replace with (default: [REDACTED])

	// Retry action
	RetryCount       int               `json:"retry_count,omitempty"`
	RetryDelay       int               `json:"retry_delay,omitempty"` // milliseconds
	ModifyParameters map[string]string `json:"modify_parameters,omitempty"`

	// Fallback action
	FallbackModel    string `json:"fallback_model,omitempty"`
	FallbackResponse string `json:"fallback_response,omitempty"`

	// Alert action
	AlertChannel  string   `json:"alert_channel,omitempty"` // email, slack, webhook
	AlertRecipients []string `json:"alert_recipients,omitempty"`

	// Transform action
	TransformType string                 `json:"transform_type,omitempty"` // truncate, format, extract
	TransformConfig map[string]interface{} `json:"transform_config,omitempty"`
}

// RemediationResult represents the result of a remediation action
type RemediationResult struct {
	Success      bool
	Action       string
	ModifiedText string
	Message      string
	Details      map[string]interface{}
}

// ExecuteRemediation executes the appropriate remediation action
func (s *RemediationService) ExecuteRemediation(
	ctx context.Context,
	text string,
	ruleType string,
	config RemediationConfig,
) (*RemediationResult, error) {
	switch config.Action {
	case "block":
		return s.executeBlock(ctx, config)
	case "sanitize":
		return s.executeSanitize(ctx, text, ruleType, config)
	case "retry":
		return s.executeRetry(ctx, text, config)
	case "fallback":
		return s.executeFallback(ctx, text, config)
	case "alert":
		return s.executeAlert(ctx, text, ruleType, config)
	case "transform":
		return s.executeTransform(ctx, text, config)
	default:
		return &RemediationResult{
			Success: false,
			Action:  config.Action,
			Message: fmt.Sprintf("Unknown remediation action: %s", config.Action),
		}, nil
	}
}

// executeBlock blocks the request and returns a safe response
func (s *RemediationService) executeBlock(ctx context.Context, config RemediationConfig) (*RemediationResult, error) {
	response := config.BlockResponse
	if response == "" {
		response = "I cannot process this request as it violates our content policy."
	}

	return &RemediationResult{
		Success:      true,
		Action:       "block",
		ModifiedText: response,
		Message:      "Request blocked and safe response returned",
		Details: map[string]interface{}{
			"blocked": true,
		},
	}, nil
}

// executeSanitize sanitizes the text by redacting sensitive information
func (s *RemediationService) executeSanitize(
	ctx context.Context,
	text string,
	ruleType string,
	config RemediationConfig,
) (*RemediationResult, error) {
	redactText := config.RedactText
	if redactText == "" {
		redactText = "[REDACTED]"
	}

	sanitized := text
	redactionCount := 0

	// Determine what to sanitize based on rule type or config
	sanitizeTypes := config.SanitizeTypes
	if len(sanitizeTypes) == 0 {
		// Infer from rule type
		switch ruleType {
		case "pii_detection":
			sanitizeTypes = []string{"email", "phone", "ssn", "credit_card", "ip_address"}
		case "secrets_detection":
			sanitizeTypes = []string{"api_key", "password", "token"}
		}
	}

	// PII redaction
	if contains(sanitizeTypes, "email") {
		emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
		matches := emailRegex.FindAllString(sanitized, -1)
		sanitized = emailRegex.ReplaceAllString(sanitized, redactText)
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "phone") {
		phoneRegex := regexp.MustCompile(`\b(\+?1[-.]?)?\(?\d{3}\)?[-.]?\d{3}[-.]?\d{4}\b`)
		matches := phoneRegex.FindAllString(sanitized, -1)
		sanitized = phoneRegex.ReplaceAllString(sanitized, redactText)
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "ssn") {
		ssnRegex := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
		matches := ssnRegex.FindAllString(sanitized, -1)
		sanitized = ssnRegex.ReplaceAllString(sanitized, redactText)
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "credit_card") {
		ccRegex := regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
		matches := ccRegex.FindAllString(sanitized, -1)
		sanitized = ccRegex.ReplaceAllString(sanitized, redactText)
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "ip_address") {
		ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
		matches := ipRegex.FindAllString(sanitized, -1)
		sanitized = ipRegex.ReplaceAllString(sanitized, redactText)
		redactionCount += len(matches)
	}

	// Secrets redaction
	if contains(sanitizeTypes, "api_key") {
		apiKeyRegex := regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret[_-]?key|access[_-]?token)["\s:=]+([a-zA-Z0-9_\-]{20,})`)
		matches := apiKeyRegex.FindAllString(sanitized, -1)
		sanitized = apiKeyRegex.ReplaceAllString(sanitized, fmt.Sprintf("$1 %s", redactText))
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "password") {
		passwordRegex := regexp.MustCompile(`(?i)(password|passwd|pwd)["\s:=]+([^\s"']{6,})`)
		matches := passwordRegex.FindAllString(sanitized, -1)
		sanitized = passwordRegex.ReplaceAllString(sanitized, fmt.Sprintf("$1 %s", redactText))
		redactionCount += len(matches)
	}

	if contains(sanitizeTypes, "token") || contains(sanitizeTypes, "bearer_token") {
		bearerRegex := regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`)
		matches := bearerRegex.FindAllString(sanitized, -1)
		sanitized = bearerRegex.ReplaceAllString(sanitized, fmt.Sprintf("bearer %s", redactText))
		redactionCount += len(matches)
	}

	return &RemediationResult{
		Success:      true,
		Action:       "sanitize",
		ModifiedText: sanitized,
		Message:      fmt.Sprintf("Sanitized text with %d redactions", redactionCount),
		Details: map[string]interface{}{
			"redaction_count": redactionCount,
			"sanitize_types":  sanitizeTypes,
		},
	}, nil
}

// executeRetry prepares the request for retry with modified parameters
func (s *RemediationService) executeRetry(
	ctx context.Context,
	text string,
	config RemediationConfig,
) (*RemediationResult, error) {
	retryCount := config.RetryCount
	if retryCount == 0 {
		retryCount = 1
	}

	modifications := []string{}
	for key, value := range config.ModifyParameters {
		modifications = append(modifications, fmt.Sprintf("%s=%s", key, value))
	}

	return &RemediationResult{
		Success:      true,
		Action:       "retry",
		ModifiedText: text,
		Message:      fmt.Sprintf("Retry requested (%d attempts) with modifications: %s", retryCount, strings.Join(modifications, ", ")),
		Details: map[string]interface{}{
			"retry_count":        retryCount,
			"retry_delay":        config.RetryDelay,
			"modify_parameters":  config.ModifyParameters,
		},
	}, nil
}

// executeFallback executes fallback strategy
func (s *RemediationService) executeFallback(
	ctx context.Context,
	text string,
	config RemediationConfig,
) (*RemediationResult, error) {
	response := config.FallbackResponse
	if response == "" {
		response = "I apologize, but I cannot provide a complete response at this time. Please try rephrasing your request."
	}

	details := map[string]interface{}{
		"fallback": true,
	}

	if config.FallbackModel != "" {
		details["fallback_model"] = config.FallbackModel
	}

	return &RemediationResult{
		Success:      true,
		Action:       "fallback",
		ModifiedText: response,
		Message:      "Fallback response used",
		Details:      details,
	}, nil
}

// executeAlert sends an alert but allows processing to continue
func (s *RemediationService) executeAlert(
	ctx context.Context,
	text string,
	ruleType string,
	config RemediationConfig,
) (*RemediationResult, error) {
	// In production, this would send actual alerts via configured channels
	// For now, we just log the alert

	alertDetails := map[string]interface{}{
		"rule_type": ruleType,
		"text_preview": truncateText(text, 100),
	}

	if config.AlertChannel != "" {
		alertDetails["channel"] = config.AlertChannel
	}
	if len(config.AlertRecipients) > 0 {
		alertDetails["recipients"] = config.AlertRecipients
	}

	s.logger.Warn("guardrail alert triggered",
		zap.String("rule_type", ruleType),
		zap.String("channel", config.AlertChannel),
		zap.Any("details", alertDetails),
	)

	return &RemediationResult{
		Success:      true,
		Action:       "alert",
		ModifiedText: text, // Text unchanged
		Message:      fmt.Sprintf("Alert sent via %s", config.AlertChannel),
		Details:      alertDetails,
	}, nil
}

// executeTransform transforms the output based on configuration
func (s *RemediationService) executeTransform(
	ctx context.Context,
	text string,
	config RemediationConfig,
) (*RemediationResult, error) {
	transformed := text
	transformType := config.TransformType

	switch transformType {
	case "truncate":
		maxLength := 500 // default
		if val, ok := config.TransformConfig["max_length"]; ok {
			if length, ok := val.(float64); ok {
				maxLength = int(length)
			}
		}

		if len(transformed) > maxLength {
			transformed = transformed[:maxLength] + "..."
		}

		return &RemediationResult{
			Success:      true,
			Action:       "transform",
			ModifiedText: transformed,
			Message:      fmt.Sprintf("Text truncated to %d characters", maxLength),
			Details: map[string]interface{}{
				"transform_type": "truncate",
				"max_length":     maxLength,
				"original_length": len(text),
			},
		}, nil

	case "format":
		// Format text (e.g., ensure proper JSON formatting)
		format := "text"
		if val, ok := config.TransformConfig["format"]; ok {
			if f, ok := val.(string); ok {
				format = f
			}
		}

		if format == "json" {
			// Try to parse and re-format JSON
			var data interface{}
			if err := json.Unmarshal([]byte(text), &data); err == nil {
				if formatted, err := json.MarshalIndent(data, "", "  "); err == nil {
					transformed = string(formatted)
				}
			}
		}

		return &RemediationResult{
			Success:      true,
			Action:       "transform",
			ModifiedText: transformed,
			Message:      fmt.Sprintf("Text formatted as %s", format),
			Details: map[string]interface{}{
				"transform_type": "format",
				"format":         format,
			},
		}, nil

	case "extract":
		// Extract specific parts (e.g., extract JSON from markdown)
		pattern := `\{[^}]+\}`
		if val, ok := config.TransformConfig["pattern"]; ok {
			if p, ok := val.(string); ok {
				pattern = p
			}
		}

		regex, err := regexp.Compile(pattern)
		if err == nil {
			if match := regex.FindString(text); match != "" {
				transformed = match
			}
		}

		return &RemediationResult{
			Success:      true,
			Action:       "transform",
			ModifiedText: transformed,
			Message:      "Text extracted using pattern",
			Details: map[string]interface{}{
				"transform_type": "extract",
				"pattern":        pattern,
			},
		}, nil

	case "lowercase":
		transformed = strings.ToLower(text)
		return &RemediationResult{
			Success:      true,
			Action:       "transform",
			ModifiedText: transformed,
			Message:      "Text converted to lowercase",
			Details: map[string]interface{}{
				"transform_type": "lowercase",
			},
		}, nil

	case "uppercase":
		transformed = strings.ToUpper(text)
		return &RemediationResult{
			Success:      true,
			Action:       "transform",
			ModifiedText: transformed,
			Message:      "Text converted to uppercase",
			Details: map[string]interface{}{
				"transform_type": "uppercase",
			},
		}, nil

	default:
		return &RemediationResult{
			Success:      false,
			Action:       "transform",
			ModifiedText: text,
			Message:      fmt.Sprintf("Unknown transform type: %s", transformType),
		}, nil
	}
}

// ExecuteRemediationChain executes multiple remediation actions in sequence
func (s *RemediationService) ExecuteRemediationChain(
	ctx context.Context,
	text string,
	ruleType string,
	configs []RemediationConfig,
) (*RemediationResult, error) {
	currentText := text
	combinedResult := &RemediationResult{
		Success: true,
		Action:  "chain",
		Details: map[string]interface{}{
			"actions": []string{},
		},
	}

	for i, config := range configs {
		result, err := s.ExecuteRemediation(ctx, currentText, ruleType, config)
		if err != nil {
			s.logger.Error("remediation action failed",
				zap.Error(err),
				zap.Int("action_index", i),
				zap.String("action", config.Action),
			)
			combinedResult.Success = false
			combinedResult.Message = fmt.Sprintf("Action %d failed: %v", i, err)
			return combinedResult, err
		}

		if !result.Success {
			combinedResult.Success = false
			combinedResult.Message = fmt.Sprintf("Action %d unsuccessful: %s", i, result.Message)
			return combinedResult, nil
		}

		// Update current text for next action
		currentText = result.ModifiedText

		// Track actions
		if actions, ok := combinedResult.Details["actions"].([]string); ok {
			combinedResult.Details["actions"] = append(actions, result.Action)
		}
	}

	combinedResult.ModifiedText = currentText
	combinedResult.Message = fmt.Sprintf("Remediation chain completed (%d actions)", len(configs))

	return combinedResult, nil
}

// Helper functions

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
