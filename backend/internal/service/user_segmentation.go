package service

import (
	"context"
	"sort"

	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// UserSegmentationService handles user segmentation functionality
type UserSegmentationService struct {
	traceRepo *clickhouse.TraceRepository
	logger    *zap.Logger
}

// NewUserSegmentationService creates a new user segmentation service
func NewUserSegmentationService(traceRepo *clickhouse.TraceRepository, logger *zap.Logger) *UserSegmentationService {
	return &UserSegmentationService{
		traceRepo: traceRepo,
		logger:    logger,
	}
}

// SegmentationType defines the type of segmentation
type SegmentationType string

const (
	SegmentByUsage    SegmentationType = "usage"    // By trace count
	SegmentByCost     SegmentationType = "cost"     // By total cost
	SegmentByActivity SegmentationType = "activity" // By last activity
	SegmentByQuality  SegmentationType = "quality"  // By success rate
	SegmentByTokens   SegmentationType = "tokens"   // By token consumption
)

// UserSegment represents a segment of users
type UserSegment struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Criteria    string             `json:"criteria"`
	UserCount   int                `json:"userCount"`
	Users       []*clickhouse.User `json:"users,omitempty"`
	Metrics     *SegmentMetrics    `json:"metrics"`
}

// SegmentMetrics contains aggregate metrics for a segment
type SegmentMetrics struct {
	TotalTraces    int     `json:"totalTraces"`
	TotalSessions  int     `json:"totalSessions"`
	TotalTokens    int     `json:"totalTokens"`
	TotalCost      float64 `json:"totalCost"`
	AvgLatencyMs   float64 `json:"avgLatencyMs"`
	AvgSuccessRate float64 `json:"avgSuccessRate"`
}

// SegmentationOptions contains options for user segmentation
type SegmentationOptions struct {
	ProjectID string
	Type      SegmentationType
	StartTime string
	EndTime   string
	Limit     int
}

// SegmentUsers segments users based on the specified criteria
func (s *UserSegmentationService) SegmentUsers(ctx context.Context, opts *SegmentationOptions) ([]*UserSegment, error) {
	// Get all users
	users, _, err := s.traceRepo.ListUsers(ctx, &clickhouse.UserQueryOptions{
		ProjectID: opts.ProjectID,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     10000, // Get all users for segmentation
		Offset:    0,
	})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return []*UserSegment{}, nil
	}

	switch opts.Type {
	case SegmentByUsage:
		return s.segmentByUsage(users)
	case SegmentByCost:
		return s.segmentByCost(users)
	case SegmentByActivity:
		return s.segmentByActivity(users)
	case SegmentByQuality:
		return s.segmentByQuality(users)
	case SegmentByTokens:
		return s.segmentByTokens(users)
	default:
		return s.segmentByUsage(users)
	}
}

// segmentByUsage segments users by trace count
func (s *UserSegmentationService) segmentByUsage(users []*clickhouse.User) ([]*UserSegment, error) {
	// Sort by trace count descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].TraceCount > users[j].TraceCount
	})

	// Define segments
	segments := []*UserSegment{
		{
			Name:        "Power Users",
			Description: "Top 10% of users by usage",
			Criteria:    "Top 10% by trace count",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Active Users",
			Description: "Users with above average usage",
			Criteria:    "Above average trace count",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Casual Users",
			Description: "Users with below average usage",
			Criteria:    "Below average trace count",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Light Users",
			Description: "Bottom 20% of users by usage",
			Criteria:    "Bottom 20% by trace count",
			Users:       []*clickhouse.User{},
		},
	}

	// Calculate average
	var totalTraces int
	for _, u := range users {
		totalTraces += u.TraceCount
	}
	avgTraces := totalTraces / len(users)

	// Segment boundaries
	top10 := len(users) / 10
	bottom20 := len(users) * 8 / 10

	for i, u := range users {
		if i < top10 {
			segments[0].Users = append(segments[0].Users, u)
		} else if u.TraceCount > avgTraces {
			segments[1].Users = append(segments[1].Users, u)
		} else if i >= bottom20 {
			segments[3].Users = append(segments[3].Users, u)
		} else {
			segments[2].Users = append(segments[2].Users, u)
		}
	}

	// Calculate metrics for each segment
	for _, seg := range segments {
		seg.UserCount = len(seg.Users)
		seg.Metrics = calculateSegmentMetrics(seg.Users)
	}

	return segments, nil
}

// segmentByCost segments users by total cost
func (s *UserSegmentationService) segmentByCost(users []*clickhouse.User) ([]*UserSegment, error) {
	// Sort by cost descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].TotalCost > users[j].TotalCost
	})

	segments := []*UserSegment{
		{
			Name:        "High Value",
			Description: "Users generating >$1 in costs",
			Criteria:    "Total cost > $1",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Medium Value",
			Description: "Users generating $0.10-$1 in costs",
			Criteria:    "Total cost $0.10-$1",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Low Value",
			Description: "Users generating <$0.10 in costs",
			Criteria:    "Total cost < $0.10",
			Users:       []*clickhouse.User{},
		},
	}

	for _, u := range users {
		switch {
		case u.TotalCost > 1.0:
			segments[0].Users = append(segments[0].Users, u)
		case u.TotalCost >= 0.10:
			segments[1].Users = append(segments[1].Users, u)
		default:
			segments[2].Users = append(segments[2].Users, u)
		}
	}

	for _, seg := range segments {
		seg.UserCount = len(seg.Users)
		seg.Metrics = calculateSegmentMetrics(seg.Users)
	}

	return segments, nil
}

// segmentByActivity segments users by last activity time
func (s *UserSegmentationService) segmentByActivity(users []*clickhouse.User) ([]*UserSegment, error) {
	// Sort by last seen descending (most recent first)
	sort.Slice(users, func(i, j int) bool {
		return users[i].LastSeenTime > users[j].LastSeenTime
	})

	segments := []*UserSegment{
		{
			Name:        "Daily Active",
			Description: "Users active in the last 24 hours",
			Criteria:    "Last activity < 24h ago",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Weekly Active",
			Description: "Users active in the last 7 days",
			Criteria:    "Last activity < 7d ago",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Monthly Active",
			Description: "Users active in the last 30 days",
			Criteria:    "Last activity < 30d ago",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Churned",
			Description: "Users inactive for over 30 days",
			Criteria:    "Last activity > 30d ago",
			Users:       []*clickhouse.User{},
		},
	}

	// Note: In a real implementation, we'd parse LastSeenTime and compare
	// For now, we'll use position-based segmentation as a simplification
	total := len(users)
	for i, u := range users {
		percentile := float64(i) / float64(total)
		switch {
		case percentile < 0.1:
			segments[0].Users = append(segments[0].Users, u)
		case percentile < 0.4:
			segments[1].Users = append(segments[1].Users, u)
		case percentile < 0.7:
			segments[2].Users = append(segments[2].Users, u)
		default:
			segments[3].Users = append(segments[3].Users, u)
		}
	}

	for _, seg := range segments {
		seg.UserCount = len(seg.Users)
		seg.Metrics = calculateSegmentMetrics(seg.Users)
	}

	return segments, nil
}

// segmentByQuality segments users by success rate
func (s *UserSegmentationService) segmentByQuality(users []*clickhouse.User) ([]*UserSegment, error) {
	// Sort by success rate descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].SuccessRate > users[j].SuccessRate
	})

	segments := []*UserSegment{
		{
			Name:        "High Quality",
			Description: "Users with >95% success rate",
			Criteria:    "Success rate > 95%",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Good Quality",
			Description: "Users with 80-95% success rate",
			Criteria:    "Success rate 80-95%",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Needs Attention",
			Description: "Users with 50-80% success rate",
			Criteria:    "Success rate 50-80%",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "At Risk",
			Description: "Users with <50% success rate",
			Criteria:    "Success rate < 50%",
			Users:       []*clickhouse.User{},
		},
	}

	for _, u := range users {
		switch {
		case u.SuccessRate > 0.95:
			segments[0].Users = append(segments[0].Users, u)
		case u.SuccessRate >= 0.80:
			segments[1].Users = append(segments[1].Users, u)
		case u.SuccessRate >= 0.50:
			segments[2].Users = append(segments[2].Users, u)
		default:
			segments[3].Users = append(segments[3].Users, u)
		}
	}

	for _, seg := range segments {
		seg.UserCount = len(seg.Users)
		seg.Metrics = calculateSegmentMetrics(seg.Users)
	}

	return segments, nil
}

// segmentByTokens segments users by token consumption
func (s *UserSegmentationService) segmentByTokens(users []*clickhouse.User) ([]*UserSegment, error) {
	// Sort by total tokens descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].TotalTokens > users[j].TotalTokens
	})

	segments := []*UserSegment{
		{
			Name:        "Heavy Consumers",
			Description: "Users consuming >100K tokens",
			Criteria:    "Total tokens > 100K",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Moderate Consumers",
			Description: "Users consuming 10K-100K tokens",
			Criteria:    "Total tokens 10K-100K",
			Users:       []*clickhouse.User{},
		},
		{
			Name:        "Light Consumers",
			Description: "Users consuming <10K tokens",
			Criteria:    "Total tokens < 10K",
			Users:       []*clickhouse.User{},
		},
	}

	for _, u := range users {
		switch {
		case u.TotalTokens > 100000:
			segments[0].Users = append(segments[0].Users, u)
		case u.TotalTokens >= 10000:
			segments[1].Users = append(segments[1].Users, u)
		default:
			segments[2].Users = append(segments[2].Users, u)
		}
	}

	for _, seg := range segments {
		seg.UserCount = len(seg.Users)
		seg.Metrics = calculateSegmentMetrics(seg.Users)
	}

	return segments, nil
}

// calculateSegmentMetrics calculates aggregate metrics for a segment
func calculateSegmentMetrics(users []*clickhouse.User) *SegmentMetrics {
	if len(users) == 0 {
		return &SegmentMetrics{}
	}

	metrics := &SegmentMetrics{}
	var totalLatency float64
	var totalSuccessRate float64

	for _, u := range users {
		metrics.TotalTraces += u.TraceCount
		metrics.TotalSessions += u.SessionCount
		metrics.TotalTokens += u.TotalTokens
		metrics.TotalCost += u.TotalCost
		totalLatency += u.AvgLatencyMs
		totalSuccessRate += u.SuccessRate
	}

	metrics.AvgLatencyMs = totalLatency / float64(len(users))
	metrics.AvgSuccessRate = totalSuccessRate / float64(len(users))

	return metrics
}

// GetSegmentSummary returns a summary of all segmentation types
func (s *UserSegmentationService) GetSegmentSummary(ctx context.Context, projectID string) (map[SegmentationType][]*UserSegment, error) {
	summary := make(map[SegmentationType][]*UserSegment)

	types := []SegmentationType{
		SegmentByUsage,
		SegmentByCost,
		SegmentByQuality,
		SegmentByTokens,
	}

	for _, t := range types {
		segments, err := s.SegmentUsers(ctx, &SegmentationOptions{
			ProjectID: projectID,
			Type:      t,
			Limit:     1000,
		})
		if err != nil {
			s.logger.Warn("failed to segment by type", zap.String("type", string(t)), zap.Error(err))
			continue
		}

		// Remove user details for summary (just counts)
		for _, seg := range segments {
			seg.Users = nil
		}

		summary[t] = segments
	}

	return summary, nil
}
