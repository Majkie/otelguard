package service

import (
	"fmt"
	"strings"

	"github.com/otelguard/otelguard/internal/domain"
)

// PricingService handles cost calculation for LLM usage
type PricingService struct {
	modelPricing map[string]domain.Pricing
}

// NewPricingService creates a new pricing service
func NewPricingService() *PricingService {
	return &PricingService{
		modelPricing: getModelPricing(),
	}
}

// EstimateCost calculates the estimated cost for an LLM request
func (s *PricingService) EstimateCost(provider, model string, inputTokens, outputTokens int) (float64, error) {
	pricing, exists := s.modelPricing[getPricingKey(provider, model)]
	if !exists {
		return 0, fmt.Errorf("pricing not found for model %s/%s", provider, model)
	}

	inputCost := float64(inputTokens) / 1000.0 * pricing.InputTokens
	outputCost := float64(outputTokens) / 1000.0 * pricing.OutputTokens

	return inputCost + outputCost, nil
}

// GetPricing returns pricing information for a model
func (s *PricingService) GetPricing(provider, model string) (domain.Pricing, error) {
	pricing, exists := s.modelPricing[getPricingKey(provider, model)]
	if !exists {
		return domain.Pricing{}, fmt.Errorf("pricing not found for model %s/%s", provider, model)
	}
	return pricing, nil
}

// CalculateActualCost calculates actual cost from token usage
func (s *PricingService) CalculateActualCost(provider, model string, usage domain.TokenUsage) (float64, error) {
	return s.EstimateCost(provider, model, usage.PromptTokens, usage.CompletionTokens)
}

// GetAllPricing returns all pricing information
func (s *PricingService) GetAllPricing() map[string]domain.Pricing {
	// Return a copy to prevent external modification
	result := make(map[string]domain.Pricing)
	for k, v := range s.modelPricing {
		result[k] = v
	}
	return result
}

// getPricingKey creates a consistent key for pricing lookup
func getPricingKey(provider, model string) string {
	return fmt.Sprintf("%s/%s", provider, model)
}

// getModelPricing returns hardcoded pricing information
// In production, this could be fetched from an API or database
func getModelPricing() map[string]domain.Pricing {
	return map[string]domain.Pricing{
		// OpenAI pricing (per 1K tokens, as of 2024)
		"openai/gpt-4": {
			InputTokens:  0.03,
			OutputTokens: 0.06,
			Currency:     "USD",
		},
		"openai/gpt-4-turbo": {
			InputTokens:  0.01,
			OutputTokens: 0.03,
			Currency:     "USD",
		},
		"openai/gpt-4-turbo-preview": {
			InputTokens:  0.01,
			OutputTokens: 0.03,
			Currency:     "USD",
		},
		"openai/gpt-3.5-turbo": {
			InputTokens:  0.0015,
			OutputTokens: 0.002,
			Currency:     "USD",
		},
		"openai/gpt-3.5-turbo-16k": {
			InputTokens:  0.003,
			OutputTokens: 0.004,
			Currency:     "USD",
		},

		// Anthropic pricing (per 1K tokens, as of 2024)
		"anthropic/claude-3-opus-20240229": {
			InputTokens:  0.015,
			OutputTokens: 0.075,
			Currency:     "USD",
		},
		"anthropic/claude-3-sonnet-20240229": {
			InputTokens:  0.003,
			OutputTokens: 0.015,
			Currency:     "USD",
		},
		"anthropic/claude-3-haiku-20240307": {
			InputTokens:  0.00025,
			OutputTokens: 0.00125,
			Currency:     "USD",
		},
		"anthropic/claude-2": {
			InputTokens:  0.008,
			OutputTokens: 0.024,
			Currency:     "USD",
		},

		// Google pricing (per 1K tokens, as of 2024)
		"google/gemini-pro": {
			InputTokens:  0.00025,
			OutputTokens: 0.0005,
			Currency:     "USD",
		},
		"google/gemini-pro-vision": {
			InputTokens:  0.00025,
			OutputTokens: 0.0005,
			Currency:     "USD",
		},
		"google/palm-2": {
			InputTokens:  0.0005,
			OutputTokens: 0.0005,
			Currency:     "USD",
		},

		// Ollama (local models - free)
		"ollama/llama2": {
			InputTokens:  0,
			OutputTokens: 0,
			Currency:     "USD",
		},
		"ollama/llama2:13b": {
			InputTokens:  0,
			OutputTokens: 0,
			Currency:     "USD",
		},
		"ollama/codellama": {
			InputTokens:  0,
			OutputTokens: 0,
			Currency:     "USD",
		},
	}
}

// FormatCost formats a cost value with currency
func (s *PricingService) FormatCost(cost float64, currency string) string {
	switch strings.ToUpper(currency) {
	case "USD":
		return fmt.Sprintf("$%.4f", cost)
	case "EUR":
		return fmt.Sprintf("€%.4f", cost)
	case "GBP":
		return fmt.Sprintf("£%.4f", cost)
	default:
		return fmt.Sprintf("%.4f %s", cost, currency)
	}
}

// GetCostBreakdown returns detailed cost breakdown
func (s *PricingService) GetCostBreakdown(provider, model string, inputTokens, outputTokens int) (*CostBreakdown, error) {
	pricing, err := s.GetPricing(provider, model)
	if err != nil {
		return nil, err
	}

	inputCost := float64(inputTokens) / 1000.0 * pricing.InputTokens
	outputCost := float64(outputTokens) / 1000.0 * pricing.OutputTokens
	totalCost := inputCost + outputCost

	return &CostBreakdown{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Currency:     pricing.Currency,
		InputRate:    pricing.InputTokens,
		OutputRate:   pricing.OutputTokens,
	}, nil
}

// CostBreakdown provides detailed cost information
type CostBreakdown struct {
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	InputCost    float64 `json:"inputCost"`
	OutputCost   float64 `json:"outputCost"`
	TotalCost    float64 `json:"totalCost"`
	Currency     string  `json:"currency"`
	InputRate    float64 `json:"inputRate"`  // Cost per 1K input tokens
	OutputRate   float64 `json:"outputRate"` // Cost per 1K output tokens
}
