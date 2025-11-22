package service

import (
	"context"

	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// PromptService handles prompt business logic
type PromptService struct {
	promptRepo *postgres.PromptRepository
	logger     *zap.Logger
}

// NewPromptService creates a new prompt service
func NewPromptService(promptRepo *postgres.PromptRepository, logger *zap.Logger) *PromptService {
	return &PromptService{
		promptRepo: promptRepo,
		logger:     logger,
	}
}

// ListOptions contains options for listing resources
type ListOptions struct {
	Limit  int
	Offset int
}

// Create creates a new prompt
func (s *PromptService) Create(ctx context.Context, prompt *domain.Prompt) error {
	return s.promptRepo.Create(ctx, prompt)
}

// GetByID retrieves a prompt by ID
func (s *PromptService) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
	return s.promptRepo.GetByID(ctx, id)
}

// List returns prompts for a project
func (s *PromptService) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Prompt, int, error) {
	return s.promptRepo.List(ctx, projectID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// Update updates a prompt
func (s *PromptService) Update(ctx context.Context, prompt *domain.Prompt) error {
	return s.promptRepo.Update(ctx, prompt)
}

// Delete soft-deletes a prompt
func (s *PromptService) Delete(ctx context.Context, id string) error {
	return s.promptRepo.Delete(ctx, id)
}
