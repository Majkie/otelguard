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

// AnnotationService handles annotation business logic
type AnnotationService struct {
	annotationRepo *postgres.AnnotationRepository
	projectRepo    *postgres.ProjectRepository
	userRepo       *postgres.UserRepository
	logger         *zap.Logger
}

// NewAnnotationService creates a new annotation service
func NewAnnotationService(
	annotationRepo *postgres.AnnotationRepository,
	projectRepo *postgres.ProjectRepository,
	userRepo *postgres.UserRepository,
	logger *zap.Logger,
) *AnnotationService {
	return &AnnotationService{
		annotationRepo: annotationRepo,
		projectRepo:    projectRepo,
		userRepo:       userRepo,
		logger:         logger,
	}
}

// Queue Management

// CreateQueue creates a new annotation queue
func (s *AnnotationService) CreateQueue(ctx context.Context, req *domain.AnnotationQueueCreate) (*domain.AnnotationQueue, error) {
	// Verify project exists and user has access
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, domain.ErrNotFound
	}

	queue := &domain.AnnotationQueue{
		ID:                    uuid.New(),
		ProjectID:             req.ProjectID,
		Name:                  req.Name,
		Description:           req.Description,
		ScoreConfigs:          []byte("[]"),
		Config:                []byte("{}"),
		ItemSource:            "manual",
		ItemSourceConfig:      []byte("{}"),
		AssignmentStrategy:    "round_robin",
		MaxAnnotationsPerItem: 1,
		Instructions:          req.Instructions,
		IsActive:              true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Handle score configs
	if req.ScoreConfigs != nil {
		scoreConfigsJSON, err := json.Marshal(req.ScoreConfigs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal score configs: %w", err)
		}
		queue.ScoreConfigs = scoreConfigsJSON
	}

	// Handle config
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		queue.Config = configJSON
	}

	// Handle item source config
	if req.ItemSourceConfig != nil {
		itemSourceConfigJSON, err := json.Marshal(req.ItemSourceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item source config: %w", err)
		}
		queue.ItemSourceConfig = itemSourceConfigJSON
	}

	if req.ItemSource != "" {
		queue.ItemSource = req.ItemSource
	}
	if req.AssignmentStrategy != "" {
		queue.AssignmentStrategy = req.AssignmentStrategy
	}
	if req.MaxAnnotationsPerItem > 0 {
		queue.MaxAnnotationsPerItem = req.MaxAnnotationsPerItem
	}

	err = s.annotationRepo.CreateQueue(ctx, queue)
	if err != nil {
		s.logger.Error("failed to create annotation queue", zap.Error(err))
		return nil, fmt.Errorf("failed to create annotation queue: %w", err)
	}

	s.logger.Info("created annotation queue",
		zap.String("queue_id", queue.ID.String()),
		zap.String("project_id", queue.ProjectID.String()),
		zap.String("name", queue.Name))

	return queue, nil
}

// GetQueue retrieves an annotation queue by ID
func (s *AnnotationService) GetQueue(ctx context.Context, queueID string) (*domain.AnnotationQueue, error) {
	queue, err := s.annotationRepo.GetQueueByID(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	return queue, nil
}

// ListQueuesByProject retrieves annotation queues for a project
func (s *AnnotationService) ListQueuesByProject(ctx context.Context, projectID string) ([]domain.AnnotationQueue, error) {
	queues, err := s.annotationRepo.ListQueuesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}
	return queues, nil
}

// UpdateQueue updates an annotation queue
func (s *AnnotationService) UpdateQueue(ctx context.Context, queueID string, req *domain.AnnotationQueueUpdate) (*domain.AnnotationQueue, error) {
	queue, err := s.annotationRepo.GetQueueByID(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	if queue == nil {
		return nil, domain.ErrNotFound
	}

	// Apply updates
	if req.Name != nil {
		queue.Name = *req.Name
	}
	if req.Description != nil {
		queue.Description = *req.Description
	}
	if req.ScoreConfigs != nil {
		scoreConfigsJSON, err := json.Marshal(req.ScoreConfigs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal score configs: %w", err)
		}
		queue.ScoreConfigs = scoreConfigsJSON
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		queue.Config = configJSON
	}
	if req.ItemSource != nil {
		queue.ItemSource = *req.ItemSource
	}
	if req.ItemSourceConfig != nil {
		itemSourceConfigJSON, err := json.Marshal(req.ItemSourceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item source config: %w", err)
		}
		queue.ItemSourceConfig = itemSourceConfigJSON
	}
	if req.AssignmentStrategy != nil {
		queue.AssignmentStrategy = *req.AssignmentStrategy
	}
	if req.MaxAnnotationsPerItem != nil {
		queue.MaxAnnotationsPerItem = *req.MaxAnnotationsPerItem
	}
	if req.Instructions != nil {
		queue.Instructions = *req.Instructions
	}
	if req.IsActive != nil {
		queue.IsActive = *req.IsActive
	}

	queue.UpdatedAt = time.Now()

	err = s.annotationRepo.UpdateQueue(ctx, queue)
	if err != nil {
		s.logger.Error("failed to update annotation queue", zap.Error(err))
		return nil, fmt.Errorf("failed to update annotation queue: %w", err)
	}

	return queue, nil
}

// DeleteQueue soft deletes an annotation queue
func (s *AnnotationService) DeleteQueue(ctx context.Context, queueID string) error {
	err := s.annotationRepo.DeleteQueue(ctx, queueID)
	if err != nil {
		s.logger.Error("failed to delete annotation queue", zap.Error(err))
		return fmt.Errorf("failed to delete annotation queue: %w", err)
	}

	s.logger.Info("deleted annotation queue", zap.String("queue_id", queueID))
	return nil
}

// Queue Item Management

// CreateQueueItem creates a new queue item
func (s *AnnotationService) CreateQueueItem(ctx context.Context, req *domain.AnnotationQueueItemCreate) (*domain.AnnotationQueueItem, error) {
	// Verify queue exists
	queue, err := s.annotationRepo.GetQueueByID(ctx, req.QueueID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	if queue == nil {
		return nil, domain.ErrNotFound
	}

	item := &domain.AnnotationQueueItem{
		ID:             uuid.New(),
		QueueID:        req.QueueID,
		ItemType:       req.ItemType,
		ItemID:         req.ItemID,
		ItemData:       []byte("{}"),
		Metadata:       []byte("{}"),
		Priority:       req.Priority,
		MaxAnnotations: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Handle item data
	if req.ItemData != nil {
		itemDataJSON, err := json.Marshal(req.ItemData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item data: %w", err)
		}
		item.ItemData = itemDataJSON
	}

	// Handle metadata
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		item.Metadata = metadataJSON
	}

	if req.MaxAnnotations > 0 {
		item.MaxAnnotations = req.MaxAnnotations
	}

	err = s.annotationRepo.CreateQueueItem(ctx, item)
	if err != nil {
		s.logger.Error("failed to create queue item", zap.Error(err))
		return nil, fmt.Errorf("failed to create queue item: %w", err)
	}

	return item, nil
}

// Assignment Management

// AssignNextItem assigns the next available item to a user
func (s *AnnotationService) AssignNextItem(ctx context.Context, queueID, userID string) (*domain.AnnotationAssignment, error) {
	// Get next assignable item
	item, err := s.annotationRepo.GetNextAssignableItem(ctx, queueID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next assignable item: %w", err)
	}
	if item == nil {
		return nil, domain.ErrNotFound // No items available
	}

	// Check if assignment already exists
	existingAssignment, err := s.annotationRepo.GetAssignmentByQueueItemAndUser(ctx, item.ID.String(), userID)
	if err == nil && existingAssignment != nil {
		return existingAssignment, nil // Return existing assignment
	}

	// Create new assignment
	assignment := &domain.AnnotationAssignment{
		ID:          uuid.New(),
		QueueItemID: item.ID,
		UserID:      uuid.MustParse(userID),
		Status:      "assigned",
		AssignedAt:  time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = s.annotationRepo.CreateAssignment(ctx, assignment)
	if err != nil {
		s.logger.Error("failed to create assignment", zap.Error(err))
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	s.logger.Info("assigned item to user",
		zap.String("assignment_id", assignment.ID.String()),
		zap.String("queue_item_id", item.ID.String()),
		zap.String("user_id", userID))

	return assignment, nil
}

// StartAssignment marks an assignment as in progress
func (s *AnnotationService) StartAssignment(ctx context.Context, assignmentID, userID string) error {
	assignment, err := s.annotationRepo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return domain.ErrNotFound
	}

	// Verify ownership
	if assignment.UserID.String() != userID {
		return domain.ErrForbidden
	}

	assignment.Status = "in_progress"
	assignment.StartedAt = sql.NullTime{Time: time.Now(), Valid: true}
	assignment.UpdatedAt = time.Now()

	err = s.annotationRepo.UpdateAssignment(ctx, assignment)
	if err != nil {
		s.logger.Error("failed to start assignment", zap.Error(err))
		return fmt.Errorf("failed to start assignment: %w", err)
	}

	return nil
}

// SkipAssignment marks an assignment as skipped
func (s *AnnotationService) SkipAssignment(ctx context.Context, assignmentID, userID, notes string) error {
	assignment, err := s.annotationRepo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return domain.ErrNotFound
	}

	// Verify ownership
	if assignment.UserID.String() != userID {
		return domain.ErrForbidden
	}

	assignment.Status = "skipped"
	assignment.SkippedAt = sql.NullTime{Time: time.Now(), Valid: true}
	assignment.Notes = notes
	assignment.UpdatedAt = time.Now()

	err = s.annotationRepo.UpdateAssignment(ctx, assignment)
	if err != nil {
		s.logger.Error("failed to skip assignment", zap.Error(err))
		return fmt.Errorf("failed to skip assignment: %w", err)
	}

	return nil
}

// Annotation Management

// CreateAnnotation creates a new annotation
func (s *AnnotationService) CreateAnnotation(ctx context.Context, req *domain.AnnotationCreate) (*domain.Annotation, error) {
	// Get assignment
	assignment, err := s.annotationRepo.GetAssignmentByID(ctx, req.AssignmentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return nil, domain.ErrNotFound
	}

	// Get queue item
	queueItem, err := s.annotationRepo.GetQueueItemByID(ctx, assignment.QueueItemID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}
	if queueItem == nil {
		return nil, domain.ErrNotFound
	}

	annotation := &domain.Annotation{
		ID:              uuid.New(),
		AssignmentID:    req.AssignmentID,
		QueueID:         queueItem.QueueID,
		QueueItemID:     assignment.QueueItemID,
		UserID:          assignment.UserID,
		Scores:          []byte("{}"),
		Labels:          req.Labels,
		Notes:           req.Notes,
		ConfidenceScore: sql.NullFloat64{Valid: false},
		AnnotationTime:  sql.NullString{Valid: false},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Handle scores
	if req.Scores != nil {
		scoresJSON, err := json.Marshal(req.Scores)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scores: %w", err)
		}
		annotation.Scores = scoresJSON
	}

	// Handle confidence score
	if req.ConfidenceScore != nil {
		annotation.ConfidenceScore = sql.NullFloat64{Float64: *req.ConfidenceScore, Valid: true}
	}

	// Handle annotation time
	if req.AnnotationTime != nil {
		annotation.AnnotationTime = sql.NullString{String: *req.AnnotationTime, Valid: true}
	}

	err = s.annotationRepo.CreateAnnotation(ctx, annotation)
	if err != nil {
		s.logger.Error("failed to create annotation", zap.Error(err))
		return nil, fmt.Errorf("failed to create annotation: %w", err)
	}

	// Mark assignment as completed
	assignment.Status = "completed"
	assignment.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	assignment.UpdatedAt = time.Now()

	err = s.annotationRepo.UpdateAssignment(ctx, assignment)
	if err != nil {
		s.logger.Error("failed to update assignment status", zap.Error(err))
		// Don't fail the annotation creation for this
	}

	s.logger.Info("created annotation",
		zap.String("annotation_id", annotation.ID.String()),
		zap.String("assignment_id", req.AssignmentID.String()),
		zap.String("user_id", assignment.UserID.String()))

	return annotation, nil
}

// GetAnnotation retrieves an annotation by ID
func (s *AnnotationService) GetAnnotation(ctx context.Context, annotationID string) (*domain.Annotation, error) {
	annotation, err := s.annotationRepo.GetAnnotationByID(ctx, annotationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get annotation: %w", err)
	}
	return annotation, nil
}

// ListAnnotationsByQueueItem retrieves annotations for a queue item
func (s *AnnotationService) ListAnnotationsByQueueItem(ctx context.Context, queueItemID string) ([]domain.Annotation, error) {
	annotations, err := s.annotationRepo.ListAnnotationsByQueueItem(ctx, queueItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to list annotations: %w", err)
	}
	return annotations, nil
}

// ListAnnotationsByQueue retrieves all annotations for a queue
func (s *AnnotationService) ListAnnotationsByQueue(ctx context.Context, queueID string, limit, offset int) ([]domain.Annotation, error) {
	annotations, err := s.annotationRepo.ListAnnotationsByQueue(ctx, queueID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list annotations by queue: %w", err)
	}
	return annotations, nil
}

// ListUserAssignments retrieves assignments for a user
func (s *AnnotationService) ListUserAssignments(ctx context.Context, userID string, status *string, limit, offset int) ([]domain.AnnotationAssignment, error) {
	assignments, err := s.annotationRepo.ListAssignmentsByUser(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user assignments: %w", err)
	}
	return assignments, nil
}

// Statistics and Analytics

// GetQueueStats gets statistics for a queue
func (s *AnnotationService) GetQueueStats(ctx context.Context, queueID string) (map[string]interface{}, error) {
	return s.annotationRepo.GetQueueStats(ctx, queueID)
}

// ListQueueItems retrieves queue items for a queue
func (s *AnnotationService) ListQueueItems(ctx context.Context, queueID string, limit, offset int) ([]domain.AnnotationQueueItem, error) {
	return s.annotationRepo.ListQueueItems(ctx, queueID, limit, offset)
}

// GetUserStats gets annotation statistics for a user
func (s *AnnotationService) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	return s.annotationRepo.GetUserStats(ctx, userID)
}

// CalculateInterAnnotatorAgreement calculates agreement metrics for a queue item
func (s *AnnotationService) CalculateInterAnnotatorAgreement(ctx context.Context, queueID, queueItemID, scoreConfigName string) (*domain.InterAnnotatorAgreement, error) {
	agreement, err := s.annotationRepo.CalculateInterAnnotatorAgreement(ctx, queueID, queueItemID, scoreConfigName)
	if err != nil {
		s.logger.Error("failed to calculate inter-annotator agreement", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate inter-annotator agreement: %w", err)
	}
	return agreement, nil
}

// GetInterAnnotatorAgreements retrieves agreement metrics for a queue
func (s *AnnotationService) GetInterAnnotatorAgreements(ctx context.Context, queueID string, limit, offset int) ([]domain.InterAnnotatorAgreement, error) {
	agreements, err := s.annotationRepo.GetInterAnnotatorAgreements(ctx, queueID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get inter-annotator agreements", zap.Error(err))
		return nil, fmt.Errorf("failed to get inter-annotator agreements: %w", err)
	}
	return agreements, nil
}

// GetQueueAgreementStats gets overall agreement statistics for a queue
func (s *AnnotationService) GetQueueAgreementStats(ctx context.Context, queueID string) (map[string]interface{}, error) {
	stats, err := s.annotationRepo.GetQueueAgreementStats(ctx, queueID)
	if err != nil {
		s.logger.Error("failed to get queue agreement stats", zap.Error(err))
		return nil, fmt.Errorf("failed to get queue agreement stats: %w", err)
	}
	return stats, nil
}
