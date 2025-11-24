package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// FeedbackRepository handles user feedback data access
type FeedbackRepository struct {
	db *pgxpool.Pool
}

// NewFeedbackRepository creates a new feedback repository
func NewFeedbackRepository(db *pgxpool.Pool) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

// Create creates a new user feedback record
func (r *FeedbackRepository) Create(ctx context.Context, feedback *domain.UserFeedback) error {
	metadataJSON, err := json.Marshal(feedback.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO user_feedback (
			id, project_id, user_id, session_id, trace_id, span_id,
			item_type, item_id, thumbs_up, rating, comment,
			metadata, user_agent, ip_address, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err = r.db.Exec(ctx, query,
		feedback.ID, feedback.ProjectID, feedback.UserID, feedback.SessionID,
		feedback.TraceID, feedback.SpanID, feedback.ItemType, feedback.ItemID,
		feedback.ThumbsUp, feedback.Rating, feedback.Comment,
		metadataJSON, feedback.UserAgent, feedback.IPAddress,
		feedback.CreatedAt, feedback.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create feedback: %w", err)
	}

	return nil
}

// GetByID retrieves feedback by ID
func (r *FeedbackRepository) GetByID(ctx context.Context, id string) (*domain.UserFeedback, error) {
	query := `
		SELECT id, project_id, user_id, session_id, trace_id, span_id,
			   item_type, item_id, thumbs_up, rating, comment,
			   metadata, user_agent, ip_address, created_at, updated_at
		FROM user_feedback
		WHERE id = $1`

	var feedback domain.UserFeedback
	err := pgxscan.Get(ctx, r.db, &feedback, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feedback not found")
		}
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	return &feedback, nil
}

// Update updates an existing feedback record
func (r *FeedbackRepository) Update(ctx context.Context, feedback *domain.UserFeedback) error {
	metadataJSON, err := json.Marshal(feedback.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE user_feedback
		SET thumbs_up = $1, rating = $2, comment = $3,
			metadata = $4, updated_at = $5
		WHERE id = $6`

	_, err = r.db.Exec(ctx, query,
		feedback.ThumbsUp, feedback.Rating, feedback.Comment,
		metadataJSON, time.Now(), feedback.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update feedback: %w", err)
	}

	return nil
}

// Delete deletes a feedback record
func (r *FeedbackRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM user_feedback WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feedback: %w", err)
	}

	return nil
}

// List retrieves feedback with filtering and pagination
func (r *FeedbackRepository) List(ctx context.Context, filter domain.FeedbackFilter) ([]*domain.UserFeedback, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.ProjectID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND project_id = $%d", argCount)
		args = append(args, filter.ProjectID)
	}

	if filter.UserID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, filter.UserID)
	}

	if filter.ItemType != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND item_type = $%d", argCount)
		args = append(args, filter.ItemType)
	}

	if filter.ItemID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND item_id = $%d", argCount)
		args = append(args, filter.ItemID)
	}

	if filter.TraceID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND trace_id = $%d", argCount)
		args = append(args, filter.TraceID)
	}

	if filter.SessionID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND session_id = $%d", argCount)
		args = append(args, filter.SessionID)
	}

	if filter.ThumbsUp != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND thumbs_up = $%d", argCount)
		args = append(args, *filter.ThumbsUp)
	}

	if filter.Rating != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND rating = $%d", argCount)
		args = append(args, *filter.Rating)
	}

	if !filter.StartDate.IsZero() {
		argCount++
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		argCount++
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_feedback %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count feedback: %w", err)
	}

	// Data query with ordering
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
		if filter.OrderDesc {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}
	}

	limit := 50
	if filter.Limit > 0 && filter.Limit <= 1000 {
		limit = filter.Limit
	}

	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT id, project_id, user_id, session_id, trace_id, span_id,
			   item_type, item_id, thumbs_up, rating, comment,
			   metadata, user_agent, ip_address, created_at, updated_at
		FROM user_feedback %s
		ORDER BY %s
		LIMIT %d OFFSET %d`, whereClause, orderBy, limit, offset)

	var feedback []*domain.UserFeedback
	err = pgxscan.Select(ctx, r.db, &feedback, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list feedback: %w", err)
	}

	return feedback, total, nil
}

// GetAnalytics retrieves aggregated feedback analytics
func (r *FeedbackRepository) GetAnalytics(ctx context.Context, projectID string, itemType string, startDate, endDate time.Time) (*domain.FeedbackAnalytics, error) {
	query := `
		WITH feedback_stats AS (
			SELECT
				COUNT(*) as total_feedback,
				COUNT(CASE WHEN thumbs_up = true THEN 1 END) as thumbs_up_count,
				COUNT(CASE WHEN thumbs_up = false THEN 1 END) as thumbs_down_count,
				ROUND(AVG(CASE WHEN rating IS NOT NULL THEN rating END), 2) as average_rating,
				COUNT(CASE WHEN comment IS NOT NULL AND comment != '' THEN 1 END) as comment_count
			FROM user_feedback
			WHERE project_id = $1
				AND item_type = $2
				AND created_at >= $3
				AND created_at <= $4
		),
		rating_counts AS (
			SELECT
				rating,
				COUNT(*) as count
			FROM user_feedback
			WHERE project_id = $1
				AND item_type = $2
				AND created_at >= $3
				AND created_at <= $4
				AND rating IS NOT NULL
			GROUP BY rating
		)
		SELECT
			fs.total_feedback,
			fs.thumbs_up_count,
			fs.thumbs_down_count,
			fs.average_rating,
			fs.comment_count,
			json_object_agg(COALESCE(rc.rating::text, '0'), COALESCE(rc.count, 0)) as rating_counts
		FROM feedback_stats fs
		LEFT JOIN rating_counts rc ON true
		GROUP BY fs.total_feedback, fs.thumbs_up_count, fs.thumbs_down_count, fs.average_rating, fs.comment_count`

	var analytics domain.FeedbackAnalytics
	analytics.ProjectID = domain.MustParseUUID(projectID)
	analytics.ItemType = itemType
	analytics.DateRange = fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var ratingCountsJSON []byte
	err := r.db.QueryRow(ctx, query, projectID, itemType, startDate, endDate).Scan(
		&analytics.TotalFeedback,
		&analytics.ThumbsUpCount,
		&analytics.ThumbsDownCount,
		&analytics.AverageRating,
		&analytics.CommentCount,
		&ratingCountsJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	// Parse rating counts JSON
	if ratingCountsJSON != nil {
		ratingCounts := make(map[string]int64)
		if err := json.Unmarshal(ratingCountsJSON, &ratingCounts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rating counts: %w", err)
		}

		analytics.RatingCounts = make(map[int]int64)
		for ratingStr, count := range ratingCounts {
			if ratingStr != "0" {
				var rating int
				fmt.Sscanf(ratingStr, "%d", &rating)
				analytics.RatingCounts[rating] = count
			}
		}
	}

	return &analytics, nil
}

// GetTrends retrieves feedback trends over time
func (r *FeedbackRepository) GetTrends(ctx context.Context, projectID string, itemType string, startDate, endDate time.Time, interval string) ([]*domain.FeedbackTrend, error) {
	var dateFormat string
	switch interval {
	case "hour":
		dateFormat = "YYYY-MM-DD HH24:00:00"
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "YYYY-MM-DD"
	case "month":
		dateFormat = "YYYY-MM-01"
	default:
		dateFormat = "YYYY-MM-DD"
	}

	query := fmt.Sprintf(`
		WITH daily_stats AS (
			SELECT
				to_char(date_trunc('%s', created_at), '%s') as date,
				COUNT(*) as total_feedback,
				ROUND(
					COUNT(CASE WHEN thumbs_up = true THEN 1 END)::decimal /
					GREATEST(COUNT(*), 1) * 100, 2
				) as thumbs_up_rate,
				ROUND(AVG(CASE WHEN rating IS NOT NULL THEN rating END), 2) as average_rating,
				COUNT(CASE WHEN comment IS NOT NULL AND comment != '' THEN 1 END) as comment_count
			FROM user_feedback
			WHERE project_id = $1
				AND item_type = $2
				AND created_at >= $3
				AND created_at <= $4
			GROUP BY date_trunc('%s', created_at)
			ORDER BY date_trunc('%s', created_at)
		)
		SELECT date, total_feedback, thumbs_up_rate, average_rating, comment_count
		FROM daily_stats`, interval, dateFormat, interval, interval)

	var trends []*domain.FeedbackTrend
	err := pgxscan.Select(ctx, r.db, &trends, query, projectID, itemType, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}

	return trends, nil
}
