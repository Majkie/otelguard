package service

import (
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

// TokenizerService handles token counting for different LLM models
type TokenizerService struct {
	encodings map[string]*tiktoken.Tiktoken
}

// NewTokenizerService creates a new tokenizer service
func NewTokenizerService() *TokenizerService {
	return &TokenizerService{
		encodings: make(map[string]*tiktoken.Tiktoken),
	}
}

// CountTokens counts the number of tokens in the given text for the specified model
func (s *TokenizerService) CountTokens(text string, model string) (int, error) {
	// Use tiktoken for OpenAI models
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "text-") {
		return s.countTokensTikToken(text, model)
	}

	// For Anthropic models, use approximate counting
	if strings.HasPrefix(model, "claude-") {
		return s.countTokensApproximate(text), nil
	}

	// For Google models, use approximate counting
	if strings.HasPrefix(model, "gemini-") || strings.HasPrefix(model, "palm-") {
		return s.countTokensApproximate(text), nil
	}

	// For local models (Ollama), use approximate counting
	return s.countTokensApproximate(text), nil
}

// countTokensTikToken uses tiktoken to count tokens for OpenAI models
func (s *TokenizerService) countTokensTikToken(text string, model string) (int, error) {
	var encoding string

	// Map model names to tiktoken encodings
	switch {
	case strings.HasPrefix(model, "gpt-4"):
		encoding = "cl100k_base"
	case strings.HasPrefix(model, "gpt-3.5"):
		encoding = "cl100k_base"
	case strings.HasPrefix(model, "text-davinci"):
		encoding = "p50k_base"
	case strings.HasPrefix(model, "text-curie"):
		encoding = "p50k_base"
	case strings.HasPrefix(model, "text-babbage"):
		encoding = "p50k_base"
	case strings.HasPrefix(model, "text-ada"):
		encoding = "p50k_base"
	case strings.HasPrefix(model, "code"):
		encoding = "p50k_base"
	default:
		// Default to cl100k_base for newer models
		encoding = "cl100k_base"
	}

	tkm, err := s.getEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding for %s: %w", encoding, err)
	}

	return len(tkm.Encode(text, nil, nil)), nil
}

// countTokensApproximate provides a rough approximation for models without tiktoken support
func (s *TokenizerService) countTokensApproximate(text string) int {
	// Rough approximation: ~4 characters per token for most models
	// This is not accurate but provides a reasonable estimate
	return int(float64(len(strings.Fields(text)))*1.3) + len(text)/4
}

// getEncoding gets or creates a tiktoken encoding
func (s *TokenizerService) getEncoding(encodingName string) (*tiktoken.Tiktoken, error) {
	if tkm, exists := s.encodings[encodingName]; exists {
		return tkm, nil
	}

	tkm, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, err
	}

	s.encodings[encodingName] = tkm
	return tkm, nil
}

// EstimateOutputTokens estimates the number of output tokens based on input tokens and model
func (s *TokenizerService) EstimateOutputTokens(inputTokens int, model string) int {
	// Rough estimation: output tokens are typically 1/3 to 1/2 of input tokens
	// This is highly variable depending on the task
	switch {
	case strings.HasPrefix(model, "gpt-4"):
		return inputTokens / 3
	case strings.HasPrefix(model, "gpt-3.5"):
		return inputTokens / 4
	case strings.HasPrefix(model, "claude-"):
		return inputTokens / 3
	default:
		return inputTokens / 4
	}
}

// GetContextSize returns the context size for a given model
func (s *TokenizerService) GetContextSize(model string) int {
	switch {
	case strings.HasPrefix(model, "gpt-4-turbo"):
		return 128000
	case strings.HasPrefix(model, "gpt-4"):
		return 8192
	case strings.HasPrefix(model, "gpt-3.5-turbo"):
		return 16384
	case strings.HasPrefix(model, "claude-3-opus"):
		return 200000
	case strings.HasPrefix(model, "claude-3-sonnet"):
		return 200000
	case strings.HasPrefix(model, "claude-3-haiku"):
		return 200000
	case strings.HasPrefix(model, "gemini-pro"):
		return 30720
	case strings.HasPrefix(model, "llama2"):
		return 4096
	default:
		return 4096 // Default context size
	}
}

// IsWithinContext checks if the given token count is within the model's context size
func (s *TokenizerService) IsWithinContext(tokenCount int, model string) bool {
	contextSize := s.GetContextSize(model)
	return tokenCount <= contextSize
}
