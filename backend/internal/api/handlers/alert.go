package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// AlertHandler handles alert-related endpoints
type AlertHandler struct {
	alertService *service.AlertService
	logger       *zap.Logger
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(alertService *service.AlertService, logger *zap.Logger) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
		logger:       logger,
	}
}

// CreateAlertRuleRequest represents the request to create an alert rule
type CreateAlertRuleRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	Description          *string                `json:"description"`
	Enabled              bool                   `json:"enabled"`
	MetricType           string                 `json:"metric_type" binding:"required"`
	MetricField          *string                `json:"metric_field"`
	ConditionType        string                 `json:"condition_type" binding:"required"`
	Operator             string                 `json:"operator" binding:"required"`
	ThresholdValue       *float64               `json:"threshold_value" binding:"required"`
	WindowDuration       int                    `json:"window_duration"`
	EvaluationFrequency  int                    `json:"evaluation_frequency"`
	Filters              map[string]interface{} `json:"filters"`
	NotificationChannels []string               `json:"notification_channels"`
	NotificationMessage  *string                `json:"notification_message"`
	EscalationPolicyID   *uuid.UUID             `json:"escalation_policy_id"`
	GroupBy              []string               `json:"group_by"`
	GroupWait            int                    `json:"group_wait"`
	RepeatInterval       int                    `json:"repeat_interval"`
	Severity             string                 `json:"severity"`
	Tags                 []string               `json:"tags"`
}

// UpdateAlertRuleRequest represents the request to update an alert rule
type UpdateAlertRuleRequest struct {
	Name                 string                 `json:"name"`
	Description          *string                `json:"description"`
	Enabled              *bool                  `json:"enabled"`
	MetricType           string                 `json:"metric_type"`
	MetricField          *string                `json:"metric_field"`
	ConditionType        string                 `json:"condition_type"`
	Operator             string                 `json:"operator"`
	ThresholdValue       *float64               `json:"threshold_value"`
	WindowDuration       *int                   `json:"window_duration"`
	EvaluationFrequency  *int                   `json:"evaluation_frequency"`
	Filters              map[string]interface{} `json:"filters"`
	NotificationChannels []string               `json:"notification_channels"`
	NotificationMessage  *string                `json:"notification_message"`
	EscalationPolicyID   *uuid.UUID             `json:"escalation_policy_id"`
	GroupBy              []string               `json:"group_by"`
	GroupWait            *int                   `json:"group_wait"`
	RepeatInterval       *int                   `json:"repeat_interval"`
	Severity             string                 `json:"severity"`
	Tags                 []string               `json:"tags"`
}

// AcknowledgeAlertRequest represents the request to acknowledge an alert
type AcknowledgeAlertRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// CreateAlertRule creates a new alert rule
// @Summary Create alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param rule body CreateAlertRuleRequest true "Alert rule"
// @Success 201 {object} domain.AlertRule
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/rules [post]
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Invalid project ID format",
		})
		return
	}

	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	rule := &domain.AlertRule{
		ProjectID:            projectID,
		Name:                 req.Name,
		Description:          req.Description,
		Enabled:              req.Enabled,
		MetricType:           req.MetricType,
		MetricField:          req.MetricField,
		ConditionType:        req.ConditionType,
		Operator:             req.Operator,
		ThresholdValue:       req.ThresholdValue,
		WindowDuration:       req.WindowDuration,
		EvaluationFrequency:  req.EvaluationFrequency,
		Filters:              req.Filters,
		NotificationChannels: req.NotificationChannels,
		NotificationMessage:  req.NotificationMessage,
		EscalationPolicyID:   req.EscalationPolicyID,
		GroupBy:              req.GroupBy,
		GroupWait:            req.GroupWait,
		RepeatInterval:       req.RepeatInterval,
		Severity:             req.Severity,
		Tags:                 req.Tags,
	}

	// Set defaults
	if rule.WindowDuration == 0 {
		rule.WindowDuration = 300
	}
	if rule.EvaluationFrequency == 0 {
		rule.EvaluationFrequency = 60
	}
	if rule.GroupWait == 0 {
		rule.GroupWait = 30
	}
	if rule.RepeatInterval == 0 {
		rule.RepeatInterval = 3600
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}

	if err := h.alertService.CreateAlertRule(c.Request.Context(), rule); err != nil {
		h.logger.Error("failed to create alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create alert rule",
		})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListAlertRules lists alert rules for a project
// @Summary List alert rules
// @Tags alerts
// @Produce json
// @Param projectId path string true "Project ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} ListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/rules [get]
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Invalid project ID format",
		})
		return
	}

	var query PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_query",
			Message: err.Error(),
		})
		return
	}

	opts := &postgres.ListOptions{
		Limit:  query.Limit,
		Offset: query.Offset,
	}

	rules, total, err := h.alertService.ListAlertRules(c.Request.Context(), projectID, opts)
	if err != nil {
		h.logger.Error("failed to list alert rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list alert rules",
		})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:   rules,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	})
}

// GetAlertRule gets an alert rule by ID
// @Summary Get alert rule
// @Tags alerts
// @Produce json
// @Param projectId path string true "Project ID"
// @Param ruleId path string true "Rule ID"
// @Success 200 {object} domain.AlertRule
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/rules/{ruleId} [get]
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
	ruleIDStr := c.Param("ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_rule_id",
			Message: "Invalid rule ID format",
		})
		return
	}

	rule, err := h.alertService.GetAlertRule(c.Request.Context(), ruleID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Alert rule not found",
			})
			return
		}
		h.logger.Error("failed to get alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get alert rule",
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateAlertRule updates an alert rule
// @Summary Update alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param ruleId path string true "Rule ID"
// @Param rule body UpdateAlertRuleRequest true "Alert rule updates"
// @Success 200 {object} domain.AlertRule
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/rules/{ruleId} [put]
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	ruleIDStr := c.Param("ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_rule_id",
			Message: "Invalid rule ID format",
		})
		return
	}

	rule, err := h.alertService.GetAlertRule(c.Request.Context(), ruleID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Alert rule not found",
			})
			return
		}
		h.logger.Error("failed to get alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get alert rule",
		})
		return
	}

	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Update fields
	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != nil {
		rule.Description = req.Description
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.MetricType != "" {
		rule.MetricType = req.MetricType
	}
	if req.MetricField != nil {
		rule.MetricField = req.MetricField
	}
	if req.ConditionType != "" {
		rule.ConditionType = req.ConditionType
	}
	if req.Operator != "" {
		rule.Operator = req.Operator
	}
	if req.ThresholdValue != nil {
		rule.ThresholdValue = req.ThresholdValue
	}
	if req.WindowDuration != nil {
		rule.WindowDuration = *req.WindowDuration
	}
	if req.EvaluationFrequency != nil {
		rule.EvaluationFrequency = *req.EvaluationFrequency
	}
	if req.Filters != nil {
		rule.Filters = req.Filters
	}
	if req.NotificationChannels != nil {
		rule.NotificationChannels = req.NotificationChannels
	}
	if req.NotificationMessage != nil {
		rule.NotificationMessage = req.NotificationMessage
	}
	if req.EscalationPolicyID != nil {
		rule.EscalationPolicyID = req.EscalationPolicyID
	}
	if req.GroupBy != nil {
		rule.GroupBy = req.GroupBy
	}
	if req.GroupWait != nil {
		rule.GroupWait = *req.GroupWait
	}
	if req.RepeatInterval != nil {
		rule.RepeatInterval = *req.RepeatInterval
	}
	if req.Severity != "" {
		rule.Severity = req.Severity
	}
	if req.Tags != nil {
		rule.Tags = req.Tags
	}

	if err := h.alertService.UpdateAlertRule(c.Request.Context(), rule); err != nil {
		h.logger.Error("failed to update alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update alert rule",
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteAlertRule deletes an alert rule
// @Summary Delete alert rule
// @Tags alerts
// @Param projectId path string true "Project ID"
// @Param ruleId path string true "Rule ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/rules/{ruleId} [delete]
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	ruleIDStr := c.Param("ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_rule_id",
			Message: "Invalid rule ID format",
		})
		return
	}

	if err := h.alertService.DeleteAlertRule(c.Request.Context(), ruleID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Alert rule not found",
			})
			return
		}
		h.logger.Error("failed to delete alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete alert rule",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAlertHistory lists alert history for a project
// @Summary List alert history
// @Tags alerts
// @Produce json
// @Param projectId path string true "Project ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} ListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/history [get]
func (h *AlertHandler) ListAlertHistory(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Invalid project ID format",
		})
		return
	}

	var query PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_query",
			Message: err.Error(),
		})
		return
	}

	opts := &postgres.ListOptions{
		Limit:  query.Limit,
		Offset: query.Offset,
	}

	history, total, err := h.alertService.ListAlertHistory(c.Request.Context(), projectID, opts)
	if err != nil {
		h.logger.Error("failed to list alert history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list alert history",
		})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:   history,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	})
}

// GetAlertHistory gets alert history by ID
// @Summary Get alert history
// @Tags alerts
// @Produce json
// @Param projectId path string true "Project ID"
// @Param alertId path string true "Alert ID"
// @Success 200 {object} domain.AlertHistory
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/history/{alertId} [get]
func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
	alertIDStr := c.Param("alertId")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_alert_id",
			Message: "Invalid alert ID format",
		})
		return
	}

	history, err := h.alertService.GetAlertHistory(c.Request.Context(), alertID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Alert not found",
			})
			return
		}
		h.logger.Error("failed to get alert history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get alert history",
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// AcknowledgeAlert acknowledges an alert
// @Summary Acknowledge alert
// @Tags alerts
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param alertId path string true "Alert ID"
// @Param request body AcknowledgeAlertRequest true "Acknowledgment request"
// @Success 200 {object} domain.AlertHistory
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/history/{alertId}/acknowledge [post]
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	alertIDStr := c.Param("alertId")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_alert_id",
			Message: "Invalid alert ID format",
		})
		return
	}

	var req AcknowledgeAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.alertService.AcknowledgeAlert(c.Request.Context(), alertID, req.UserID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Alert not found or already acknowledged",
			})
			return
		}
		h.logger.Error("failed to acknowledge alert", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to acknowledge alert",
		})
		return
	}

	history, err := h.alertService.GetAlertHistory(c.Request.Context(), alertID)
	if err != nil {
		h.logger.Error("failed to get alert history after acknowledgment", zap.Error(err))
	}

	c.JSON(http.StatusOK, history)
}

// EvaluateAlerts triggers alert evaluation for a project
// @Summary Evaluate alerts
// @Tags alerts
// @Param projectId path string true "Project ID"
// @Success 202
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/projects/{projectId}/alerts/evaluate [post]
func (h *AlertHandler) EvaluateAlerts(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Invalid project ID format",
		})
		return
	}

	// Trigger evaluation asynchronously
	go func() {
		ctx := context.Background()
		if err := h.alertService.EvaluateAlerts(ctx, projectID); err != nil {
			h.logger.Error("failed to evaluate alerts", zap.Error(err))
		}
	}()

	c.Status(http.StatusAccepted)
}
