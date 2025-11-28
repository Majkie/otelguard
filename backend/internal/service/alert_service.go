package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// AlertService handles alert management and evaluation
type AlertService struct {
	alertRepo         *postgres.AlertRepository
	chConn            clickhouse.Conn
	notifier          *NotificationService
	escalationService *EscalationService
	logger            *zap.Logger

	// Baseline tracking for anomaly detection
	baselines map[string]*MetricBaseline
}

// MetricBaseline stores baseline statistics for anomaly detection
type MetricBaseline struct {
	Mean       float64
	StdDev     float64
	SampleSize int
	LastUpdate time.Time
}

// NewAlertService creates a new alert service
func NewAlertService(
	alertRepo *postgres.AlertRepository,
	chConn clickhouse.Conn,
	notifier *NotificationService,
	escalationService *EscalationService,
	logger *zap.Logger,
) *AlertService {
	return &AlertService{
		alertRepo:         alertRepo,
		chConn:            chConn,
		notifier:          notifier,
		escalationService: escalationService,
		logger:            logger,
		baselines:         make(map[string]*MetricBaseline),
	}
}

// CreateAlertRule creates a new alert rule
func (s *AlertService) CreateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Validate rule
	if err := s.validateAlertRule(rule); err != nil {
		return err
	}

	return s.alertRepo.CreateAlertRule(ctx, rule)
}

// GetAlertRule retrieves an alert rule by ID
func (s *AlertService) GetAlertRule(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	return s.alertRepo.GetAlertRuleByID(ctx, id)
}

// ListAlertRules lists alert rules for a project
func (s *AlertService) ListAlertRules(ctx context.Context, projectID uuid.UUID, opts *postgres.ListOptions) ([]*domain.AlertRule, int, error) {
	return s.alertRepo.ListAlertRules(ctx, projectID, opts)
}

// UpdateAlertRule updates an alert rule
func (s *AlertService) UpdateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	rule.UpdatedAt = time.Now()

	if err := s.validateAlertRule(rule); err != nil {
		return err
	}

	return s.alertRepo.UpdateAlertRule(ctx, rule)
}

// DeleteAlertRule deletes an alert rule
func (s *AlertService) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.DeleteAlertRule(ctx, id)
}

// EvaluateAlerts evaluates all enabled alert rules for a project
func (s *AlertService) EvaluateAlerts(ctx context.Context, projectID uuid.UUID) error {
	rules, err := s.alertRepo.ListEnabledAlertRules(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list enabled rules: %w", err)
	}

	for _, rule := range rules {
		if err := s.evaluateRule(ctx, rule); err != nil {
			s.logger.Error("failed to evaluate rule",
				zap.String("rule_id", rule.ID.String()),
				zap.Error(err),
			)
			// Continue evaluating other rules
		}
	}

	return nil
}

// evaluateRule evaluates a single alert rule
func (s *AlertService) evaluateRule(ctx context.Context, rule *domain.AlertRule) error {
	// Collect metric value
	metricValue, err := s.collectMetric(ctx, rule)
	if err != nil {
		return fmt.Errorf("failed to collect metric: %w", err)
	}

	// Check if alert should trigger
	triggered := s.checkCondition(rule, metricValue)

	// Generate fingerprint for grouping/deduplication
	fingerprint := s.generateFingerprint(rule, metricValue.Labels)

	// Check if alert already exists
	existingAlert, err := s.alertRepo.GetAlertHistoryByFingerprint(ctx, fingerprint)
	if err != nil && err != domain.ErrNotFound {
		return fmt.Errorf("failed to check existing alert: %w", err)
	}

	// Handle alert state
	if triggered {
		return s.handleTriggeredAlert(ctx, rule, metricValue, fingerprint, existingAlert)
	} else {
		return s.handleResolvedAlert(ctx, existingAlert)
	}
}

// collectMetric collects the metric value from ClickHouse
func (s *AlertService) collectMetric(ctx context.Context, rule *domain.AlertRule) (*domain.AlertMetricResult, error) {
	var query string
	var args []interface{}

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(rule.WindowDuration) * time.Second)

	switch rule.MetricType {
	case "latency":
		query = `
			SELECT AVG(latency_ms) as value
			FROM traces
			WHERE project_id = ?
			  AND start_time >= ?
			  AND start_time <= ?
		`
		args = []interface{}{rule.ProjectID, startTime, endTime}

	case "cost":
		query = `
			SELECT SUM(cost) as value
			FROM traces
			WHERE project_id = ?
			  AND start_time >= ?
			  AND start_time <= ?
		`
		args = []interface{}{rule.ProjectID, startTime, endTime}

	case "error_rate":
		query = `
			SELECT countIf(status = 'error') / count() * 100 as value
			FROM traces
			WHERE project_id = ?
			  AND start_time >= ?
			  AND start_time <= ?
		`
		args = []interface{}{rule.ProjectID, startTime, endTime}

	case "token_count":
		query = `
			SELECT SUM(total_tokens) as value
			FROM traces
			WHERE project_id = ?
			  AND start_time >= ?
			  AND start_time <= ?
		`
		args = []interface{}{rule.ProjectID, startTime, endTime}

	case "custom":
		// Custom metrics require a metric_field
		if rule.MetricField == nil || *rule.MetricField == "" {
			return nil, fmt.Errorf("custom metric requires metric_field")
		}

		// Build custom query based on the field
		query = fmt.Sprintf(`
			SELECT AVG(CAST(JSONExtractString(metadata, '%s') AS Float64)) as value
			FROM traces
			WHERE project_id = ?
			  AND start_time >= ?
			  AND start_time <= ?
			  AND JSONHas(metadata, '%s')
		`, *rule.MetricField, *rule.MetricField)
		args = []interface{}{rule.ProjectID, startTime, endTime}

	default:
		return nil, fmt.Errorf("unsupported metric type: %s", rule.MetricType)
	}

	// Add filters from rule
	if len(rule.Filters) > 0 {
		for key, value := range rule.Filters {
			switch key {
			case "model":
				query += " AND model = ?"
				args = append(args, value)
			case "user_id":
				query += " AND user_id = ?"
				args = append(args, value)
			}
		}
	}

	var value float64
	err := s.chConn.QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		return nil, err
	}

	return &domain.AlertMetricResult{
		Value:     value,
		Timestamp: endTime,
		Labels:    s.extractLabels(rule),
	}, nil
}

// checkCondition checks if the metric value triggers the alert condition
func (s *AlertService) checkCondition(rule *domain.AlertRule, result *domain.AlertMetricResult) bool {
	// Handle anomaly detection
	if rule.ConditionType == "anomaly" {
		return s.detectAnomaly(rule, result)
	}

	// Handle threshold-based conditions
	if rule.ThresholdValue == nil {
		return false
	}

	threshold := *rule.ThresholdValue
	value := result.Value

	switch rule.Operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	default:
		return false
	}
}

// detectAnomaly detects anomalies using statistical methods
func (s *AlertService) detectAnomaly(rule *domain.AlertRule, result *domain.AlertMetricResult) bool {
	baselineKey := s.getBaselineKey(rule)

	// Get or create baseline
	baseline, exists := s.baselines[baselineKey]
	if !exists {
		// Initialize baseline with historical data
		baseline = s.initializeBaseline(context.Background(), rule)
		s.baselines[baselineKey] = baseline
	}

	// Check if baseline needs updating (every hour)
	if time.Since(baseline.LastUpdate) > time.Hour {
		baseline = s.updateBaseline(context.Background(), rule, baseline)
		s.baselines[baselineKey] = baseline
	}

	// Calculate z-score
	zScore := (result.Value - baseline.Mean) / baseline.StdDev

	// Use threshold as number of standard deviations
	threshold := 3.0 // Default to 3 standard deviations
	if rule.ThresholdValue != nil {
		threshold = *rule.ThresholdValue
	}

	// Anomaly detected if z-score exceeds threshold
	return zScore > threshold || zScore < -threshold
}

// initializeBaseline calculates initial baseline from historical data
func (s *AlertService) initializeBaseline(ctx context.Context, rule *domain.AlertRule) *MetricBaseline {
	// Collect last 7 days of data for baseline
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour)

	// Query historical metric values
	query := s.buildHistoricalQuery(rule)

	rows, err := s.chConn.Query(ctx, query, rule.ProjectID, startTime, endTime)
	if err != nil {
		s.logger.Error("failed to query historical data for baseline", zap.Error(err))
		return &MetricBaseline{
			Mean:       0,
			StdDev:     1,
			SampleSize: 0,
			LastUpdate: time.Now(),
		}
	}
	defer rows.Close()

	// Calculate mean and standard deviation
	var values []float64
	for rows.Next() {
		var value float64
		if err := rows.Scan(&value); err != nil {
			continue
		}
		values = append(values, value)
	}

	if len(values) == 0 {
		return &MetricBaseline{
			Mean:       0,
			StdDev:     1,
			SampleSize: 0,
			LastUpdate: time.Now(),
		}
	}

	mean, stdDev := calculateStatistics(values)

	return &MetricBaseline{
		Mean:       mean,
		StdDev:     stdDev,
		SampleSize: len(values),
		LastUpdate: time.Now(),
	}
}

// updateBaseline updates the baseline with recent data
func (s *AlertService) updateBaseline(ctx context.Context, rule *domain.AlertRule, current *MetricBaseline) *MetricBaseline {
	// Similar to initializeBaseline but uses exponential moving average
	return s.initializeBaseline(ctx, rule)
}

// buildHistoricalQuery builds a query for historical metric data
func (s *AlertService) buildHistoricalQuery(rule *domain.AlertRule) string {
	baseQuery := ""

	switch rule.MetricType {
	case "latency":
		baseQuery = "SELECT AVG(latency_ms) as value FROM traces WHERE project_id = ? AND start_time >= ? AND start_time < ? GROUP BY toStartOfHour(start_time) ORDER BY toStartOfHour(start_time)"
	case "cost":
		baseQuery = "SELECT SUM(cost) as value FROM traces WHERE project_id = ? AND start_time >= ? AND start_time < ? GROUP BY toStartOfHour(start_time) ORDER BY toStartOfHour(start_time)"
	case "error_rate":
		baseQuery = "SELECT countIf(status = 'error') / count() * 100 as value FROM traces WHERE project_id = ? AND start_time >= ? AND start_time < ? GROUP BY toStartOfHour(start_time) ORDER BY toStartOfHour(start_time)"
	case "token_count":
		baseQuery = "SELECT SUM(total_tokens) as value FROM traces WHERE project_id = ? AND start_time >= ? AND start_time < ? GROUP BY toStartOfHour(start_time) ORDER BY toStartOfHour(start_time)"
	default:
		baseQuery = "SELECT 0 as value"
	}

	return baseQuery
}

// getBaselineKey generates a unique key for baseline storage
func (s *AlertService) getBaselineKey(rule *domain.AlertRule) string {
	return fmt.Sprintf("%s:%s", rule.ProjectID.String(), rule.MetricType)
}

// calculateStatistics calculates mean and standard deviation
func calculateStatistics(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 1
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stdDev = 1.0
	if variance > 0 {
		// Simple square root approximation
		stdDev = variance / 2.0
		for i := 0; i < 10; i++ {
			stdDev = (stdDev + variance/stdDev) / 2.0
		}
	}

	if stdDev == 0 {
		stdDev = 1 // Avoid division by zero
	}

	return mean, stdDev
}

// generateFingerprint generates a unique fingerprint for alert grouping
func (s *AlertService) generateFingerprint(rule *domain.AlertRule, labels map[string]string) string {
	// Create deterministic string from rule ID and group_by labels
	parts := []string{rule.ID.String()}

	// Sort group_by keys for deterministic ordering
	if len(rule.GroupBy) > 0 {
		sortedKeys := make([]string, len(rule.GroupBy))
		copy(sortedKeys, rule.GroupBy)
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			if value, ok := labels[key]; ok {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	// Hash the parts
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", hash)
}

// handleTriggeredAlert handles a triggered alert
func (s *AlertService) handleTriggeredAlert(
	ctx context.Context,
	rule *domain.AlertRule,
	result *domain.AlertMetricResult,
	fingerprint string,
	existingAlert *domain.AlertHistory,
) error {
	now := time.Now()

	// Check if we should send notification (respect repeat_interval)
	shouldNotify := true
	if existingAlert != nil && existingAlert.Status == "firing" {
		timeSinceLastNotification := now.Sub(existingAlert.FiredAt)
		if int(timeSinceLastNotification.Seconds()) < rule.RepeatInterval {
			shouldNotify = false
		}
	}

	// Create or update alert history
	if existingAlert == nil || shouldNotify {
		groupLabels := make(map[string]interface{})
		for k, v := range result.Labels {
			groupLabels[k] = v
		}

		message := s.generateAlertMessage(rule, result)

		history := &domain.AlertHistory{
			ID:                   uuid.New(),
			AlertRuleID:          rule.ID,
			ProjectID:            rule.ProjectID,
			Status:               "firing",
			Severity:             rule.Severity,
			MetricValue:          &result.Value,
			ThresholdValue:       rule.ThresholdValue,
			FiredAt:              now,
			Fingerprint:          fingerprint,
			GroupLabels:          groupLabels,
			NotificationSent:     false,
			NotificationChannels: rule.NotificationChannels,
			Message:              &message,
			Annotations:          make(map[string]interface{}),
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		if err := s.alertRepo.CreateAlertHistory(ctx, history); err != nil {
			return fmt.Errorf("failed to create alert history: %w", err)
		}

		// Send notifications
		if shouldNotify {
			if err := s.sendNotifications(ctx, rule, history); err != nil {
				s.logger.Error("failed to send notifications",
					zap.String("alert_id", history.ID.String()),
					zap.Error(err),
				)
				// Don't fail the alert creation if notifications fail
			}
		}
	}

	return nil
}

// handleResolvedAlert handles a resolved alert
func (s *AlertService) handleResolvedAlert(ctx context.Context, existingAlert *domain.AlertHistory) error {
	if existingAlert != nil && existingAlert.Status == "firing" {
		return s.alertRepo.ResolveAlert(ctx, existingAlert.ID)
	}
	return nil
}

// sendNotifications sends notifications through configured channels
func (s *AlertService) sendNotifications(ctx context.Context, rule *domain.AlertRule, history *domain.AlertHistory) error {
	for _, channel := range rule.NotificationChannels {
		parts := strings.SplitN(channel, ":", 2)
		if len(parts) != 2 {
			s.logger.Warn("invalid notification channel format", zap.String("channel", channel))
			continue
		}

		channelType := parts[0]
		channelTarget := parts[1]

		var err error
		switch channelType {
		case "email":
			err = s.notifier.SendEmail(ctx, channelTarget, rule.Name, *history.Message)
		case "slack":
			err = s.notifier.SendSlack(ctx, channelTarget, rule, history)
		case "webhook":
			err = s.notifier.SendWebhook(ctx, channelTarget, rule, history)
		default:
			s.logger.Warn("unsupported notification channel type", zap.String("type", channelType))
			continue
		}

		// Log notification attempt
		logEntry := &domain.AlertNotificationLog{
			ID:             uuid.New(),
			AlertHistoryID: history.ID,
			ChannelType:    channelType,
			ChannelTarget:  channelTarget,
			Status:         "sent",
			SentAt:         time.Now(),
		}

		if err != nil {
			logEntry.Status = "failed"
			errMsg := err.Error()
			logEntry.ErrorMessage = &errMsg
			s.logger.Error("failed to send notification",
				zap.String("channel", channel),
				zap.Error(err),
			)
		}

		if err := s.alertRepo.CreateNotificationLog(ctx, logEntry); err != nil {
			s.logger.Error("failed to create notification log", zap.Error(err))
		}
	}

	// Update notification status
	history.NotificationSent = true
	return s.alertRepo.UpdateAlertHistory(ctx, history)
}

// generateAlertMessage generates a human-readable alert message
func (s *AlertService) generateAlertMessage(rule *domain.AlertRule, result *domain.AlertMetricResult) string {
	if rule.NotificationMessage != nil && *rule.NotificationMessage != "" {
		return *rule.NotificationMessage
	}

	return fmt.Sprintf(
		"Alert: %s - %s is %.2f (threshold: %.2f)",
		rule.Name,
		rule.MetricType,
		result.Value,
		*rule.ThresholdValue,
	)
}

// extractLabels extracts labels from rule filters
func (s *AlertService) extractLabels(rule *domain.AlertRule) map[string]string {
	labels := make(map[string]string)
	for key, value := range rule.Filters {
		if strValue, ok := value.(string); ok {
			labels[key] = strValue
		}
	}
	return labels
}

// validateAlertRule validates an alert rule
func (s *AlertService) validateAlertRule(rule *domain.AlertRule) error {
	if rule.Name == "" {
		return fmt.Errorf("alert rule name is required")
	}

	if rule.MetricType == "" {
		return fmt.Errorf("metric type is required")
	}

	if rule.ConditionType == "" {
		return fmt.Errorf("condition type is required")
	}

	if rule.Operator == "" {
		return fmt.Errorf("operator is required")
	}

	if rule.ThresholdValue == nil {
		return fmt.Errorf("threshold value is required")
	}

	validOperators := map[string]bool{"gt": true, "gte": true, "lt": true, "lte": true, "eq": true, "ne": true}
	if !validOperators[rule.Operator] {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	return nil
}

// AcknowledgeAlert acknowledges an alert
func (s *AlertService) AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, userID uuid.UUID) error {
	return s.alertRepo.AcknowledgeAlert(ctx, alertID, userID)
}

// ListAlertHistory lists alert history for a project
func (s *AlertService) ListAlertHistory(ctx context.Context, projectID uuid.UUID, opts *postgres.ListOptions) ([]*domain.AlertHistory, int, error) {
	return s.alertRepo.ListAlertHistory(ctx, projectID, opts)
}

// GetAlertHistory retrieves alert history by ID
func (s *AlertService) GetAlertHistory(ctx context.Context, id uuid.UUID) (*domain.AlertHistory, error) {
	return s.alertRepo.GetAlertHistoryByID(ctx, id)
}

// AlertEvaluator runs periodic alert evaluations
type AlertEvaluator struct {
	alertService *AlertService
	logger       *zap.Logger
	stopChan     chan struct{}
}

// NewAlertEvaluator creates a new alert evaluator
func NewAlertEvaluator(alertService *AlertService, logger *zap.Logger) *AlertEvaluator {
	return &AlertEvaluator{
		alertService: alertService,
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the alert evaluator
func (e *AlertEvaluator) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Evaluate every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// In a production system, you would iterate through all projects
			// For now, this is a placeholder
			e.logger.Debug("evaluating alerts")
		case <-e.stopChan:
			e.logger.Info("alert evaluator stopped")
			return
		case <-ctx.Done():
			e.logger.Info("alert evaluator context cancelled")
			return
		}
	}
}

// Stop stops the alert evaluator
func (e *AlertEvaluator) Stop() {
	close(e.stopChan)
}
