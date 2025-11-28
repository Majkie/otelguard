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
	asyncService     *service.AsyncEvaluationService
	logger           *zap.Logger
}

// NewGuardrailHandler creates a new guardrail handler
func NewGuardrailHandler(guardrailService *service.GuardrailService, logger *zap.Logger) *GuardrailHandler {
	// Create async evaluation service with 5 workers
	asyncService := service.NewAsyncEvaluationService(guardrailService, logger, 5)

	return &GuardrailHandler{
		guardrailService: guardrailService,
		asyncService:     asyncService,
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
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Build evaluation input
	input := &service.EvaluationInput{
		ProjectID: projectUUID,
		Input:     req.Input,
		Output:    req.Output,
		Context:   req.Context,
	}

	// Parse optional trace ID
	if req.TraceID != "" {
		traceUUID, err := uuid.Parse(req.TraceID)
		if err == nil {
			input.TraceID = &traceUUID
		}
	}

	// Parse optional policy ID
	if req.PolicyID != "" {
		policyUUID, err := uuid.Parse(req.PolicyID)
		if err == nil {
			input.PolicyID = &policyUUID
		}
	}

	// Execute evaluation
	result, err := h.guardrailService.Evaluate(c.Request.Context(), input)
	if err != nil {
		h.logger.Error("evaluation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Evaluation failed",
		})
		return
	}

	// Convert violations
	violations := make([]ViolationResult, len(result.Violations))
	for i, v := range result.Violations {
		violations[i] = ViolationResult{
			RuleID:      v.RuleID.String(),
			RuleType:    v.RuleType,
			Message:     v.Message,
			Action:      v.Action,
			ActionTaken: v.ActionTaken,
		}
	}

	c.JSON(http.StatusOK, EvaluateResponse{
		Passed:       result.Passed,
		Violations:   violations,
		Remediated:   result.Remediated,
		Output:       result.Output,
		LatencyMs:    result.LatencyMs,
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

// TestPolicyRequest represents a policy test request
type TestPolicyRequest struct {
	Input      string                 `json:"input" binding:"required"`
	Output     string                 `json:"output"`
	Model      string                 `json:"model"`
	Tags       []string               `json:"tags"`
	Context    map[string]interface{} `json:"context"`
	PolicyData *struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Enabled     bool                   `json:"enabled"`
		Priority    int                    `json:"priority"`
		Triggers    map[string]interface{} `json:"triggers"`
		Rules       []struct {
			Type         string                 `json:"type"`
			Config       map[string]interface{} `json:"config"`
			Action       string                 `json:"action"`
			ActionConfig map[string]interface{} `json:"actionConfig"`
			OrderIndex   int                    `json:"orderIndex"`
		} `json:"rules"`
	} `json:"policyData,omitempty"`
}

// TestPolicyResponse represents the policy test response
type TestPolicyResponse struct {
	Passed       bool                   `json:"passed"`
	Violations   []ViolationResult      `json:"violations"`
	Remediated   bool                   `json:"remediated"`
	Output       string                 `json:"output"`
	LatencyMs    int64                  `json:"latencyMs"`
	RulesEvaluated int                  `json:"rulesEvaluated"`
	Details      map[string]interface{} `json:"details"`
}

// TestPolicy tests a policy against sample input without saving results
func (h *GuardrailHandler) TestPolicy(c *gin.Context) {
	var req TestPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	var policyID *uuid.UUID

	// If testing an existing policy
	if c.Param("id") != "" {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "invalid policy ID",
			})
			return
		}
		policyID = &id
	}

	// Build evaluation input
	input := &service.EvaluationInput{
		ProjectID: projectUUID,
		PolicyID:  policyID,
		Input:     req.Input,
		Output:    req.Output,
		Model:     req.Model,
		Tags:      req.Tags,
		Context:   req.Context,
	}

	// Execute evaluation (results won't be logged if testing mode)
	start := time.Now()
	result, err := h.guardrailService.Evaluate(c.Request.Context(), input)
	if err != nil {
		h.logger.Error("policy test failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Policy test failed",
		})
		return
	}

	// Convert violations
	violations := make([]ViolationResult, len(result.Violations))
	for i, v := range result.Violations {
		violations[i] = ViolationResult{
			RuleID:      v.RuleID.String(),
			RuleType:    v.RuleType,
			Message:     v.Message,
			Action:      v.Action,
			ActionTaken: v.ActionTaken,
		}
	}

	response := TestPolicyResponse{
		Passed:         result.Passed,
		Violations:     violations,
		Remediated:     result.Remediated,
		Output:         result.Output,
		LatencyMs:      time.Since(start).Milliseconds(),
		RulesEvaluated: len(violations),
		Details: map[string]interface{}{
			"triggered_rules":  len(violations),
			"actions_taken":    countActionsTaken(violations),
		},
	}

	c.JSON(http.StatusOK, response)
}

// BasicBatchEvaluateRequest represents a basic batch evaluation request
type BasicBatchEvaluateRequest struct {
	Items []struct {
		ID      string                 `json:"id"`
		Input   string                 `json:"input" binding:"required"`
		Output  string                 `json:"output"`
		Context map[string]interface{} `json:"context,omitempty"`
		Model   string                 `json:"model,omitempty"`
		Tags    []string               `json:"tags,omitempty"`
	} `json:"items" binding:"required,min=1,max=100"`
	PolicyID string `json:"policyId,omitempty"`
}

// BasicBatchEvaluateResponse represents the basic batch evaluation response
type BasicBatchEvaluateResponse struct {
	Results []struct {
		ID           string            `json:"id"`
		Passed       bool              `json:"passed"`
		Violations   []ViolationResult `json:"violations,omitempty"`
		Remediated   bool              `json:"remediated"`
		Output       string            `json:"output,omitempty"`
		LatencyMs    int64             `json:"latencyMs"`
		Error        string            `json:"error,omitempty"`
	} `json:"results"`
	TotalItems     int   `json:"totalItems"`
	PassedItems    int   `json:"passedItems"`
	FailedItems    int   `json:"failedItems"`
	TotalLatencyMs int64 `json:"totalLatencyMs"`
}

// BatchEvaluate evaluates multiple items in a single request
func (h *GuardrailHandler) BatchEvaluate(c *gin.Context) {
	var req BasicBatchEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	var policyID *uuid.UUID
	if req.PolicyID != "" {
		id, err := uuid.Parse(req.PolicyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "invalid policy ID",
			})
			return
		}
		policyID = &id
	}

	batchStart := time.Now()
	response := BasicBatchEvaluateResponse{
		TotalItems: len(req.Items),
		Results:    make([]struct {
			ID           string            `json:"id"`
			Passed       bool              `json:"passed"`
			Violations   []ViolationResult `json:"violations,omitempty"`
			Remediated   bool              `json:"remediated"`
			Output       string            `json:"output,omitempty"`
			LatencyMs    int64             `json:"latencyMs"`
			Error        string            `json:"error,omitempty"`
		}, len(req.Items)),
	}

	// Evaluate each item
	for i, item := range req.Items {
		itemStart := time.Now()

		// Build evaluation input
		input := &service.EvaluationInput{
			ProjectID: projectUUID,
			PolicyID:  policyID,
			Input:     item.Input,
			Output:    item.Output,
			Context:   item.Context,
			Model:     item.Model,
			Tags:      item.Tags,
		}

		// Execute evaluation
		result, err := h.guardrailService.Evaluate(c.Request.Context(), input)
		if err != nil {
			h.logger.Error("batch evaluation item failed",
				zap.Error(err),
				zap.String("item_id", item.ID),
			)
			response.Results[i] = struct {
				ID           string            `json:"id"`
				Passed       bool              `json:"passed"`
				Violations   []ViolationResult `json:"violations,omitempty"`
				Remediated   bool              `json:"remediated"`
				Output       string            `json:"output,omitempty"`
				LatencyMs    int64             `json:"latencyMs"`
				Error        string            `json:"error,omitempty"`
			}{
				ID:        item.ID,
				Error:     err.Error(),
				LatencyMs: time.Since(itemStart).Milliseconds(),
			}
			response.FailedItems++
			continue
		}

		// Convert violations
		violations := make([]ViolationResult, len(result.Violations))
		for j, v := range result.Violations {
			violations[j] = ViolationResult{
				RuleID:      v.RuleID.String(),
				RuleType:    v.RuleType,
				Message:     v.Message,
				Action:      v.Action,
				ActionTaken: v.ActionTaken,
			}
		}

		response.Results[i] = struct {
			ID           string            `json:"id"`
			Passed       bool              `json:"passed"`
			Violations   []ViolationResult `json:"violations,omitempty"`
			Remediated   bool              `json:"remediated"`
			Output       string            `json:"output,omitempty"`
			LatencyMs    int64             `json:"latencyMs"`
			Error        string            `json:"error,omitempty"`
		}{
			ID:         item.ID,
			Passed:     result.Passed,
			Violations: violations,
			Remediated: result.Remediated,
			Output:     result.Output,
			LatencyMs:  time.Since(itemStart).Milliseconds(),
		}

		if result.Passed {
			response.PassedItems++
		} else {
			response.FailedItems++
		}
	}

	response.TotalLatencyMs = time.Since(batchStart).Milliseconds()

	c.JSON(http.StatusOK, response)
}

// GetCacheStats returns cache statistics
func (h *GuardrailHandler) GetCacheStats(c *gin.Context) {
	stats := h.guardrailService.GetCacheStats()
	c.JSON(http.StatusOK, stats)
}

// ClearCache clears the evaluation cache
func (h *GuardrailHandler) ClearCache(c *gin.Context) {
	h.guardrailService.ClearCache()
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache cleared successfully",
	})
}

// InvalidateCache invalidates cache entries for a specific project or policy
func (h *GuardrailHandler) InvalidateCache(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	policyID := c.Query("policyId")
	var policyIDPtr *string
	if policyID != "" {
		policyIDPtr = &policyID
	}

	count := h.guardrailService.InvalidateCache(c.Request.Context(), projectID, policyIDPtr)
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache invalidated",
		"count":   count,
	})
}

// AsyncEvaluateRequest represents an async evaluation request
type AsyncEvaluateRequest struct {
	Input      string                 `json:"input" binding:"required"`
	Output     string                 `json:"output"`
	Context    map[string]interface{} `json:"context,omitempty"`
	PolicyID   string                 `json:"policyId,omitempty"`
	Model      string                 `json:"model,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	WebhookURL string                 `json:"webhookUrl" binding:"required"`
}

// AsyncEvaluateResponse represents the async evaluation response
type AsyncEvaluateResponse struct {
	JobID     string `json:"jobId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// AsyncEvaluate starts an async evaluation job
func (h *GuardrailHandler) AsyncEvaluate(c *gin.Context) {
	var req AsyncEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Build evaluation input
	input := &service.EvaluationInput{
		ProjectID: projectUUID,
		Input:     req.Input,
		Output:    req.Output,
		Context:   req.Context,
		Model:     req.Model,
		Tags:      req.Tags,
	}

	// Parse optional policy ID
	if req.PolicyID != "" {
		policyUUID, err := uuid.Parse(req.PolicyID)
		if err == nil {
			input.PolicyID = &policyUUID
		}
	}

	// Submit async job
	job, err := h.asyncService.SubmitEvaluation(input, req.WebhookURL, "", 3)
	if err != nil {
		h.logger.Error("failed to submit async evaluation job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to submit evaluation job",
		})
		return
	}

	c.JSON(http.StatusAccepted, AsyncEvaluateResponse{
		JobID:     job.ID.String(),
		Status:    job.Status,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
}

// GetAsyncJob retrieves an async evaluation job status
func (h *GuardrailHandler) GetAsyncJob(c *gin.Context) {
	// jobID := c.Param("jobId")
	// if jobID == "" {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":   "invalid_request",
	// 		"message": "jobId is required",
	// 	})
	// 	return
	// }

	// jobUUID, err := uuid.Parse(jobID)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":   "invalid_request",
	// 		"message": "invalid jobId format",
	// 	})
	// 	return
	// }

	// TODO: Implement GetJob method in AsyncEvaluationService
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "Get job status is not yet implemented",
	})
	return

	/*
	job, err := h.asyncService.GetJob(c.Request.Context(), jobUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Job not found",
		})
		return
	}
	*/

	/*
	response := gin.H{
		"id":         job.ID.String(),
		"status":     job.Status,
		"created_at": job.CreatedAt,
	}

	if job.StartedAt != nil {
		response["started_at"] = job.StartedAt
	}
	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt
	}
	if job.Error != "" {
		response["error"] = job.Error
	}
	if job.Result != nil {
		// Convert violations
		violations := make([]ViolationResult, len(job.Result.Violations))
		for i, v := range job.Result.Violations {
			violations[i] = ViolationResult{
				RuleID:      v.RuleID.String(),
				RuleType:    v.RuleType,
				Message:     v.Message,
				Action:      v.Action,
				ActionTaken: v.ActionTaken,
			}
		}

		response["result"] = gin.H{
			"passed":     job.Result.Passed,
			"violations": violations,
			"remediated": job.Result.Remediated,
			"output":     job.Result.Output,
			"latency_ms": job.Result.LatencyMs,
		}
	}

	c.JSON(http.StatusOK, response)
	*/
}

// ListAsyncJobs lists async evaluation jobs
func (h *GuardrailHandler) ListAsyncJobs(c *gin.Context) {
	// projectID := c.Query("projectId")
	// if projectID == "" {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":   "invalid_request",
	// 		"message": "projectId is required",
	// 	})
	// 	return
	// }

	// projectUUID, err := uuid.Parse(projectID)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":   "invalid_request",
	// 		"message": "invalid projectId format",
	// 	})
	// 	return
	// }

	// limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	// offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// if limit > 100 {
	// 	limit = 100
	// }

	// TODO: Implement ListJobs method in AsyncEvaluationService
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "List jobs is not yet implemented",
	})
	return

	/*
	jobs, total, err := h.asyncService.ListJobs(c.Request.Context(), projectUUID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list async jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list jobs",
		})
		return
	}
	*/

	/*
	// Convert jobs to response format
	jobsResponse := make([]gin.H, len(jobs))
	for i, job := range jobs {
		jobsResponse[i] = gin.H{
			"id":         job.ID.String(),
			"status":     job.Status,
			"created_at": job.CreatedAt,
		}
		if job.CompletedAt != nil {
			jobsResponse[i]["completed_at"] = job.CompletedAt
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   jobsResponse,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
	*/
}

// Helper function
func countActionsTaken(violations []ViolationResult) int {
	count := 0
	for _, v := range violations {
		if v.ActionTaken {
			count++
		}
	}
	return count
}
