package service

import (
	"context"
	"fmt"

	"github.com/otelguard/otelguard/internal/domain"
	"go.uber.org/zap"

	"github.com/anthropics/anthropic-sdk-go"
	// "github.com/google/generative-ai-go/genai"
	// "github.com/ollama/ollama/api"
	openai "github.com/sashabaranov/go-openai"
)

// LLMService defines the interface for LLM operations
type LLMService interface {
	// ListAvailableModels returns all available models across providers
	ListAvailableModels() ([]domain.LLMModel, error)

	// ExecutePrompt executes a prompt against an LLM provider
	ExecutePrompt(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error)

	// StreamPrompt executes a prompt with streaming response
	StreamPrompt(ctx context.Context, req domain.LLMRequest) (<-chan domain.LLMResponse, <-chan error)

	// CountTokens counts tokens for a given text and model
	CountTokens(text string, model string) (int, error)

	// EstimateCost estimates the cost for a given request
	EstimateCost(req domain.LLMRequest, estimatedOutputTokens int) (float64, error)

	// SaveExecution saves a playground execution record
	SaveExecution(ctx context.Context, exec *domain.PlaygroundExecution) error
}

// LLMServiceImpl implements the LLMService interface
type LLMServiceImpl struct {
	logger    *zap.Logger
	tokenizer *TokenizerService
	pricing   *PricingService

	// Provider clients
	openaiClient    *openai.Client
	anthropicClient anthropic.Client
	// googleClient    *genai.Client
	// ollamaClient    *api.Client

	// Available models cache
	models []domain.LLMModel
}

// NewLLMService creates a new LLM service instance
func NewLLMService(logger *zap.Logger, tokenizer *TokenizerService, pricing *PricingService) *LLMServiceImpl {
	service := &LLMServiceImpl{
		logger:    logger,
		tokenizer: tokenizer,
		pricing:   pricing,
		models:    getAvailableModels(),
	}

	// Initialize clients if API keys are available
	// These will be populated from environment variables
	service.initializeClients()

	return service
}

func (s *LLMServiceImpl) initializeClients() {
	// TODO: Initialize clients from environment config
	// For now, we'll create clients without API keys - they'll fail gracefully when used
	s.openaiClient = openai.NewClient("")
	s.anthropicClient = anthropic.NewClient()
	// Google and Ollama clients need different initialization
}

func (s *LLMServiceImpl) ListAvailableModels() ([]domain.LLMModel, error) {
	return s.models, nil
}

func (s *LLMServiceImpl) ExecutePrompt(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error) {
	switch req.Provider {
	case domain.ProviderOpenAI:
		return s.executeOpenAI(ctx, req)
	case domain.ProviderAnthropic:
		return s.executeAnthropic(ctx, req)
	case domain.ProviderGoogle:
		return s.executeGoogle(ctx, req)
	case domain.ProviderOllama:
		return s.executeOllama(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

func (s *LLMServiceImpl) StreamPrompt(ctx context.Context, req domain.LLMRequest) (<-chan domain.LLMResponse, <-chan error) {
	respCh := make(chan domain.LLMResponse)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		switch req.Provider {
		case domain.ProviderOpenAI:
			s.streamOpenAI(ctx, req, respCh, errCh)
		case domain.ProviderAnthropic:
			s.streamAnthropic(ctx, req, respCh, errCh)
		case domain.ProviderGoogle:
			s.streamGoogle(ctx, req, respCh, errCh)
		case domain.ProviderOllama:
			s.streamOllama(ctx, req, respCh, errCh)
		default:
			errCh <- fmt.Errorf("unsupported provider: %s", req.Provider)
		}
	}()

	return respCh, errCh
}

func (s *LLMServiceImpl) CountTokens(text string, model string) (int, error) {
	return s.tokenizer.CountTokens(text, model)
}

func (s *LLMServiceImpl) EstimateCost(req domain.LLMRequest, estimatedOutputTokens int) (float64, error) {
	return s.pricing.EstimateCost(req.Provider, req.Model, 0, estimatedOutputTokens)
}

func (s *LLMServiceImpl) SaveExecution(ctx context.Context, exec *domain.PlaygroundExecution) error {
	// TODO: Implement database storage for execution history
	// For now, just log it
	s.logger.Info("Playground execution saved",
		zap.String("id", exec.ID.String()),
		zap.String("provider", exec.Request.Provider),
		zap.String("model", exec.Request.Model),
		zap.Float64("cost", exec.Cost),
		zap.Duration("executionTime", exec.ExecutionTime),
	)
	return nil
}

func (s *LLMServiceImpl) findModel(provider, modelID string) *domain.LLMModel {
	for _, model := range s.models {
		if model.Provider == provider && model.ModelID == modelID {
			return &model
		}
	}
	return nil
}

// OpenAI implementation
func (s *LLMServiceImpl) executeOpenAI(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: req.Prompt},
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	temperature := float32(req.Temperature)
	if temperature == 0 {
		temperature = 0.7
	}

	resp, err := s.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return &domain.LLMResponse{
		Text: resp.Choices[0].Message.Content,
		Usage: domain.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		FinishReason: string(resp.Choices[0].FinishReason),
	}, nil
}

func (s *LLMServiceImpl) streamOpenAI(ctx context.Context, req domain.LLMRequest, respCh chan<- domain.LLMResponse, errCh chan<- error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: req.Prompt},
	}

	stream, err := s.openaiClient.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
		Stream:      true,
	})
	if err != nil {
		errCh <- err
		return
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "stream closed" || err.Error() == "EOF" {
				break
			}
			errCh <- err
			return
		}

		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta.Content
			if delta != "" {
				respCh <- domain.LLMResponse{
					Text: delta,
					Usage: domain.TokenUsage{
						PromptTokens:     response.Usage.PromptTokens,
						CompletionTokens: response.Usage.CompletionTokens,
						TotalTokens:      response.Usage.TotalTokens,
					},
				}
			}
		}
	}
}

// Anthropic implementation
func (s *LLMServiceImpl) executeAnthropic(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 1000
	}

	message, err := s.anthropicClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
		Temperature: anthropic.Float(req.Temperature),
	})
	if err != nil {
		return nil, err
	}

	if len(message.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	text := ""
	for _, block := range message.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	return &domain.LLMResponse{
		Text: text,
		Usage: domain.TokenUsage{
			PromptTokens:     int(message.Usage.InputTokens),
			CompletionTokens: int(message.Usage.OutputTokens),
			TotalTokens:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
		FinishReason: string(message.StopReason),
	}, nil
}

func (s *LLMServiceImpl) streamAnthropic(ctx context.Context, req domain.LLMRequest, respCh chan<- domain.LLMResponse, errCh chan<- error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 1000
	}

	stream := s.anthropicClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
		Temperature: anthropic.Float(req.Temperature),
	})

	for stream.Next() {
		event := stream.Current()
		if event.Delta.Text != "" {
			respCh <- domain.LLMResponse{
				Text: event.Delta.Text,
			}
		}
	}

	if err := stream.Err(); err != nil {
		errCh <- err
	}
}

// Google implementation
func (s *LLMServiceImpl) executeGoogle(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error) {
	// TODO: Implement Google Generative AI
	return nil, fmt.Errorf("Google provider not yet implemented")
}

func (s *LLMServiceImpl) streamGoogle(ctx context.Context, req domain.LLMRequest, respCh chan<- domain.LLMResponse, errCh chan<- error) {
	// TODO: Implement Google Generative AI streaming
	errCh <- fmt.Errorf("Google provider not yet implemented")
}

// Ollama implementation
func (s *LLMServiceImpl) executeOllama(ctx context.Context, req domain.LLMRequest) (*domain.LLMResponse, error) {
	// TODO: Implement Ollama
	return nil, fmt.Errorf("Ollama provider not yet implemented")
}

func (s *LLMServiceImpl) streamOllama(ctx context.Context, req domain.LLMRequest, respCh chan<- domain.LLMResponse, errCh chan<- error) {
	// TODO: Implement Ollama streaming
	errCh <- fmt.Errorf("Ollama provider not yet implemented")
}

// getAvailableModels returns a hardcoded list of available models
// In production, this could be fetched from APIs or configured dynamically
func getAvailableModels() []domain.LLMModel {
	return []domain.LLMModel{
		// OpenAI models
		{
			ID:          "gpt-4",
			Name:        "GPT-4",
			Provider:    domain.ProviderOpenAI,
			ModelID:     "gpt-4",
			ContextSize: 8192,
			Pricing: domain.Pricing{
				InputTokens:  0.03,
				OutputTokens: 0.06,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},
		{
			ID:          "gpt-4-turbo",
			Name:        "GPT-4 Turbo",
			Provider:    domain.ProviderOpenAI,
			ModelID:     "gpt-4-turbo-preview",
			ContextSize: 128000,
			Pricing: domain.Pricing{
				InputTokens:  0.01,
				OutputTokens: 0.03,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},
		{
			ID:          "gpt-3.5-turbo",
			Name:        "GPT-3.5 Turbo",
			Provider:    domain.ProviderOpenAI,
			ModelID:     "gpt-3.5-turbo",
			ContextSize: 16384,
			Pricing: domain.Pricing{
				InputTokens:  0.0015,
				OutputTokens: 0.002,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},

		// Anthropic models
		{
			ID:          "claude-3-opus",
			Name:        "Claude 3 Opus",
			Provider:    domain.ProviderAnthropic,
			ModelID:     "claude-3-opus-20240229",
			ContextSize: 200000,
			Pricing: domain.Pricing{
				InputTokens:  0.015,
				OutputTokens: 0.075,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},
		{
			ID:          "claude-3-sonnet",
			Name:        "Claude 3 Sonnet",
			Provider:    domain.ProviderAnthropic,
			ModelID:     "claude-3-sonnet-20240229",
			ContextSize: 200000,
			Pricing: domain.Pricing{
				InputTokens:  0.003,
				OutputTokens: 0.015,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},
		{
			ID:          "claude-3-haiku",
			Name:        "Claude 3 Haiku",
			Provider:    domain.ProviderAnthropic,
			ModelID:     "claude-3-haiku-20240307",
			ContextSize: 200000,
			Pricing: domain.Pricing{
				InputTokens:  0.00025,
				OutputTokens: 0.00125,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},

		// Google models
		{
			ID:          "gemini-pro",
			Name:        "Gemini Pro",
			Provider:    domain.ProviderGoogle,
			ModelID:     "gemini-pro",
			ContextSize: 30720,
			Pricing: domain.Pricing{
				InputTokens:  0.00025,
				OutputTokens: 0.0005,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},

		// Ollama models (local)
		{
			ID:          "llama2",
			Name:        "Llama 2 7B",
			Provider:    domain.ProviderOllama,
			ModelID:     "llama2",
			ContextSize: 4096,
			Pricing: domain.Pricing{
				InputTokens:  0,
				OutputTokens: 0,
				Currency:     "USD",
			},
			Capabilities: []string{"chat", "completion"},
		},
	}
}
