package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// FeedbackService handles user feedback business logic
type FeedbackService struct {
	feedbackRepo       *postgres.FeedbackRepository
	feedbackMappingSvc *FeedbackScoreMappingService
	projectRepo        *postgres.ProjectRepository
	userRepo           *postgres.UserRepository
	logger             *zap.Logger
}

// NewFeedbackService creates a new feedback service
func NewFeedbackService(
	feedbackRepo *postgres.FeedbackRepository,
	feedbackMappingSvc *FeedbackScoreMappingService,
	projectRepo *postgres.ProjectRepository,
	userRepo *postgres.UserRepository,
	logger *zap.Logger,
) *FeedbackService {
	return &FeedbackService{
		feedbackRepo:       feedbackRepo,
		feedbackMappingSvc: feedbackMappingSvc,
		projectRepo:        projectRepo,
		userRepo:           userRepo,
		logger:             logger,
	}
}

// CreateFeedback creates new user feedback
func (s *FeedbackService) CreateFeedback(ctx context.Context, req *domain.UserFeedbackCreate) (*domain.UserFeedback, error) {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, domain.ErrNotFound
	}

	// Verify user exists if provided
	if req.UserID != nil {
		user, err := s.userRepo.GetByID(ctx, req.UserID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		if user == nil {
			return nil, domain.ErrNotFound
		}
	}

	// Validate item type and ID combination
	if err := s.validateFeedbackItem(ctx, req); err != nil {
		return nil, err
	}

	// Create feedback entity
	now := time.Now()
	feedback := &domain.UserFeedback{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		TraceID:   req.TraceID,
		SpanID:    req.SpanID,
		ItemType:  req.ItemType,
		ItemID:    req.ItemID,
		Comment:   req.Comment,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.ThumbsUp != nil {
		feedback.ThumbsUp = sql.NullBool{Bool: *req.ThumbsUp, Valid: true}
	}

	if req.Rating != nil {
		feedback.Rating = sql.NullInt32{Int32: int32(*req.Rating), Valid: true}
	}

	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		feedback.Metadata = metadataJSON
	}

	// Create in repository
	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		s.logger.Error("failed to create feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	s.logger.Info("feedback created",
		zap.String("id", feedback.ID.String()),
		zap.String("project_id", req.ProjectID.String()),
		zap.String("item_type", req.ItemType),
		zap.String("item_id", req.ItemID),
	)

	// Process feedback for score mapping (async, don't fail the request if this fails)
	go func() {
		ctx := context.Background() // Use background context for async processing
		if err := s.feedbackMappingSvc.ProcessFeedback(ctx, feedback); err != nil {
			s.logger.Warn("failed to process feedback for score mapping",
				zap.String("feedback_id", feedback.ID.String()),
				zap.Error(err),
			)
		}
	}()

	return feedback, nil
}

// GetFeedback retrieves feedback by ID
func (s *FeedbackService) GetFeedback(ctx context.Context, id string) (*domain.UserFeedback, error) {
	feedback, err := s.feedbackRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	return feedback, nil
}

// UpdateFeedback updates existing feedback
func (s *FeedbackService) UpdateFeedback(ctx context.Context, id string, req *domain.UserFeedbackUpdate) (*domain.UserFeedback, error) {
	// Get existing feedback
	feedback, err := s.feedbackRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	// Update fields
	if req.ThumbsUp != nil {
		feedback.ThumbsUp = sql.NullBool{Bool: *req.ThumbsUp, Valid: true}
	}

	if req.Rating != nil {
		feedback.Rating = sql.NullInt32{Int32: int32(*req.Rating), Valid: true}
	}

	if req.Comment != nil {
		feedback.Comment = *req.Comment
	}

	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(*req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		feedback.Metadata = metadataJSON
	}

	feedback.UpdatedAt = time.Now()

	// Update in repository
	if err := s.feedbackRepo.Update(ctx, feedback); err != nil {
		s.logger.Error("failed to update feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to update feedback: %w", err)
	}

	s.logger.Info("feedback updated", zap.String("id", id))

	return feedback, nil
}

// DeleteFeedback deletes feedback
func (s *FeedbackService) DeleteFeedback(ctx context.Context, id string) error {
	if err := s.feedbackRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete feedback", zap.Error(err))
		return fmt.Errorf("failed to delete feedback: %w", err)
	}

	s.logger.Info("feedback deleted", zap.String("id", id))
	return nil
}

// ListFeedback retrieves feedback with filtering
func (s *FeedbackService) ListFeedback(ctx context.Context, filter domain.FeedbackFilter) ([]*domain.UserFeedback, int64, error) {
	feedback, total, err := s.feedbackRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list feedback: %w", err)
	}

	return feedback, total, nil
}

// GetFeedbackAnalytics retrieves aggregated analytics
func (s *FeedbackService) GetFeedbackAnalytics(ctx context.Context, projectID string, itemType string, startDate, endDate time.Time) (*domain.FeedbackAnalytics, error) {
	analytics, err := s.feedbackRepo.GetAnalytics(ctx, projectID, itemType, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback analytics: %w", err)
	}

	return analytics, nil
}

// GetFeedbackTrends retrieves feedback trends over time
func (s *FeedbackService) GetFeedbackTrends(ctx context.Context, projectID string, itemType string, startDate, endDate time.Time, interval string) ([]*domain.FeedbackTrend, error) {
	trends, err := s.feedbackRepo.GetTrends(ctx, projectID, itemType, startDate, endDate, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback trends: %w", err)
	}

	return trends, nil
}

// validateFeedbackItem validates that the item being referenced exists
func (s *FeedbackService) validateFeedbackItem(ctx context.Context, req *domain.UserFeedbackCreate) error {
	switch req.ItemType {
	case "trace":
		// For traces, we could validate against ClickHouse, but for now just ensure TraceID is provided
		if req.TraceID == nil || *req.TraceID == "" {
			return fmt.Errorf("trace_id is required for trace feedback")
		}
	case "session":
		// For sessions, we could validate against ClickHouse, but for now just ensure SessionID is provided
		if req.SessionID == nil || *req.SessionID == "" {
			return fmt.Errorf("session_id is required for session feedback")
		}
	case "span":
		// For spans, we could validate against ClickHouse, but for now just ensure SpanID is provided
		if req.SpanID == nil || *req.SpanID == "" {
			return fmt.Errorf("span_id is required for span feedback")
		}
	case "prompt":
		// For prompts, we could validate against PostgreSQL, but for now just ensure ItemID is provided
		if req.ItemID == "" {
			return fmt.Errorf("item_id is required for prompt feedback")
		}
	default:
		return fmt.Errorf("invalid item_type: %s", req.ItemType)
	}

	return nil
}
