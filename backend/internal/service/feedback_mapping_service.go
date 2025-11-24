package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// FeedbackScoreMappingService handles feedback to score mapping logic
type FeedbackScoreMappingService struct {
	feedbackMappingRepo *postgres.FeedbackScoreMappingRepository
	traceService        *TraceService
	projectRepo         *postgres.ProjectRepository
	logger              *zap.Logger
}

// NewFeedbackScoreMappingService creates a new feedback score mapping service
func NewFeedbackScoreMappingService(
	feedbackMappingRepo *postgres.FeedbackScoreMappingRepository,
	traceService *TraceService,
	projectRepo *postgres.ProjectRepository,
	logger *zap.Logger,
) *FeedbackScoreMappingService {
	return &FeedbackScoreMappingService{
		feedbackMappingRepo: feedbackMappingRepo,
		traceService:        traceService,
		projectRepo:         projectRepo,
		logger:              logger,
	}
}

// CreateMapping creates a new feedback score mapping
func (s *FeedbackScoreMappingService) CreateMapping(ctx context.Context, req *domain.FeedbackScoreMappingCreate) (*domain.FeedbackScoreMapping, error) {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, domain.ErrNotFound
	}

	// Create mapping entity
	now := time.Now()
	mapping := &domain.FeedbackScoreMapping{
		ID:          uuid.New(),
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		ItemType:    req.ItemType,
		Enabled:     req.Enabled != nil && *req.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		mapping.Config = configJSON
	}

	// Create in repository
	if err := s.feedbackMappingRepo.Create(ctx, mapping); err != nil {
		s.logger.Error("failed to create feedback score mapping", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback score mapping: %w", err)
	}

	s.logger.Info("feedback score mapping created",
		zap.String("id", mapping.ID.String()),
		zap.String("project_id", req.ProjectID.String()),
		zap.String("name", req.Name),
	)

	return mapping, nil
}

// GetMapping retrieves a feedback score mapping by ID
func (s *FeedbackScoreMappingService) GetMapping(ctx context.Context, id string) (*domain.FeedbackScoreMapping, error) {
	mapping, err := s.feedbackMappingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback score mapping: %w", err)
	}

	return mapping, nil
}

// UpdateMapping updates an existing feedback score mapping
func (s *FeedbackScoreMappingService) UpdateMapping(ctx context.Context, id string, req *domain.FeedbackScoreMappingUpdate) (*domain.FeedbackScoreMapping, error) {
	// Get existing mapping
	mapping, err := s.feedbackMappingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback score mapping: %w", err)
	}

	// Update fields
	if req.Name != nil {
		mapping.Name = *req.Name
	}
	if req.Description != nil {
		mapping.Description = *req.Description
	}
	if req.Enabled != nil {
		mapping.Enabled = *req.Enabled
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		mapping.Config = configJSON
	}

	mapping.UpdatedAt = time.Now()

	// Update in repository
	if err := s.feedbackMappingRepo.Update(ctx, mapping); err != nil {
		s.logger.Error("failed to update feedback score mapping", zap.Error(err))
		return nil, fmt.Errorf("failed to update feedback score mapping: %w", err)
	}

	s.logger.Info("feedback score mapping updated", zap.String("id", id))

	return mapping, nil
}

// DeleteMapping deletes a feedback score mapping
func (s *FeedbackScoreMappingService) DeleteMapping(ctx context.Context, id string) error {
	if err := s.feedbackMappingRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete feedback score mapping", zap.Error(err))
		return fmt.Errorf("failed to delete feedback score mapping: %w", err)
	}

	s.logger.Info("feedback score mapping deleted", zap.String("id", id))
	return nil
}

// ListMappings retrieves feedback score mappings with filtering
func (s *FeedbackScoreMappingService) ListMappings(ctx context.Context, projectID string, itemType string, enabled *bool) ([]*domain.FeedbackScoreMapping, error) {
	mappings, err := s.feedbackMappingRepo.List(ctx, projectID, itemType, enabled)
	if err != nil {
		return nil, fmt.Errorf("failed to list feedback score mappings: %w", err)
	}

	return mappings, nil
}

// ProcessFeedback converts feedback into scores based on mappings
func (s *FeedbackScoreMappingService) ProcessFeedback(ctx context.Context, feedback *domain.UserFeedback) error {
	// Get enabled mappings for this project and item type
	mappings, err := s.feedbackMappingRepo.GetEnabledForItemType(ctx, feedback.ProjectID.String(), feedback.ItemType)
	if err != nil {
		return fmt.Errorf("failed to get feedback mappings: %w", err)
	}

	if len(mappings) == 0 {
		// No mappings configured, skip processing
		return nil
	}

	// Process each mapping
	for _, mapping := range mappings {
		var config domain.FeedbackScoreMappingConfig
		if err := json.Unmarshal(mapping.Config, &config); err != nil {
			s.logger.Warn("failed to unmarshal mapping config",
				zap.String("mapping_id", mapping.ID.String()),
				zap.Error(err),
			)
			continue
		}

		scores := s.generateScoresFromFeedback(feedback, &config)
		for _, score := range scores {
			// Create the score
			scoreReq := &domain.Score{
				ID:        uuid.New(),
				ProjectID: feedback.ProjectID,
				Name:      score.Name,
				Value:     score.Value,
				DataType:  score.DataType,
				Source:    "user_feedback",
				ConfigID:  &score.ScoreConfigID,
				Comment:   &score.Comment,
				CreatedAt: time.Now(),
			}

			// Convert TraceID from string pointer to UUID if available
			if feedback.TraceID != nil {
				if traceUUID, err := uuid.Parse(*feedback.TraceID); err == nil {
					scoreReq.TraceID = traceUUID
				}
			}

			// Convert SpanID from string pointer to UUID pointer if available
			if feedback.SpanID != nil {
				if spanUUID, err := uuid.Parse(*feedback.SpanID); err == nil {
					scoreReq.SpanID = &spanUUID
				}
			}

			// Add session ID if available
			if feedback.SessionID != nil {
				// For now, we might need to extend the Score entity to include session_id
				// This would require a database migration
			}

			if err := s.traceService.SubmitScore(ctx, scoreReq); err != nil {
				s.logger.Error("failed to create score from feedback",
					zap.String("feedback_id", feedback.ID.String()),
					zap.String("mapping_id", mapping.ID.String()),
					zap.Error(err),
				)
				continue
			}

			s.logger.Info("score created from feedback",
				zap.String("feedback_id", feedback.ID.String()),
				zap.String("mapping_id", mapping.ID.String()),
				zap.String("score_name", score.Name),
				zap.Float64("score_value", score.Value),
			)
		}
	}

	return nil
}

// generateScoresFromFeedback generates scores from feedback based on mapping config
func (s *FeedbackScoreMappingService) generateScoresFromFeedback(
	feedback *domain.UserFeedback,
	config *domain.FeedbackScoreMappingConfig,
) []*scoreFromMapping {
	var scores []*scoreFromMapping

	// Process thumbs up/down
	if feedback.ThumbsUp.Valid && config.ThumbsUpScore != nil {
		if feedback.ThumbsUp.Bool && config.ThumbsUpScore != nil {
			scores = append(scores, &scoreFromMapping{
				Name:          "thumbs_up_feedback",
				Value:         config.ThumbsUpScore.Value,
				DataType:      "numeric",
				ScoreConfigID: config.ThumbsUpScore.ScoreConfigID,
				Comment:       config.ThumbsUpScore.Comment,
			})
		} else if !feedback.ThumbsUp.Bool && config.ThumbsDownScore != nil {
			scores = append(scores, &scoreFromMapping{
				Name:          "thumbs_down_feedback",
				Value:         config.ThumbsDownScore.Value,
				DataType:      "numeric",
				ScoreConfigID: config.ThumbsDownScore.ScoreConfigID,
				Comment:       config.ThumbsDownScore.Comment,
			})
		}
	}

	// Process rating
	if feedback.Rating.Valid && config.RatingScores != nil {
		if ratingMapping, exists := config.RatingScores[int(feedback.Rating.Int32)]; exists {
			scores = append(scores, &scoreFromMapping{
				Name:          fmt.Sprintf("rating_%d_star_feedback", int(feedback.Rating.Int32)),
				Value:         ratingMapping.Value,
				DataType:      "numeric",
				ScoreConfigID: ratingMapping.ScoreConfigID,
				Comment:       ratingMapping.Comment,
			})
		}
	}

	// Process comment (basic keyword matching)
	if feedback.Comment != "" && config.CommentAnalysis != nil && config.CommentAnalysis.Enabled {
		commentScores := s.analyzeComment(feedback.Comment, config.CommentAnalysis)
		scores = append(scores, commentScores...)
	}

	return scores
}

// analyzeComment performs basic analysis on feedback comments
func (s *FeedbackScoreMappingService) analyzeComment(comment string, analysis *domain.CommentAnalysisConfig) []*scoreFromMapping {
	var scores []*scoreFromMapping

	// Convert comment to lowercase for case-insensitive matching
	commentLower := strings.ToLower(comment)

	// Check keyword mappings
	for _, keywordMapping := range analysis.KeywordMappings {
		for _, keyword := range keywordMapping.Keywords {
			if strings.Contains(commentLower, strings.ToLower(keyword)) {
				scores = append(scores, &scoreFromMapping{
					Name:          fmt.Sprintf("comment_keyword_%s", keyword),
					Value:         keywordMapping.Mapping.Value,
					DataType:      "numeric",
					ScoreConfigID: keywordMapping.Mapping.ScoreConfigID,
					Comment:       keywordMapping.Mapping.Comment,
				})
				break // Only match once per keyword mapping
			}
		}
	}

	// TODO: Add sentiment analysis when we integrate with external services

	return scores
}

// scoreFromMapping represents a score generated from feedback mapping
type scoreFromMapping struct {
	Name          string
	Value         float64
	DataType      string
	ScoreConfigID uuid.UUID
	Comment       string
}
