package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/otelguard/otelguard/internal/domain"
	"go.uber.org/zap"
)

// EnrichmentConfig contains configuration for trace enrichment
type EnrichmentConfig struct {
	EnablePIIDetection    bool              `envconfig:"ENRICHMENT_PII" default:"false"`
	EnableCostCalculation bool              `envconfig:"ENRICHMENT_COST" default:"true"`
	EnableModelDetection  bool              `envconfig:"ENRICHMENT_MODEL" default:"true"`
	CustomTags            map[string]string `envconfig:"ENRICHMENT_TAGS"`
}

// DefaultEnrichmentConfig returns default enrichment configuration
func DefaultEnrichmentConfig() *EnrichmentConfig {
	return &EnrichmentConfig{
		EnablePIIDetection:    false,
		EnableCostCalculation: true,
		EnableModelDetection:  true,
	}
}

// TraceEnrichmentPipeline handles trace enrichment operations
type TraceEnrichmentPipeline struct {
	config *EnrichmentConfig
	logger *zap.Logger
}

// NewTraceEnrichmentPipeline creates a new enrichment pipeline
func NewTraceEnrichmentPipeline(config *EnrichmentConfig, logger *zap.Logger) *TraceEnrichmentPipeline {
	if config == nil {
		config = DefaultEnrichmentConfig()
	}
	return &TraceEnrichmentPipeline{
		config: config,
		logger: logger,
	}
}

// EnrichTrace enriches a trace with additional data
func (p *TraceEnrichmentPipeline) EnrichTrace(ctx context.Context, trace *domain.Trace) error {
	// Calculate latency if not set
	if trace.LatencyMs == 0 && !trace.EndTime.IsZero() && !trace.StartTime.IsZero() {
		trace.LatencyMs = uint32(trace.EndTime.Sub(trace.StartTime).Milliseconds())
	}

	// Set end time if not provided
	if trace.EndTime.IsZero() && trace.LatencyMs > 0 {
		trace.EndTime = trace.StartTime.Add(time.Duration(trace.LatencyMs) * time.Millisecond)
	}

	// Calculate total tokens if not set
	if trace.TotalTokens == 0 && (trace.PromptTokens > 0 || trace.CompletionTokens > 0) {
		trace.TotalTokens = trace.PromptTokens + trace.CompletionTokens
	}

	// Model detection and normalization
	if p.config.EnableModelDetection {
		trace.Model = normalizeModelName(trace.Model)
	}

	// Cost calculation
	if p.config.EnableCostCalculation && trace.Cost == 0 && trace.TotalTokens > 0 {
		trace.Cost = calculateCost(trace.Model, trace.PromptTokens, trace.CompletionTokens)
	}

	// Set default status if not provided
	if trace.Status == "" {
		trace.Status = domain.StatusSuccess
	}

	// PII detection (adds tags)
	if p.config.EnablePIIDetection {
		if hasPII(trace.Input) || hasPII(trace.Output) {
			trace.Tags = appendUnique(trace.Tags, "pii_detected")
		}
	}

	// Add custom tags
	for key, value := range p.config.CustomTags {
		trace.Tags = appendUnique(trace.Tags, key+":"+value)
	}

	// Enrich metadata
	trace.Metadata = enrichMetadata(trace.Metadata)

	return nil
}

// EnrichSpan enriches a span with additional data
func (p *TraceEnrichmentPipeline) EnrichSpan(ctx context.Context, span *domain.Span) error {
	// Calculate latency if not set
	if span.LatencyMs == 0 && !span.EndTime.IsZero() && !span.StartTime.IsZero() {
		span.LatencyMs = uint32(span.EndTime.Sub(span.StartTime).Milliseconds())
	}

	// Set end time if not provided
	if span.EndTime.IsZero() && span.LatencyMs > 0 {
		span.EndTime = span.StartTime.Add(time.Duration(span.LatencyMs) * time.Millisecond)
	}

	// Detect span type if not set
	if span.Type == "" {
		span.Type = detectSpanType(span.Name, span.Metadata)
	}

	// Model normalization
	if p.config.EnableModelDetection && span.Model != nil {
		normalized := normalizeModelName(*span.Model)
		span.Model = &normalized
	}

	// Cost calculation
	if p.config.EnableCostCalculation && span.Cost == 0 && span.Tokens > 0 && span.Model != nil {
		// Assume roughly 50/50 split for prompt/completion if not known
		span.Cost = calculateCost(*span.Model, span.Tokens/2, span.Tokens/2)
	}

	// Set default status if not provided
	if span.Status == "" {
		span.Status = domain.StatusSuccess
	}

	return nil
}

// normalizeModelName normalizes model names to a standard format
func normalizeModelName(model string) string {
	if model == "" {
		return model
	}

	model = strings.ToLower(model)

	// Common model name mappings
	modelMappings := map[string]string{
		"gpt-4-turbo":         "gpt-4-turbo",
		"gpt-4-turbo-preview": "gpt-4-turbo",
		"gpt-4-0125-preview":  "gpt-4-turbo",
		"gpt-4-1106-preview":  "gpt-4-turbo",
		"gpt-4o":              "gpt-4o",
		"gpt-4o-mini":         "gpt-4o-mini",
		"gpt-4":               "gpt-4",
		"gpt-3.5-turbo":       "gpt-3.5-turbo",
		"gpt-3.5-turbo-0125":  "gpt-3.5-turbo",
		"claude-3-opus":       "claude-3-opus",
		"claude-3-sonnet":     "claude-3-sonnet",
		"claude-3-haiku":      "claude-3-haiku",
		"claude-3-5-sonnet":   "claude-3.5-sonnet",
		"claude-3.5-sonnet":   "claude-3.5-sonnet",
		"claude-2":            "claude-2",
		"claude-instant":      "claude-instant",
		"text-embedding-ada":  "text-embedding-ada-002",
		"text-embedding-3":    "text-embedding-3-small",
	}

	for pattern, normalized := range modelMappings {
		if strings.Contains(model, pattern) {
			return normalized
		}
	}

	return model
}

// Model pricing per 1K tokens (input, output)
var modelPricing = map[string][2]float64{
	"gpt-4o":                 {0.005, 0.015},
	"gpt-4o-mini":            {0.00015, 0.0006},
	"gpt-4-turbo":            {0.01, 0.03},
	"gpt-4":                  {0.03, 0.06},
	"gpt-3.5-turbo":          {0.0005, 0.0015},
	"claude-3-opus":          {0.015, 0.075},
	"claude-3-sonnet":        {0.003, 0.015},
	"claude-3.5-sonnet":      {0.003, 0.015},
	"claude-3-haiku":         {0.00025, 0.00125},
	"claude-2":               {0.008, 0.024},
	"claude-instant":         {0.0008, 0.0024},
	"text-embedding-ada-002": {0.0001, 0},
	"text-embedding-3-small": {0.00002, 0},
	"text-embedding-3-large": {0.00013, 0},
}

// calculateCost calculates the cost based on model and tokens
func calculateCost(model string, promptTokens, completionTokens uint32) float64 {
	pricing, ok := modelPricing[normalizeModelName(model)]
	if !ok {
		return 0
	}

	inputCost := float64(promptTokens) / 1000 * pricing[0]
	outputCost := float64(completionTokens) / 1000 * pricing[1]

	return inputCost + outputCost
}

// hasPII checks if text contains potential PII
func hasPII(text string) bool {
	if text == "" {
		return false
	}

	patterns := []string{
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`, // Email
		`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,                      // Phone
		`\b\d{3}[-]?\d{2}[-]?\d{4}\b`,                        // SSN
		`\b(?:\d{4}[-\s]?){3}\d{4}\b`,                        // Credit card
		`\b[A-Z][a-z]+\s+[A-Z][a-z]+\b`,                      // Names (simple)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(text) {
			return true
		}
	}

	return false
}

// appendUnique appends a value to a slice if it doesn't already exist
func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

// detectSpanType detects the span type based on name and metadata
func detectSpanType(name string, metadata string) string {
	nameLower := strings.ToLower(name)

	// Check name patterns
	switch {
	case strings.Contains(nameLower, "llm") || strings.Contains(nameLower, "chat") ||
		strings.Contains(nameLower, "completion") || strings.Contains(nameLower, "generate"):
		return domain.SpanTypeLLM
	case strings.Contains(nameLower, "embed"):
		return domain.SpanTypeEmbedding
	case strings.Contains(nameLower, "retriev") || strings.Contains(nameLower, "search") ||
		strings.Contains(nameLower, "vector"):
		return domain.SpanTypeRetrieval
	case strings.Contains(nameLower, "tool") || strings.Contains(nameLower, "function"):
		return domain.SpanTypeTool
	case strings.Contains(nameLower, "agent"):
		return domain.SpanTypeAgent
	}

	// Check metadata for hints
	if metadata != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(metadata), &m); err == nil {
			if _, ok := m["model"]; ok {
				return domain.SpanTypeLLM
			}
			if _, ok := m["tool_name"]; ok {
				return domain.SpanTypeTool
			}
		}
	}

	return domain.SpanTypeCustom
}

// enrichMetadata enriches the metadata JSON
func enrichMetadata(metadata string) string {
	if metadata == "" {
		return "{}"
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &m); err != nil {
		return metadata
	}

	// Add enrichment timestamp
	m["_enriched_at"] = time.Now().UTC().Format(time.RFC3339)

	// Re-serialize
	enriched, err := json.Marshal(m)
	if err != nil {
		return metadata
	}

	return string(enriched)
}
