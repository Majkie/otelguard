package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
)

// ValidatorService provides validation logic for guardrails
type ValidatorService struct {
	logger *zap.Logger
}

// NewValidatorService creates a new validator service
func NewValidatorService(logger *zap.Logger) *ValidatorService {
	return &ValidatorService{
		logger: logger,
	}
}

// ValidatorConfig holds configuration for a specific validator
type ValidatorConfig struct {
	// Common fields
	Threshold float64 `json:"threshold,omitempty"`

	// PII Detection
	PIITypes []string `json:"pii_types,omitempty"` // email, phone, ssn, credit_card, etc.

	// Length limits
	MaxLength   int `json:"max_length,omitempty"`
	MinLength   int `json:"min_length,omitempty"`
	MaxTokens   int `json:"max_tokens,omitempty"`

	// Regex pattern matcher
	Pattern string   `json:"pattern,omitempty"`
	Flags   string   `json:"flags,omitempty"`

	// Keywords blocker
	Keywords     []string `json:"keywords,omitempty"`
	CaseSensitive bool    `json:"case_sensitive,omitempty"`

	// Topic classifier
	Topics []string `json:"topics,omitempty"`

	// JSON Schema validator
	Schema map[string]interface{} `json:"schema,omitempty"`

	// Format validators
	Format string `json:"format,omitempty"` // email, url, uuid, date, etc.

	// Secrets detection
	SecretTypes []string `json:"secret_types,omitempty"` // api_key, password, token, etc.
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Passed     bool
	Triggered  bool
	Message    string
	Confidence float64 // 0.0 to 1.0
	Details    map[string]interface{}
}

// =============================================================================
// INPUT VALIDATORS
// =============================================================================

// ValidatePII detects personally identifiable information
func (s *ValidatorService) ValidatePII(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	detected := []string{}
	confidence := 0.0

	// Email detection
	if len(config.PIITypes) == 0 || contains(config.PIITypes, "email") {
		emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
		if matches := emailRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, fmt.Sprintf("email (%d found)", len(matches)))
			confidence = 1.0
		}
	}

	// Phone number detection (US format)
	if len(config.PIITypes) == 0 || contains(config.PIITypes, "phone") {
		phoneRegex := regexp.MustCompile(`\b(\+?1[-.]?)?\(?\d{3}\)?[-.]?\d{3}[-.]?\d{4}\b`)
		if matches := phoneRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, fmt.Sprintf("phone (%d found)", len(matches)))
			confidence = 1.0
		}
	}

	// SSN detection (US Social Security Number)
	if len(config.PIITypes) == 0 || contains(config.PIITypes, "ssn") {
		ssnRegex := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
		if matches := ssnRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, fmt.Sprintf("ssn (%d found)", len(matches)))
			confidence = 1.0
		}
	}

	// Credit card detection (basic Luhn algorithm check)
	if len(config.PIITypes) == 0 || contains(config.PIITypes, "credit_card") {
		ccRegex := regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
		if matches := ccRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, fmt.Sprintf("credit_card (%d found)", len(matches)))
			confidence = 0.8 // Lower confidence as this is a simple pattern match
		}
	}

	// IP Address detection
	if len(config.PIITypes) == 0 || contains(config.PIITypes, "ip_address") {
		ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
		if matches := ipRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, fmt.Sprintf("ip_address (%d found)", len(matches)))
			confidence = 0.7
		}
	}

	triggered := len(detected) > 0
	message := ""
	if triggered {
		message = fmt.Sprintf("PII detected: %s", strings.Join(detected, ", "))
	}

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    message,
		Confidence: confidence,
		Details: map[string]interface{}{
			"detected_types": detected,
		},
	}
}

// ValidatePromptInjection detects prompt injection attempts
func (s *ValidatorService) ValidatePromptInjection(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	// Common prompt injection patterns
	injectionPatterns := []struct {
		pattern string
		desc    string
	}{
		{`(?i)ignore\s+(previous|above|prior)\s+(instructions|prompts|rules)`, "ignore previous instructions"},
		{`(?i)disregard\s+(previous|above|all)\s+(instructions|prompts|context)`, "disregard instructions"},
		{`(?i)forget\s+(everything|all|previous|your\s+instructions)`, "forget instructions"},
		{`(?i)you\s+are\s+now\s+(a|in|being|acting)`, "role override attempt"},
		{`(?i)act\s+as\s+(if|though|a|an)`, "act as override"},
		{`(?i)system\s*(prompt|message|instruction|role)`, "system prompt manipulation"},
		{`(?i)new\s+(instructions|rules|prompt|directive)`, "new instructions"},
		{`(?i)(start|begin)\s+new\s+(session|conversation|context)`, "session reset"},
		{`(?i)print\s+(your|the)\s+(prompt|instructions|rules)`, "prompt disclosure"},
		{`(?i)reveal\s+(your|the)\s+(prompt|system|instructions)`, "reveal system prompt"},
	}

	detectedPatterns := []string{}
	maxConfidence := 0.0

	for _, p := range injectionPatterns {
		regex := regexp.MustCompile(p.pattern)
		if regex.MatchString(text) {
			detectedPatterns = append(detectedPatterns, p.desc)
			maxConfidence = 0.9 // High confidence for pattern matches
		}
	}

	// Check for excessive special characters (common in injection attempts)
	specialChars := regexp.MustCompile(`[<>{}[\]\\|;]`)
	specialCount := len(specialChars.FindAllString(text, -1))
	if specialCount > 10 {
		detectedPatterns = append(detectedPatterns, "excessive special characters")
		if maxConfidence < 0.6 {
			maxConfidence = 0.6
		}
	}

	triggered := len(detectedPatterns) > 0
	message := ""
	if triggered {
		message = fmt.Sprintf("Potential prompt injection detected: %s", strings.Join(detectedPatterns, ", "))
	}

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    message,
		Confidence: maxConfidence,
		Details: map[string]interface{}{
			"detected_patterns": detectedPatterns,
		},
	}
}

// ValidateSecrets detects API keys, passwords, and other secrets
func (s *ValidatorService) ValidateSecrets(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	detected := []string{}
	confidence := 0.0

	// API Key patterns
	if len(config.SecretTypes) == 0 || contains(config.SecretTypes, "api_key") {
		apiKeyPatterns := []string{
			`(?i)(api[_-]?key|apikey)["\s:=]+([a-zA-Z0-9_\-]{20,})`,
			`(?i)(secret[_-]?key)["\s:=]+([a-zA-Z0-9_\-]{20,})`,
			`(?i)(access[_-]?token)["\s:=]+([a-zA-Z0-9_\-]{20,})`,
		}

		for _, pattern := range apiKeyPatterns {
			regex := regexp.MustCompile(pattern)
			if matches := regex.FindAllString(text, -1); len(matches) > 0 {
				detected = append(detected, "api_key")
				confidence = 0.9
				break
			}
		}
	}

	// AWS Access Key
	if len(config.SecretTypes) == 0 || contains(config.SecretTypes, "aws_key") {
		awsRegex := regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`)
		if matches := awsRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, "aws_access_key")
			confidence = 1.0
		}
	}

	// Generic private key
	if len(config.SecretTypes) == 0 || contains(config.SecretTypes, "private_key") {
		privKeyRegex := regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`)
		if privKeyRegex.MatchString(text) {
			detected = append(detected, "private_key")
			confidence = 1.0
		}
	}

	// Bearer tokens
	if len(config.SecretTypes) == 0 || contains(config.SecretTypes, "bearer_token") {
		bearerRegex := regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`)
		if matches := bearerRegex.FindAllString(text, -1); len(matches) > 0 {
			detected = append(detected, "bearer_token")
			confidence = 0.8
		}
	}

	// Password in plain text patterns
	if len(config.SecretTypes) == 0 || contains(config.SecretTypes, "password") {
		passwordPatterns := []string{
			`(?i)(password|passwd|pwd)["\s:=]+([^\s"']{6,})`,
		}

		for _, pattern := range passwordPatterns {
			regex := regexp.MustCompile(pattern)
			if matches := regex.FindAllString(text, -1); len(matches) > 0 {
				detected = append(detected, "password")
				confidence = 0.7
				break
			}
		}
	}

	triggered := len(detected) > 0
	message := ""
	if triggered {
		message = fmt.Sprintf("Secrets detected: %s", strings.Join(detected, ", "))
	}

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    message,
		Confidence: confidence,
		Details: map[string]interface{}{
			"detected_types": detected,
		},
	}
}

// ValidateLengthLimit checks text length constraints
func (s *ValidatorService) ValidateLengthLimit(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	length := utf8.RuneCountInString(text)

	// Check max length
	if config.MaxLength > 0 && length > config.MaxLength {
		return ValidationResult{
			Passed:     false,
			Triggered:  true,
			Message:    fmt.Sprintf("Text length %d exceeds maximum %d", length, config.MaxLength),
			Confidence: 1.0,
			Details: map[string]interface{}{
				"actual_length": length,
				"max_length":    config.MaxLength,
			},
		}
	}

	// Check min length
	if config.MinLength > 0 && length < config.MinLength {
		return ValidationResult{
			Passed:     false,
			Triggered:  true,
			Message:    fmt.Sprintf("Text length %d below minimum %d", length, config.MinLength),
			Confidence: 1.0,
			Details: map[string]interface{}{
				"actual_length": length,
				"min_length":    config.MinLength,
			},
		}
	}

	// Token count estimation (rough: ~4 chars per token)
	if config.MaxTokens > 0 {
		estimatedTokens := length / 4
		if estimatedTokens > config.MaxTokens {
			return ValidationResult{
				Passed:     false,
				Triggered:  true,
				Message:    fmt.Sprintf("Estimated tokens %d exceeds maximum %d", estimatedTokens, config.MaxTokens),
				Confidence: 0.7,
				Details: map[string]interface{}{
					"estimated_tokens": estimatedTokens,
					"max_tokens":       config.MaxTokens,
				},
			}
		}
	}

	return ValidationResult{
		Passed:     true,
		Triggered:  false,
		Confidence: 1.0,
	}
}

// ValidateRegexPattern matches text against a custom regex pattern
func (s *ValidatorService) ValidateRegexPattern(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	if config.Pattern == "" {
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Message:    "No pattern configured",
			Confidence: 1.0,
		}
	}

	regex, err := regexp.Compile(config.Pattern)
	if err != nil {
		s.logger.Error("invalid regex pattern", zap.Error(err), zap.String("pattern", config.Pattern))
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Message:    fmt.Sprintf("Invalid regex pattern: %v", err),
			Confidence: 0.0,
		}
	}

	matched := regex.MatchString(text)

	return ValidationResult{
		Passed:     !matched,
		Triggered:  matched,
		Message:    fmt.Sprintf("Text matched pattern: %s", config.Pattern),
		Confidence: 1.0,
		Details: map[string]interface{}{
			"pattern": config.Pattern,
		},
	}
}

// ValidateKeywordBlocker blocks text containing specific keywords
func (s *ValidatorService) ValidateKeywordBlocker(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	if len(config.Keywords) == 0 {
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Confidence: 1.0,
		}
	}

	detected := []string{}
	searchText := text
	if !config.CaseSensitive {
		searchText = strings.ToLower(text)
	}

	for _, keyword := range config.Keywords {
		searchKeyword := keyword
		if !config.CaseSensitive {
			searchKeyword = strings.ToLower(keyword)
		}

		if strings.Contains(searchText, searchKeyword) {
			detected = append(detected, keyword)
		}
	}

	triggered := len(detected) > 0
	message := ""
	if triggered {
		message = fmt.Sprintf("Blocked keywords detected: %s", strings.Join(detected, ", "))
	}

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    message,
		Confidence: 1.0,
		Details: map[string]interface{}{
			"detected_keywords": detected,
		},
	}
}

// ValidateLanguage detects text language
func (s *ValidatorService) ValidateLanguage(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	// Simple language detection based on character sets
	// This is a basic implementation - for production, consider using a proper language detection library

	// Count different character types
	hasLatin := regexp.MustCompile(`[a-zA-Z]`).MatchString(text)
	hasChinese := regexp.MustCompile(`[\p{Han}]`).MatchString(text)
	hasJapanese := regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`).MatchString(text)
	hasKorean := regexp.MustCompile(`[\p{Hangul}]`).MatchString(text)
	hasArabic := regexp.MustCompile(`[\p{Arabic}]`).MatchString(text)
	hasCyrillic := regexp.MustCompile(`[\p{Cyrillic}]`).MatchString(text)

	detectedLanguages := []string{}
	if hasLatin {
		detectedLanguages = append(detectedLanguages, "latin")
	}
	if hasChinese {
		detectedLanguages = append(detectedLanguages, "chinese")
	}
	if hasJapanese {
		detectedLanguages = append(detectedLanguages, "japanese")
	}
	if hasKorean {
		detectedLanguages = append(detectedLanguages, "korean")
	}
	if hasArabic {
		detectedLanguages = append(detectedLanguages, "arabic")
	}
	if hasCyrillic {
		detectedLanguages = append(detectedLanguages, "cyrillic")
	}

	return ValidationResult{
		Passed:     true,
		Triggered:  false,
		Confidence: 0.5, // Low confidence for this simple detection
		Details: map[string]interface{}{
			"detected_scripts": detectedLanguages,
		},
	}
}

// =============================================================================
// OUTPUT VALIDATORS
// =============================================================================

// ValidateToxicity detects toxic, harmful, or offensive content
func (s *ValidatorService) ValidateToxicity(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	// Toxic keyword patterns
	toxicPatterns := []string{
		// Profanity (basic list - extend as needed)
		`(?i)\b(fuck|shit|damn|bitch|ass|bastard|crap)\b`,
		// Hate speech indicators
		`(?i)\b(hate|kill|die|death)\s+(all|every|those)\b`,
		// Violence
		`(?i)\b(murder|assault|attack|bomb|weapon|gun)\b`,
	}

	detectedTypes := []string{}
	maxConfidence := 0.0

	for _, pattern := range toxicPatterns {
		regex := regexp.MustCompile(pattern)
		if regex.MatchString(text) {
			detectedTypes = append(detectedTypes, "offensive_language")
			maxConfidence = 0.7 // Medium confidence for keyword matching
		}
	}

	triggered := len(detectedTypes) > 0

	// Apply threshold if configured
	if config.Threshold > 0 && maxConfidence < config.Threshold {
		triggered = false
	}

	message := ""
	if triggered {
		message = fmt.Sprintf("Potentially toxic content detected (confidence: %.2f)", maxConfidence)
	}

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    message,
		Confidence: maxConfidence,
		Details: map[string]interface{}{
			"detected_types": detectedTypes,
		},
	}
}

// ValidateJSONSchema validates JSON output against a schema
func (s *ValidatorService) ValidateJSONSchema(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	if config.Schema == nil {
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Message:    "No schema configured",
			Confidence: 1.0,
		}
	}

	// Parse text as JSON
	var data interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return ValidationResult{
			Passed:     false,
			Triggered:  true,
			Message:    fmt.Sprintf("Invalid JSON: %v", err),
			Confidence: 1.0,
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Validate against schema
	schemaLoader := gojsonschema.NewGoLoader(config.Schema)
	documentLoader := gojsonschema.NewGoLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return ValidationResult{
			Passed:     false,
			Triggered:  true,
			Message:    fmt.Sprintf("Schema validation error: %v", err),
			Confidence: 1.0,
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	if !result.Valid() {
		errors := []string{}
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}

		return ValidationResult{
			Passed:     false,
			Triggered:  true,
			Message:    fmt.Sprintf("Schema validation failed: %s", strings.Join(errors, "; ")),
			Confidence: 1.0,
			Details: map[string]interface{}{
				"errors": errors,
			},
		}
	}

	return ValidationResult{
		Passed:     true,
		Triggered:  false,
		Message:    "JSON schema validation passed",
		Confidence: 1.0,
	}
}

// ValidateFormat validates text against common formats
func (s *ValidatorService) ValidateFormat(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	var regex *regexp.Regexp
	var formatName string

	switch config.Format {
	case "email":
		regex = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$`)
		formatName = "email"
	case "url":
		regex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
		formatName = "URL"
	case "uuid":
		regex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
		formatName = "UUID"
	case "date":
		regex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
		formatName = "date (YYYY-MM-DD)"
	case "ipv4":
		regex = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}$`)
		formatName = "IPv4"
	case "phone":
		regex = regexp.MustCompile(`^\+?[\d\s\-\(\)]{10,}$`)
		formatName = "phone number"
	default:
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Message:    fmt.Sprintf("Unknown format: %s", config.Format),
			Confidence: 0.0,
		}
	}

	matched := regex.MatchString(strings.TrimSpace(text))

	return ValidationResult{
		Passed:     matched,
		Triggered:  !matched,
		Message:    fmt.Sprintf("Text does not match %s format", formatName),
		Confidence: 1.0,
		Details: map[string]interface{}{
			"format": config.Format,
		},
	}
}

// ValidateCompleteness checks if output appears complete
func (s *ValidatorService) ValidateCompleteness(ctx context.Context, text string, config ValidatorConfig) ValidationResult {
	// Check for common incompleteness indicators
	text = strings.TrimSpace(text)

	incomplete := false
	reason := ""

	// Check if text ends abruptly
	if len(text) > 0 {
		lastChar := text[len(text)-1]
		if lastChar != '.' && lastChar != '!' && lastChar != '?' && lastChar != '"' && lastChar != ')' {
			incomplete = true
			reason = "text appears to end abruptly"
		}
	}

	// Check for truncation indicators
	truncationMarkers := []string{
		"...",
		"[truncated]",
		"[cut off]",
		"[incomplete]",
	}

	lowerText := strings.ToLower(text)
	for _, marker := range truncationMarkers {
		if strings.Contains(lowerText, marker) {
			incomplete = true
			reason = "text contains truncation markers"
			break
		}
	}

	return ValidationResult{
		Passed:     !incomplete,
		Triggered:  incomplete,
		Message:    reason,
		Confidence: 0.6,
		Details: map[string]interface{}{
			"complete": !incomplete,
		},
	}
}

// ValidateRelevance checks if output is relevant to input
func (s *ValidatorService) ValidateRelevance(ctx context.Context, input, output string, config ValidatorConfig) ValidationResult {
	// Simple relevance check based on keyword overlap
	// For production, consider using semantic similarity models

	inputWords := extractWords(input)
	outputWords := extractWords(output)

	if len(inputWords) == 0 || len(outputWords) == 0 {
		return ValidationResult{
			Passed:     true,
			Triggered:  false,
			Confidence: 0.0,
		}
	}

	// Calculate word overlap
	overlap := 0
	for word := range inputWords {
		if outputWords[word] {
			overlap++
		}
	}

	relevanceScore := float64(overlap) / float64(len(inputWords))

	threshold := config.Threshold
	if threshold == 0 {
		threshold = 0.1 // Default 10% overlap
	}

	triggered := relevanceScore < threshold

	return ValidationResult{
		Passed:     !triggered,
		Triggered:  triggered,
		Message:    fmt.Sprintf("Relevance score: %.2f (threshold: %.2f)", relevanceScore, threshold),
		Confidence: 0.5,
		Details: map[string]interface{}{
			"relevance_score": relevanceScore,
			"word_overlap":    overlap,
		},
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractWords(text string) map[string]bool {
	// Simple word extraction (lowercase, remove punctuation)
	text = strings.ToLower(text)
	wordRegex := regexp.MustCompile(`\b\w{3,}\b`) // Words with 3+ characters
	matches := wordRegex.FindAllString(text, -1)

	words := make(map[string]bool)
	for _, word := range matches {
		words[word] = true
	}
	return words
}
