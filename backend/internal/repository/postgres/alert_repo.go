package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// AlertRepository handles alert data access
type AlertRepository struct {
	db *pgxpool.Pool
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

// CreateAlertRule creates a new alert rule
func (r *AlertRepository) CreateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	query := `
		INSERT INTO alert_rules (
			id, project_id, name, description, enabled,
			metric_type, metric_field, condition_type, operator, threshold_value,
			window_duration, evaluation_frequency, filters,
			notification_channels, notification_message, escalation_policy_id,
			group_by, group_wait, repeat_interval, severity, tags,
			created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`
	_, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.ProjectID,
		rule.Name,
		rule.Description,
		rule.Enabled,
		rule.MetricType,
		rule.MetricField,
		rule.ConditionType,
		rule.Operator,
		rule.ThresholdValue,
		rule.WindowDuration,
		rule.EvaluationFrequency,
		rule.Filters,
		rule.NotificationChannels,
		rule.NotificationMessage,
		rule.EscalationPolicyID,
		rule.GroupBy,
		rule.GroupWait,
		rule.RepeatInterval,
		rule.Severity,
		rule.Tags,
		rule.CreatedAt,
		rule.UpdatedAt,
		rule.CreatedBy,
	)
	return err
}

// GetAlertRuleByID retrieves an alert rule by ID
func (r *AlertRepository) GetAlertRuleByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	var rule domain.AlertRule
	query := `
		SELECT id, project_id, name, description, enabled,
			metric_type, metric_field, condition_type, operator, threshold_value,
			window_duration, evaluation_frequency, filters,
			notification_channels, notification_message, escalation_policy_id,
			group_by, group_wait, repeat_interval, severity, tags,
			created_at, updated_at, created_by
		FROM alert_rules
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &rule, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &rule, nil
}

// ListAlertRules returns alert rules for a project
func (r *AlertRepository) ListAlertRules(ctx context.Context, projectID uuid.UUID, opts *ListOptions) ([]*domain.AlertRule, int, error) {
	var rules []*domain.AlertRule
	var total int

	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM alert_rules WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, project_id, name, description, enabled,
			metric_type, metric_field, condition_type, operator, threshold_value,
			window_duration, evaluation_frequency, filters,
			notification_channels, notification_message, escalation_policy_id,
			group_by, group_wait, repeat_interval, severity, tags,
			created_at, updated_at, created_by
		FROM alert_rules
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &rules, listQuery, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListEnabledAlertRules returns all enabled alert rules for a project
func (r *AlertRepository) ListEnabledAlertRules(ctx context.Context, projectID uuid.UUID) ([]*domain.AlertRule, error) {
	var rules []*domain.AlertRule
	query := `
		SELECT id, project_id, name, description, enabled,
			metric_type, metric_field, condition_type, operator, threshold_value,
			window_duration, evaluation_frequency, filters,
			notification_channels, notification_message, escalation_policy_id,
			group_by, group_wait, repeat_interval, severity, tags,
			created_at, updated_at, created_by
		FROM alert_rules
		WHERE project_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`
	if err := pgxscan.Select(ctx, r.db, &rules, query, projectID); err != nil {
		return nil, err
	}

	return rules, nil
}

// UpdateAlertRule updates an alert rule
func (r *AlertRepository) UpdateAlertRule(ctx context.Context, rule *domain.AlertRule) error {
	query := `
		UPDATE alert_rules
		SET name = $2, description = $3, enabled = $4,
			metric_type = $5, metric_field = $6, condition_type = $7, operator = $8, threshold_value = $9,
			window_duration = $10, evaluation_frequency = $11, filters = $12,
			notification_channels = $13, notification_message = $14, escalation_policy_id = $15,
			group_by = $16, group_wait = $17, repeat_interval = $18, severity = $19, tags = $20,
			updated_at = $21
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Enabled,
		rule.MetricType,
		rule.MetricField,
		rule.ConditionType,
		rule.Operator,
		rule.ThresholdValue,
		rule.WindowDuration,
		rule.EvaluationFrequency,
		rule.Filters,
		rule.NotificationChannels,
		rule.NotificationMessage,
		rule.EscalationPolicyID,
		rule.GroupBy,
		rule.GroupWait,
		rule.RepeatInterval,
		rule.Severity,
		rule.Tags,
		time.Now(),
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteAlertRule deletes an alert rule
func (r *AlertRepository) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM alert_rules WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// CreateAlertHistory creates a new alert history entry
func (r *AlertRepository) CreateAlertHistory(ctx context.Context, history *domain.AlertHistory) error {
	query := `
		INSERT INTO alert_history (
			id, alert_rule_id, project_id, status, severity,
			metric_value, threshold_value, fired_at, resolved_at, acknowledged_at, acknowledged_by,
			fingerprint, group_labels, notification_sent, notification_channels, notification_error,
			message, annotations, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`
	_, err := r.db.Exec(ctx, query,
		history.ID,
		history.AlertRuleID,
		history.ProjectID,
		history.Status,
		history.Severity,
		history.MetricValue,
		history.ThresholdValue,
		history.FiredAt,
		history.ResolvedAt,
		history.AcknowledgedAt,
		history.AcknowledgedBy,
		history.Fingerprint,
		history.GroupLabels,
		history.NotificationSent,
		history.NotificationChannels,
		history.NotificationError,
		history.Message,
		history.Annotations,
		history.CreatedAt,
		history.UpdatedAt,
	)
	return err
}

// GetAlertHistoryByID retrieves alert history by ID
func (r *AlertRepository) GetAlertHistoryByID(ctx context.Context, id uuid.UUID) (*domain.AlertHistory, error) {
	var history domain.AlertHistory
	query := `
		SELECT id, alert_rule_id, project_id, status, severity,
			metric_value, threshold_value, fired_at, resolved_at, acknowledged_at, acknowledged_by,
			fingerprint, group_labels, notification_sent, notification_channels, notification_error,
			message, annotations, created_at, updated_at
		FROM alert_history
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &history, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &history, nil
}

// GetAlertHistoryByFingerprint retrieves the latest alert history by fingerprint
func (r *AlertRepository) GetAlertHistoryByFingerprint(ctx context.Context, fingerprint string) (*domain.AlertHistory, error) {
	var history domain.AlertHistory
	query := `
		SELECT id, alert_rule_id, project_id, status, severity,
			metric_value, threshold_value, fired_at, resolved_at, acknowledged_at, acknowledged_by,
			fingerprint, group_labels, notification_sent, notification_channels, notification_error,
			message, annotations, created_at, updated_at
		FROM alert_history
		WHERE fingerprint = $1
		ORDER BY fired_at DESC
		LIMIT 1
	`
	err := pgxscan.Get(ctx, r.db, &history, query, fingerprint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &history, nil
}

// ListAlertHistory returns alert history for a project
func (r *AlertRepository) ListAlertHistory(ctx context.Context, projectID uuid.UUID, opts *ListOptions) ([]*domain.AlertHistory, int, error) {
	var history []*domain.AlertHistory
	var total int

	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM alert_history WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, alert_rule_id, project_id, status, severity,
			metric_value, threshold_value, fired_at, resolved_at, acknowledged_at, acknowledged_by,
			fingerprint, group_labels, notification_sent, notification_channels, notification_error,
			message, annotations, created_at, updated_at
		FROM alert_history
		WHERE project_id = $1
		ORDER BY fired_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &history, listQuery, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return history, total, nil
}

// UpdateAlertHistory updates alert history
func (r *AlertRepository) UpdateAlertHistory(ctx context.Context, history *domain.AlertHistory) error {
	query := `
		UPDATE alert_history
		SET status = $2, resolved_at = $3, acknowledged_at = $4, acknowledged_by = $5,
			notification_sent = $6, notification_error = $7, updated_at = $8
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query,
		history.ID,
		history.Status,
		history.ResolvedAt,
		history.AcknowledgedAt,
		history.AcknowledgedBy,
		history.NotificationSent,
		history.NotificationError,
		time.Now(),
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (r *AlertRepository) AcknowledgeAlert(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE alert_history
		SET status = 'acknowledged', acknowledged_at = $2, acknowledged_by = $3, updated_at = $2
		WHERE id = $1 AND status = 'firing'
	`
	result, err := r.db.Exec(ctx, query, id, time.Now(), userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ResolveAlert resolves an alert
func (r *AlertRepository) ResolveAlert(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE alert_history
		SET status = 'resolved', resolved_at = $2, updated_at = $2
		WHERE id = $1 AND status IN ('firing', 'acknowledged')
	`
	result, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CreateNotificationLog creates a notification log entry
func (r *AlertRepository) CreateNotificationLog(ctx context.Context, log *domain.AlertNotificationLog) error {
	query := `
		INSERT INTO alert_notification_log (id, alert_history_id, channel_type, channel_target, status, error_message, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		log.ID,
		log.AlertHistoryID,
		log.ChannelType,
		log.ChannelTarget,
		log.Status,
		log.ErrorMessage,
		log.SentAt,
	)
	return err
}

// ListNotificationLogs returns notification logs for an alert
func (r *AlertRepository) ListNotificationLogs(ctx context.Context, alertHistoryID uuid.UUID) ([]*domain.AlertNotificationLog, error) {
	var logs []*domain.AlertNotificationLog
	query := `
		SELECT id, alert_history_id, channel_type, channel_target, status, error_message, sent_at
		FROM alert_notification_log
		WHERE alert_history_id = $1
		ORDER BY sent_at DESC
	`
	if err := pgxscan.Select(ctx, r.db, &logs, query, alertHistoryID); err != nil {
		return nil, err
	}

	return logs, nil
}

// CreateEscalationPolicy creates a new escalation policy
func (r *AlertRepository) CreateEscalationPolicy(ctx context.Context, policy *domain.AlertEscalationPolicy) error {
	query := `
		INSERT INTO alert_escalation_policies (id, project_id, name, description, steps, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		policy.ID,
		policy.ProjectID,
		policy.Name,
		policy.Description,
		policy.Steps,
		policy.CreatedAt,
		policy.UpdatedAt,
	)
	return err
}

// GetEscalationPolicyByID retrieves an escalation policy by ID
func (r *AlertRepository) GetEscalationPolicyByID(ctx context.Context, id uuid.UUID) (*domain.AlertEscalationPolicy, error) {
	var policy domain.AlertEscalationPolicy
	query := `
		SELECT id, project_id, name, description, steps, created_at, updated_at
		FROM alert_escalation_policies
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &policy, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &policy, nil
}

// ListEscalationPolicies returns escalation policies for a project
func (r *AlertRepository) ListEscalationPolicies(ctx context.Context, projectID uuid.UUID, opts *ListOptions) ([]*domain.AlertEscalationPolicy, int, error) {
	var policies []*domain.AlertEscalationPolicy
	var total int

	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM alert_escalation_policies WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, project_id, name, description, steps, created_at, updated_at
		FROM alert_escalation_policies
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &policies, listQuery, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return policies, total, nil
}

// UpdateEscalationPolicy updates an escalation policy
func (r *AlertRepository) UpdateEscalationPolicy(ctx context.Context, policy *domain.AlertEscalationPolicy) error {
	query := `
		UPDATE alert_escalation_policies
		SET name = $2, description = $3, steps = $4, updated_at = $5
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Steps,
		time.Now(),
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteEscalationPolicy deletes an escalation policy
func (r *AlertRepository) DeleteEscalationPolicy(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM alert_escalation_policies WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
