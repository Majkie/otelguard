package domain

import (
	"time"

	"github.com/google/uuid"
)

// AlertRule represents a rule that triggers alerts based on metrics
type AlertRule struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ProjectID   uuid.UUID  `json:"project_id" db:"project_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	Enabled     bool       `json:"enabled" db:"enabled"`

	// Metric to monitor
	MetricType  string  `json:"metric_type" db:"metric_type"`   // 'latency', 'cost', 'error_rate', 'token_count', 'custom'
	MetricField *string `json:"metric_field,omitempty" db:"metric_field"` // For custom metrics

	// Condition
	ConditionType  string   `json:"condition_type" db:"condition_type"`   // 'threshold', 'anomaly', 'percentage_change'
	Operator       string   `json:"operator" db:"operator"`               // 'gt', 'lt', 'gte', 'lte', 'eq', 'ne'
	ThresholdValue *float64 `json:"threshold_value,omitempty" db:"threshold_value"`

	// Time window
	WindowDuration      int `json:"window_duration" db:"window_duration"`           // seconds
	EvaluationFrequency int `json:"evaluation_frequency" db:"evaluation_frequency"` // seconds

	// Filters
	Filters map[string]interface{} `json:"filters" db:"filters"` // { "model": "gpt-4", "user_id": "abc", etc }

	// Notification settings
	NotificationChannels []string `json:"notification_channels" db:"notification_channels"` // ['email:user@example.com', 'slack:#alerts', etc]
	NotificationMessage  *string  `json:"notification_message,omitempty" db:"notification_message"`

	// Escalation
	EscalationPolicyID *uuid.UUID `json:"escalation_policy_id,omitempty" db:"escalation_policy_id"`

	// Grouping and deduplication
	GroupBy        []string `json:"group_by" db:"group_by"`               // ['model', 'user_id', etc]
	GroupWait      int      `json:"group_wait" db:"group_wait"`           // seconds
	RepeatInterval int      `json:"repeat_interval" db:"repeat_interval"` // seconds

	// Metadata
	Severity  string   `json:"severity" db:"severity"`
	Tags      []string `json:"tags" db:"tags"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// AlertHistory represents a fired alert instance
type AlertHistory struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	AlertRuleID   uuid.UUID  `json:"alert_rule_id" db:"alert_rule_id"`
	ProjectID     uuid.UUID  `json:"project_id" db:"project_id"`

	// Alert details
	Status   string `json:"status" db:"status"`     // 'firing', 'resolved', 'acknowledged'
	Severity string `json:"severity" db:"severity"` // 'info', 'warning', 'error', 'critical'

	// Values
	MetricValue    *float64 `json:"metric_value,omitempty" db:"metric_value"`
	ThresholdValue *float64 `json:"threshold_value,omitempty" db:"threshold_value"`

	// Time
	FiredAt         time.Time  `json:"fired_at" db:"fired_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy  *uuid.UUID `json:"acknowledged_by,omitempty" db:"acknowledged_by"`

	// Grouping
	Fingerprint string                 `json:"fingerprint" db:"fingerprint"` // Hash of alert_rule_id + group_by values
	GroupLabels map[string]interface{} `json:"group_labels" db:"group_labels"`

	// Notification
	NotificationSent     bool     `json:"notification_sent" db:"notification_sent"`
	NotificationChannels []string `json:"notification_channels" db:"notification_channels"`
	NotificationError    *string  `json:"notification_error,omitempty" db:"notification_error"`

	// Additional context
	Message     *string                `json:"message,omitempty" db:"message"`
	Annotations map[string]interface{} `json:"annotations" db:"annotations"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AlertEscalationPolicy defines escalation steps for alerts
type AlertEscalationPolicy struct {
	ID          uuid.UUID                `json:"id" db:"id"`
	ProjectID   uuid.UUID                `json:"project_id" db:"project_id"`
	Name        string                   `json:"name" db:"name"`
	Description *string                  `json:"description,omitempty" db:"description"`
	Steps       []EscalationStep         `json:"steps" db:"steps"`
	CreatedAt   time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at" db:"updated_at"`
}

// EscalationStep represents a single step in an escalation policy
type EscalationStep struct {
	Delay    int      `json:"delay"`    // seconds to wait before escalating
	Channels []string `json:"channels"` // notification channels to use at this step
}

// AlertNotificationLog tracks notification delivery
type AlertNotificationLog struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	AlertHistoryID uuid.UUID  `json:"alert_history_id" db:"alert_history_id"`
	ChannelType    string     `json:"channel_type" db:"channel_type"`     // 'email', 'slack', 'webhook'
	ChannelTarget  string     `json:"channel_target" db:"channel_target"` // actual target (email address, webhook URL, etc)
	Status         string     `json:"status" db:"status"`                 // 'sent', 'failed', 'pending'
	ErrorMessage   *string    `json:"error_message,omitempty" db:"error_message"`
	SentAt         time.Time  `json:"sent_at" db:"sent_at"`
}

// AlertMetricResult represents the result of a metric evaluation
type AlertMetricResult struct {
	Value     float64
	Timestamp time.Time
	Labels    map[string]string
}

// AlertEvaluation represents the result of evaluating an alert rule
type AlertEvaluation struct {
	RuleID      uuid.UUID
	Triggered   bool
	MetricValue float64
	Timestamp   time.Time
	Labels      map[string]string
	Message     string
}
