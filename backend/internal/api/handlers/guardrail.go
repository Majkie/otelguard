package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// GuardrailHandler handles guardrail-related endpoints
type GuardrailHandler struct {
	guardrailService *service.GuardrailService
	logger           *zap.Logger
}

// NewGuardrailHandler creates a new guardrail handler
func NewGuardrailHandler(guardrailService *service.GuardrailService, logger *zap.Logger) *GuardrailHandler {
	return &GuardrailHandler{
		guardrailService: guardrailService,
		logger:           logger,
	}
}

// EvaluateRequest represents a guardrail evaluation request
type EvaluateRequest struct {
	Input    string                 `json:"input" binding:"required"`
	Output   string                 `json:"output"`
	Context  map[string]interface{} `json:"context,omitempty"`
	TraceID  string                 `json:"traceId,omitempty"`
	PolicyID string                 `json:"policyId,omitempty"`
}

// EvaluateResponse represents the evaluation response
type EvaluateResponse struct {
	Passed       bool              `json:"passed"`
	Violations   []ViolationResult `json:"violations,omitempty"`
	Remediated   bool              `json:"remediated"`
	Output       string            `json:"output,omitempty"`
	LatencyMs    int64             `json:"latencyMs"`
	EvaluationID string            `json:"evaluationId"`
}

// ViolationResult represents a single rule violation
type ViolationResult struct {
	RuleID      string `json:"ruleId"`
	RuleType    string `json:"ruleType"`
	Message     string `json:"message"`
	Action      string `json:"action"`
	ActionTaken bool   `json:"actionTaken"`
}

// Evaluate evaluates content against guardrail policies
func (h *GuardrailHandler) Evaluate(c *gin.Context) {
	start := time.Now()

	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// TODO: Implement actual guardrail evaluation
	// For now, return a pass result
	latencyMs := time.Since(start).Milliseconds()

	c.JSON(http.StatusOK, EvaluateResponse{
		Passed:       true,
		Violations:   []ViolationResult{},
		Remediated:   false,
		Output:       req.Output,
		LatencyMs:    latencyMs,
		EvaluationID: uuid.New().String(),
	})
}

// List returns all guardrail policies
func (h *GuardrailHandler) List(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	policies, total, err := h.guardrailService.List(c.Request.Context(), projectID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list policies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve policies",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   policies,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create creates a new guardrail policy
func (h *GuardrailHandler) Create(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Enabled     bool                   `json:"enabled"`
		Priority    int                    `json:"priority"`
		Triggers    map[string]interface{} `json:"triggers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.Query("projectId")
	if projectID == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	triggersJSON, _ := json.Marshal(req.Triggers)
	now := time.Now()
	policy := &domain.GuardrailPolicy{
		ID:          uuid.New(),
		ProjectID:   projectUUID,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		Triggers:    triggersJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.guardrailService.Create(c.Request.Context(), policy); err != nil {
		h.logger.Error("failed to create policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create policy",
		})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// Get returns a single guardrail policy
func (h *GuardrailHandler) Get(c *gin.Context) {
	id := c.Param("id")
	policy, err := h.guardrailService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Policy not found",
			})
			return
		}
		h.logger.Error("failed to get policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve policy",
		})
		return
	}
	c.JSON(http.StatusOK, policy)
}

// Update updates a guardrail policy
func (h *GuardrailHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Enabled     *bool                  `json:"enabled"`
		Priority    *int                   `json:"priority"`
		Triggers    map[string]interface{} `json:"triggers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	policy, err := h.guardrailService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Policy not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve policy",
		})
		return
	}

	if req.Name != "" {
		policy.Name = req.Name
	}
	policy.Description = req.Description
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		policy.Priority = *req.Priority
	}
	if req.Triggers != nil {
		triggersJSON, _ := json.Marshal(req.Triggers)
		policy.Triggers = triggersJSON
	}
	policy.UpdatedAt = time.Now()

	if err := h.guardrailService.Update(c.Request.Context(), policy); err != nil {
		h.logger.Error("failed to update policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update policy",
		})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// Delete deletes a guardrail policy
func (h *GuardrailHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.guardrailService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete policy",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Policy deleted"})
}

// ListRules returns all rules for a policy
func (h *GuardrailHandler) ListRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
}

// AddRule adds a rule to a policy
func (h *GuardrailHandler) AddRule(c *gin.Context) {
	var req struct {
		Type         string                 `json:"type" binding:"required"`
		Config       map[string]interface{} `json:"config"`
		Action       string                 `json:"action" binding:"required"`
		ActionConfig map[string]interface{} `json:"actionConfig,omitempty"`
		OrderIndex   int                    `json:"orderIndex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      uuid.New().String(),
		"message": "Rule added",
	})
}

// UpdateRule updates a rule
func (h *GuardrailHandler) UpdateRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// DeleteRule deletes a rule
func (h *GuardrailHandler) DeleteRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// CreateVersion creates a new version snapshot of a policy
func (h *GuardrailHandler) CreateVersion(c *gin.Context) {
	policyID := c.Param("id")
	if _, err := uuid.Parse(policyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid policy ID",
		})
		return
	}

	var req struct {
		ChangeNotes string `json:"changeNotes,omitempty"`
		CreatedBy   string `json:"createdBy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid createdBy UUID",
		})
		return
	}

	version, err := h.guardrailService.CreateVersion(c.Request.Context(), policyID, req.ChangeNotes, createdBy)
	if err != nil {
		h.logger.Error("failed to create policy version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create version",
		})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetVersion retrieves a specific version of a policy
func (h *GuardrailHandler) GetVersion(c *gin.Context) {
	policyID := c.Param("id")
	versionStr := c.Param("version")

	if _, err := uuid.Parse(policyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid policy ID",
		})
		return
	}

	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid version number",
		})
		return
	}

	policyVersion, err := h.guardrailService.GetVersion(c.Request.Context(), policyID, version)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Version not found",
			})
			return
		}
		h.logger.Error("failed to get policy version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve version",
		})
		return
	}

	c.JSON(http.StatusOK, policyVersion)
}

// ListVersions retrieves all versions of a policy
func (h *GuardrailHandler) ListVersions(c *gin.Context) {
	policyID := c.Param("id")
	if _, err := uuid.Parse(policyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid policy ID",
		})
		return
	}

	versions, err := h.guardrailService.ListVersions(c.Request.Context(), policyID)
	if err != nil {
		h.logger.Error("failed to list policy versions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve versions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": versions,
	})
}

// RestoreVersion restores a policy to a previous version
func (h *GuardrailHandler) RestoreVersion(c *gin.Context) {
	policyID := c.Param("id")
	versionStr := c.Param("version")

	if _, err := uuid.Parse(policyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid policy ID",
		})
		return
	}

	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid version number",
		})
		return
	}

	var req struct {
		CreatedBy string `json:"createdBy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid createdBy UUID",
		})
		return
	}

	if err := h.guardrailService.RestoreVersion(c.Request.Context(), policyID, version, createdBy); err != nil {
		h.logger.Error("failed to restore policy version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Policy restored to version %d", version),
	})
}
