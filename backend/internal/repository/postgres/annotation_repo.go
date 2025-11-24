package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// AnnotationRepository handles annotation data access
type AnnotationRepository struct {
	db *pgxpool.Pool
}

// NewAnnotationRepository creates a new annotation repository
func NewAnnotationRepository(db *pgxpool.Pool) *AnnotationRepository {
	return &AnnotationRepository{db: db}
}

// Queue CRUD operations

// CreateQueue creates a new annotation queue
func (r *AnnotationRepository) CreateQueue(ctx context.Context, queue *domain.AnnotationQueue) error {
	scoreConfigsJSON, err := json.Marshal(queue.ScoreConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal score configs: %w", err)
	}

	configJSON, err := json.Marshal(queue.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	itemSourceConfigJSON, err := json.Marshal(queue.ItemSourceConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal item source config: %w", err)
	}

	query := `
		INSERT INTO annotation_queues (
			id, project_id, name, description, score_configs, config,
			item_source, item_source_config, assignment_strategy,
			max_annotations_per_item, instructions, is_active,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = r.db.Exec(ctx, query,
		queue.ID,
		queue.ProjectID,
		queue.Name,
		queue.Description,
		scoreConfigsJSON,
		configJSON,
		queue.ItemSource,
		itemSourceConfigJSON,
		queue.AssignmentStrategy,
		queue.MaxAnnotationsPerItem,
		queue.Instructions,
		queue.IsActive,
		queue.CreatedAt,
		queue.UpdatedAt,
	)
	return err
}

// GetQueueByID retrieves an annotation queue by ID
func (r *AnnotationRepository) GetQueueByID(ctx context.Context, id string) (*domain.AnnotationQueue, error) {
	var queue domain.AnnotationQueue
	query := `
		SELECT id, project_id, name, description, score_configs, config,
			   item_source, item_source_config, assignment_strategy,
			   max_annotations_per_item, instructions, is_active,
			   created_at, updated_at
		FROM annotation_queues
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &queue, query, id)
	if err != nil {
		return nil, err
	}
	return &queue, nil
}

// ListQueuesByProject retrieves annotation queues for a project
func (r *AnnotationRepository) ListQueuesByProject(ctx context.Context, projectID string) ([]domain.AnnotationQueue, error) {
	var queues []domain.AnnotationQueue
	query := `
		SELECT id, project_id, name, description, score_configs, config,
			   item_source, item_source_config, assignment_strategy,
			   max_annotations_per_item, instructions, is_active,
			   created_at, updated_at
		FROM annotation_queues
		WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &queues, query, projectID)
	return queues, err
}

// UpdateQueue updates an annotation queue
func (r *AnnotationRepository) UpdateQueue(ctx context.Context, queue *domain.AnnotationQueue) error {
	scoreConfigsJSON, err := json.Marshal(queue.ScoreConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal score configs: %w", err)
	}

	configJSON, err := json.Marshal(queue.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	itemSourceConfigJSON, err := json.Marshal(queue.ItemSourceConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal item source config: %w", err)
	}

	query := `
		UPDATE annotation_queues SET
			name = $2, description = $3, score_configs = $4, config = $5,
			item_source = $6, item_source_config = $7, assignment_strategy = $8,
			max_annotations_per_item = $9, instructions = $10, is_active = $11,
			updated_at = $12
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err = r.db.Exec(ctx, query,
		queue.ID,
		queue.Name,
		queue.Description,
		scoreConfigsJSON,
		configJSON,
		queue.ItemSource,
		itemSourceConfigJSON,
		queue.AssignmentStrategy,
		queue.MaxAnnotationsPerItem,
		queue.Instructions,
		queue.IsActive,
		queue.UpdatedAt,
	)
	return err
}

// DeleteQueue soft deletes an annotation queue
func (r *AnnotationRepository) DeleteQueue(ctx context.Context, id string) error {
	query := `
		UPDATE annotation_queues SET
			deleted_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Queue Item CRUD operations

// CreateQueueItem creates a new queue item
func (r *AnnotationRepository) CreateQueueItem(ctx context.Context, item *domain.AnnotationQueueItem) error {
	itemDataJSON, err := json.Marshal(item.ItemData)
	if err != nil {
		return fmt.Errorf("failed to marshal item data: %w", err)
	}

	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO annotation_queue_items (
			id, queue_id, item_type, item_id, item_data, metadata,
			priority, max_annotations, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = r.db.Exec(ctx, query,
		item.ID,
		item.QueueID,
		item.ItemType,
		item.ItemID,
		itemDataJSON,
		metadataJSON,
		item.Priority,
		item.MaxAnnotations,
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

// GetQueueItemByID retrieves a queue item by ID
func (r *AnnotationRepository) GetQueueItemByID(ctx context.Context, id string) (*domain.AnnotationQueueItem, error) {
	var item domain.AnnotationQueueItem
	query := `
		SELECT id, queue_id, item_type, item_id, item_data, metadata,
			   priority, max_annotations, created_at, updated_at
		FROM annotation_queue_items
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &item, query, id)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListQueueItems retrieves queue items for a queue
func (r *AnnotationRepository) ListQueueItems(ctx context.Context, queueID string, limit, offset int) ([]domain.AnnotationQueueItem, error) {
	var items []domain.AnnotationQueueItem
	query := `
		SELECT id, queue_id, item_type, item_id, item_data, metadata,
			   priority, max_annotations, created_at, updated_at
		FROM annotation_queue_items
		WHERE queue_id = $1
		ORDER BY priority DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := pgxscan.Select(ctx, r.db, &items, query, queueID, limit, offset)
	return items, err
}

// GetNextAssignableItem gets the next item that can be assigned to a user
func (r *AnnotationRepository) GetNextAssignableItem(ctx context.Context, queueID, userID string) (*domain.AnnotationQueueItem, error) {
	var item domain.AnnotationQueueItem
	query := `
		SELECT i.id, i.queue_id, i.item_type, i.item_id, i.item_data, i.metadata,
			   i.priority, i.max_annotations, i.created_at, i.updated_at
		FROM annotation_queue_items i
		WHERE i.queue_id = $1
		AND i.id NOT IN (
			SELECT queue_item_id FROM annotation_assignments
			WHERE user_id = $2 AND status IN ('assigned', 'in_progress', 'completed')
		)
		AND (
			SELECT COUNT(*) FROM annotation_assignments
			WHERE queue_item_id = i.id AND status = 'completed'
		) < i.max_annotations
		ORDER BY i.priority DESC, i.created_at ASC
		LIMIT 1
	`
	err := pgxscan.Get(ctx, r.db, &item, query, queueID, userID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Assignment CRUD operations

// CreateAssignment creates a new assignment
func (r *AnnotationRepository) CreateAssignment(ctx context.Context, assignment *domain.AnnotationAssignment) error {
	query := `
		INSERT INTO annotation_assignments (
			id, queue_item_id, user_id, status, assigned_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		assignment.ID,
		assignment.QueueItemID,
		assignment.UserID,
		assignment.Status,
		assignment.AssignedAt,
		assignment.CreatedAt,
		assignment.UpdatedAt,
	)
	return err
}

// GetAssignmentByID retrieves an assignment by ID
func (r *AnnotationRepository) GetAssignmentByID(ctx context.Context, id string) (*domain.AnnotationAssignment, error) {
	var assignment domain.AnnotationAssignment
	query := `
		SELECT id, queue_item_id, user_id, status, assigned_at,
			   started_at, completed_at, skipped_at, notes,
			   created_at, updated_at
		FROM annotation_assignments
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &assignment, query, id)
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// GetAssignmentByQueueItemAndUser retrieves an assignment by queue item and user
func (r *AnnotationRepository) GetAssignmentByQueueItemAndUser(ctx context.Context, queueItemID, userID string) (*domain.AnnotationAssignment, error) {
	var assignment domain.AnnotationAssignment
	query := `
		SELECT id, queue_item_id, user_id, status, assigned_at,
			   started_at, completed_at, skipped_at, notes,
			   created_at, updated_at
		FROM annotation_assignments
		WHERE queue_item_id = $1 AND user_id = $2
	`
	err := pgxscan.Get(ctx, r.db, &assignment, query, queueItemID, userID)
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// UpdateAssignment updates an assignment
func (r *AnnotationRepository) UpdateAssignment(ctx context.Context, assignment *domain.AnnotationAssignment) error {
	query := `
		UPDATE annotation_assignments SET
			status = $2, started_at = $3, completed_at = $4,
			skipped_at = $5, notes = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		assignment.ID,
		assignment.Status,
		assignment.StartedAt,
		assignment.CompletedAt,
		assignment.SkippedAt,
		assignment.Notes,
		assignment.UpdatedAt,
	)
	return err
}

// ListAssignmentsByUser retrieves assignments for a user
func (r *AnnotationRepository) ListAssignmentsByUser(ctx context.Context, userID string, status *string, limit, offset int) ([]domain.AnnotationAssignment, error) {
	var assignments []domain.AnnotationAssignment
	var args []interface{}
	args = append(args, userID)

	query := `
		SELECT a.id, a.queue_item_id, a.user_id, a.status, a.assigned_at,
			   a.started_at, a.completed_at, a.skipped_at, a.notes,
			   a.created_at, a.updated_at
		FROM annotation_assignments a
		WHERE a.user_id = $1
	`

	if status != nil {
		args = append(args, *status)
		query += fmt.Sprintf(" AND a.status = $%d", len(args))
	}

	query += " ORDER BY a.assigned_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1) + " OFFSET $" + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	err := pgxscan.Select(ctx, r.db, &assignments, query, args...)
	return assignments, err
}

// Annotation CRUD operations

// CreateAnnotation creates a new annotation
func (r *AnnotationRepository) CreateAnnotation(ctx context.Context, annotation *domain.Annotation) error {
	scoresJSON, err := json.Marshal(annotation.Scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores: %w", err)
	}

	query := `
		INSERT INTO annotations (
			id, assignment_id, queue_id, queue_item_id, user_id,
			scores, labels, notes, confidence_score, annotation_time,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = r.db.Exec(ctx, query,
		annotation.ID,
		annotation.AssignmentID,
		annotation.QueueID,
		annotation.QueueItemID,
		annotation.UserID,
		scoresJSON,
		annotation.Labels,
		annotation.Notes,
		annotation.ConfidenceScore,
		annotation.AnnotationTime,
		annotation.CreatedAt,
		annotation.UpdatedAt,
	)
	return err
}

// GetAnnotationByID retrieves an annotation by ID
func (r *AnnotationRepository) GetAnnotationByID(ctx context.Context, id string) (*domain.Annotation, error) {
	var annotation domain.Annotation
	query := `
		SELECT id, assignment_id, queue_id, queue_item_id, user_id,
			   scores, labels, notes, confidence_score, annotation_time,
			   created_at, updated_at
		FROM annotations
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &annotation, query, id)
	if err != nil {
		return nil, err
	}
	return &annotation, nil
}

// ListAnnotationsByQueueItem retrieves annotations for a queue item
func (r *AnnotationRepository) ListAnnotationsByQueueItem(ctx context.Context, queueItemID string) ([]domain.Annotation, error) {
	var annotations []domain.Annotation
	query := `
		SELECT id, assignment_id, queue_id, queue_item_id, user_id,
			   scores, labels, notes, confidence_score, annotation_time,
			   created_at, updated_at
		FROM annotations
		WHERE queue_item_id = $1
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &annotations, query, queueItemID)
	return annotations, err
}

// ListAnnotationsByQueue retrieves annotations for a queue
func (r *AnnotationRepository) ListAnnotationsByQueue(ctx context.Context, queueID string, limit, offset int) ([]domain.Annotation, error) {
	var annotations []domain.Annotation
	query := `
		SELECT id, assignment_id, queue_id, queue_item_id, user_id,
			   scores, labels, notes, confidence_score, annotation_time,
			   created_at, updated_at
		FROM annotations
		WHERE queue_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := pgxscan.Select(ctx, r.db, &annotations, query, queueID, limit, offset)
	return annotations, err
}

// UpdateAnnotation updates an annotation
func (r *AnnotationRepository) UpdateAnnotation(ctx context.Context, annotation *domain.Annotation) error {
	scoresJSON, err := json.Marshal(annotation.Scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores: %w", err)
	}

	query := `
		UPDATE annotations SET
			scores = $2, labels = $3, notes = $4,
			confidence_score = $5, annotation_time = $6, updated_at = $7
		WHERE id = $1
	`
	_, err = r.db.Exec(ctx, query,
		annotation.ID,
		scoresJSON,
		annotation.Labels,
		annotation.Notes,
		annotation.ConfidenceScore,
		annotation.AnnotationTime,
		annotation.UpdatedAt,
	)
	return err
}

// Statistics and Analytics

// GetQueueStats gets statistics for a queue
func (r *AnnotationRepository) GetQueueStats(ctx context.Context, queueID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(DISTINCT i.id) as total_items,
			COUNT(DISTINCT CASE WHEN a.status = 'completed' THEN a.id END) as completed_assignments,
			COUNT(DISTINCT CASE WHEN a.status IN ('assigned', 'in_progress') THEN a.id END) as active_assignments,
			COUNT(DISTINCT an.id) as total_annotations
		FROM annotation_queue_items i
		LEFT JOIN annotation_assignments a ON i.id = a.queue_item_id
		LEFT JOIN annotations an ON i.id = an.queue_item_id
		WHERE i.queue_id = $1
	`

	var stats struct {
		TotalItems           int `db:"total_items"`
		CompletedAssignments int `db:"completed_assignments"`
		ActiveAssignments    int `db:"active_assignments"`
		TotalAnnotations     int `db:"total_annotations"`
	}

	err := pgxscan.Get(ctx, r.db, &stats, query, queueID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalItems":           stats.TotalItems,
		"completedAssignments": stats.CompletedAssignments,
		"activeAssignments":    stats.ActiveAssignments,
		"totalAnnotations":     stats.TotalAnnotations,
	}, nil
}

// GetUserStats gets annotation statistics for a user
func (r *AnnotationRepository) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_assignments,
			COUNT(CASE WHEN status = 'assigned' THEN 1 END) as assigned_assignments,
			COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress_assignments,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) as skipped_assignments,
			COUNT(DISTINCT an.id) as total_annotations
		FROM annotation_assignments a
		LEFT JOIN annotations an ON a.id = an.assignment_id
		WHERE a.user_id = $1
	`

	var stats struct {
		CompletedAssignments  int `db:"completed_assignments"`
		AssignedAssignments   int `db:"assigned_assignments"`
		InProgressAssignments int `db:"in_progress_assignments"`
		SkippedAssignments    int `db:"skipped_assignments"`
		TotalAnnotations      int `db:"total_annotations"`
	}

	err := pgxscan.Get(ctx, r.db, &stats, query, userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"completedAssignments":  stats.CompletedAssignments,
		"assignedAssignments":   stats.AssignedAssignments,
		"inProgressAssignments": stats.InProgressAssignments,
		"skippedAssignments":    stats.SkippedAssignments,
		"totalAnnotations":      stats.TotalAnnotations,
	}, nil
}

// CalculateInterAnnotatorAgreement calculates agreement metrics for a queue item and score config
func (r *AnnotationRepository) CalculateInterAnnotatorAgreement(ctx context.Context, queueID, queueItemID, scoreConfigName string) (*domain.InterAnnotatorAgreement, error) {
	// Get all annotations for this queue item
	annotations, err := r.ListAnnotationsByQueueItem(ctx, queueItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get annotations: %w", err)
	}

	if len(annotations) < 2 {
		return &domain.InterAnnotatorAgreement{
			QueueID:         uuid.MustParse(queueID),
			QueueItemID:     uuid.MustParse(queueItemID),
			ScoreConfigName: scoreConfigName,
			AgreementType:   "percentage",
			AgreementValue:  sql.NullFloat64{Valid: false},
			AnnotatorCount:  len(annotations),
			CalculatedAt:    time.Now(),
		}, nil
	}

	// Extract scores for the specific config
	var scores []float64
	for _, annotation := range annotations {
		var scoresMap map[string]interface{}
		if len(annotation.Scores) > 0 {
			if err := json.Unmarshal(annotation.Scores, &scoresMap); err != nil {
				continue
			}
			if score, ok := scoresMap[scoreConfigName]; ok {
				if scoreVal, ok := score.(float64); ok {
					scores = append(scores, scoreVal)
				}
			}
		}
	}

	if len(scores) < 2 {
		return &domain.InterAnnotatorAgreement{
			QueueID:         uuid.MustParse(queueID),
			QueueItemID:     uuid.MustParse(queueItemID),
			ScoreConfigName: scoreConfigName,
			AgreementType:   "percentage",
			AgreementValue:  sql.NullFloat64{Valid: false},
			AnnotatorCount:  len(annotations),
			CalculatedAt:    time.Now(),
		}, nil
	}

	// Calculate percentage agreement (simplified - all values within 0.1 of each other)
	agreement := r.calculatePercentageAgreement(scores)

	// Store the result
	agreementRecord := &domain.InterAnnotatorAgreement{
		QueueID:         uuid.MustParse(queueID),
		QueueItemID:     uuid.MustParse(queueItemID),
		ScoreConfigName: scoreConfigName,
		AgreementType:   "percentage",
		AgreementValue:  sql.NullFloat64{Float64: agreement, Valid: true},
		AnnotatorCount:  len(scores),
		CalculatedAt:    time.Now(),
	}

	// Upsert the agreement record
	err = r.upsertInterAnnotatorAgreement(ctx, agreementRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to store agreement: %w", err)
	}

	return agreementRecord, nil
}

// calculatePercentageAgreement calculates simple percentage agreement
func (r *AnnotationRepository) calculatePercentageAgreement(scores []float64) float64 {
	if len(scores) < 2 {
		return 0
	}

	// For simplicity, consider agreement if all scores are within 0.1 of the mean
	mean := 0.0
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	agreeing := 0
	for _, score := range scores {
		if math.Abs(score-mean) <= 0.1 {
			agreeing++
		}
	}

	return float64(agreeing) / float64(len(scores))
}

// upsertInterAnnotatorAgreement inserts or updates an inter-annotator agreement record
func (r *AnnotationRepository) upsertInterAnnotatorAgreement(ctx context.Context, agreement *domain.InterAnnotatorAgreement) error {
	query := `
		INSERT INTO inter_annotator_agreements (
			queue_id, queue_item_id, score_config_name, agreement_type,
			agreement_value, annotator_count, calculated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (queue_id, queue_item_id, score_config_name, agreement_type)
		DO UPDATE SET
			agreement_value = EXCLUDED.agreement_value,
			annotator_count = EXCLUDED.annotator_count,
			calculated_at = EXCLUDED.calculated_at
	`

	_, err := r.db.Exec(ctx, query,
		agreement.QueueID,
		agreement.QueueItemID,
		agreement.ScoreConfigName,
		agreement.AgreementType,
		agreement.AgreementValue,
		agreement.AnnotatorCount,
		agreement.CalculatedAt,
	)

	return err
}

// GetInterAnnotatorAgreements retrieves agreement metrics for a queue
func (r *AnnotationRepository) GetInterAnnotatorAgreements(ctx context.Context, queueID string, limit, offset int) ([]domain.InterAnnotatorAgreement, error) {
	var agreements []domain.InterAnnotatorAgreement
	query := `
		SELECT id, queue_id, queue_item_id, score_config_name, agreement_type,
			   agreement_value, annotator_count, calculated_at
		FROM inter_annotator_agreements
		WHERE queue_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2 OFFSET $3
	`

	err := pgxscan.Select(ctx, r.db, &agreements, query, queueID, limit, offset)
	return agreements, err
}

// GetQueueAgreementStats gets overall agreement statistics for a queue
func (r *AnnotationRepository) GetQueueAgreementStats(ctx context.Context, queueID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_agreements,
			AVG(agreement_value) as avg_agreement,
			MIN(agreement_value) as min_agreement,
			MAX(agreement_value) as max_agreement,
			COUNT(CASE WHEN agreement_value >= 0.8 THEN 1 END) as high_agreement_count,
			COUNT(CASE WHEN agreement_value < 0.5 THEN 1 END) as low_agreement_count
		FROM inter_annotator_agreements
		WHERE queue_id = $1 AND agreement_value IS NOT NULL
	`

	var stats struct {
		TotalAgreements    int      `db:"total_agreements"`
		AvgAgreement       *float64 `db:"avg_agreement"`
		MinAgreement       *float64 `db:"min_agreement"`
		MaxAgreement       *float64 `db:"max_agreement"`
		HighAgreementCount int      `db:"high_agreement_count"`
		LowAgreementCount  int      `db:"low_agreement_count"`
	}

	err := pgxscan.Get(ctx, r.db, &stats, query, queueID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalAgreements":    stats.TotalAgreements,
		"avgAgreement":       stats.AvgAgreement,
		"minAgreement":       stats.MinAgreement,
		"maxAgreement":       stats.MaxAgreement,
		"highAgreementCount": stats.HighAgreementCount,
		"lowAgreementCount":  stats.LowAgreementCount,
	}, nil
}
