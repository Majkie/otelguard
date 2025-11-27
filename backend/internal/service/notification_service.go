package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/otelguard/otelguard/internal/domain"
	"go.uber.org/zap"
)

// NotificationService handles sending notifications through various channels
type NotificationService struct {
	logger     *zap.Logger
	httpClient *http.Client
}

// NewNotificationService creates a new notification service
func NewNotificationService(logger *zap.Logger) *NotificationService {
	return &NotificationService{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendEmail sends an email notification
func (s *NotificationService) SendEmail(ctx context.Context, email, subject, body string) error {
	// In a production system, integrate with an email service (SendGrid, SES, etc.)
	// For now, this is a placeholder
	s.logger.Info("sending email notification",
		zap.String("to", email),
		zap.String("subject", subject),
	)

	// TODO: Implement actual email sending
	return nil
}

// SendSlack sends a Slack notification
func (s *NotificationService) SendSlack(ctx context.Context, webhookURL string, rule *domain.AlertRule, history *domain.AlertHistory) error {
	// Build Slack message
	color := s.getSeverityColor(history.Severity)

	attachment := map[string]interface{}{
		"color": color,
		"title": fmt.Sprintf("🚨 %s", rule.Name),
		"fields": []map[string]interface{}{
			{
				"title": "Severity",
				"value": history.Severity,
				"short": true,
			},
			{
				"title": "Status",
				"value": history.Status,
				"short": true,
			},
		},
		"timestamp": history.FiredAt.Unix(),
	}

	if history.MetricValue != nil {
		attachment["fields"] = append(attachment["fields"].([]map[string]interface{}), map[string]interface{}{
			"title": "Metric Value",
			"value": fmt.Sprintf("%.2f", *history.MetricValue),
			"short": true,
		})
	}

	if history.ThresholdValue != nil {
		attachment["fields"] = append(attachment["fields"].([]map[string]interface{}), map[string]interface{}{
			"title": "Threshold",
			"value": fmt.Sprintf("%.2f", *history.ThresholdValue),
			"short": true,
		})
	}

	if history.Message != nil {
		attachment["text"] = *history.Message
	}

	payload := map[string]interface{}{
		"attachments": []interface{}{attachment},
	}

	return s.sendWebhookPayload(ctx, webhookURL, payload)
}

// SendWebhook sends a generic webhook notification
func (s *NotificationService) SendWebhook(ctx context.Context, webhookURL string, rule *domain.AlertRule, history *domain.AlertHistory) error {
	payload := map[string]interface{}{
		"alert_id":     history.ID.String(),
		"rule_id":      rule.ID.String(),
		"rule_name":    rule.Name,
		"severity":     history.Severity,
		"status":       history.Status,
		"metric_type":  rule.MetricType,
		"metric_value": history.MetricValue,
		"threshold":    history.ThresholdValue,
		"fired_at":     history.FiredAt,
		"message":      history.Message,
		"labels":       history.GroupLabels,
	}

	return s.sendWebhookPayload(ctx, webhookURL, payload)
}

// sendWebhookPayload sends a JSON payload to a webhook URL
func (s *NotificationService) sendWebhookPayload(ctx context.Context, url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	s.logger.Info("webhook notification sent successfully",
		zap.String("url", url),
		zap.Int("status", resp.StatusCode),
	)

	return nil
}

// getSeverityColor returns a color for the severity level
func (s *NotificationService) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "#ff0000" // Red
	case "error":
		return "#ff6600" // Orange
	case "warning":
		return "#ffcc00" // Yellow
	case "info":
		return "#0099ff" // Blue
	default:
		return "#cccccc" // Gray
	}
}
