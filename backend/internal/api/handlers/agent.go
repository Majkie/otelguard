package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"github.com/otelguard/otelguard/pkg/validator"
	"go.uber.org/zap"
)

// AgentHandler handles agent-related endpoints
type AgentHandler struct {
	agentService *service.AgentService
	logger       *zap.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(agentService *service.AgentService, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		logger:       logger,
	}
}

// CreateAgentRequest represents an agent creation request
type CreateAgentRequest struct {
	ID           string   `json:"id,omitempty" binding:"omitempty,uuid"`
	TraceID      string   `json:"traceId" binding:"required,uuid"`
	SpanID       string   `json:"spanId" binding:"required,uuid"`
	ParentAgent  string   `json:"parentAgentId,omitempty" binding:"omitempty,uuid"`
	Name         string   `json:"name" binding:"required,min=1,max=255"`
	Type         string   `json:"type" binding:"required,oneof=orchestrator worker tool_caller planner executor reviewer custom"`
	Role         string   `json:"role,omitempty" binding:"max=255"`
	Model        string   `json:"model,omitempty" binding:"max=100"`
	SystemPrompt string   `json:"systemPrompt,omitempty"`
	StartTime    string   `json:"startTime"`
	EndTime      string   `json:"endTime"`
	LatencyMs    uint32   `json:"latencyMs"`
	TotalTokens  uint32   `json:"totalTokens"`
	Cost         float64  `json:"cost"`
	Status       string   `json:"status" binding:"oneof=running success error timeout"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	Metadata     string   `json:"metadata,omitempty"`
	Tags         []string `json:"tags,omitempty" binding:"max=50,dive,max=50"`
}

// AgentResponse represents an agent in API responses
type AgentResponse struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"projectId"`
	TraceID      string   `json:"traceId"`
	SpanID       string   `json:"spanId"`
	ParentAgent  *string  `json:"parentAgentId,omitempty"`
	Name         string   `json:"name"`
	Type         string   `json:"agentType"`
	Role         string   `json:"role"`
	Model        *string  `json:"model,omitempty"`
	SystemPrompt *string  `json:"systemPrompt,omitempty"`
	StartTime    string   `json:"startTime"`
	EndTime      string   `json:"endTime"`
	LatencyMs    uint32   `json:"latencyMs"`
	TotalTokens  uint32   `json:"totalTokens"`
	Cost         float64  `json:"cost"`
	Status       string   `json:"status"`
	ErrorMessage *string  `json:"errorMessage,omitempty"`
	Metadata     string   `json:"metadata,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	CreatedAt    string   `json:"createdAt"`
}

// toAgentResponse converts a domain agent to API response
func toAgentResponse(agent *domain.Agent) *AgentResponse {
	resp := &AgentResponse{
		ID:           agent.ID.String(),
		ProjectID:    agent.ProjectID.String(),
		TraceID:      agent.TraceID.String(),
		SpanID:       agent.SpanID.String(),
		Name:         agent.Name,
		Type:         agent.Type,
		Role:         agent.Role,
		Model:        agent.Model,
		SystemPrompt: agent.SystemPrompt,
		StartTime:    agent.StartTime.Format(time.RFC3339Nano),
		EndTime:      agent.EndTime.Format(time.RFC3339Nano),
		LatencyMs:    agent.LatencyMs,
		TotalTokens:  agent.TotalTokens,
		Cost:         agent.Cost,
		Status:       agent.Status,
		ErrorMessage: agent.ErrorMessage,
		Metadata:     agent.Metadata,
		Tags:         agent.Tags,
		CreatedAt:    agent.CreatedAt.Format(time.RFC3339Nano),
	}

	if agent.ParentAgent != nil {
		s := agent.ParentAgent.String()
		resp.ParentAgent = &s
	}

	return resp
}

// CreateAgent creates a new agent
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := validator.ParseValidationErrors(err)
		if len(validationErrors) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": validationErrors.Error(),
				"details": validationErrors,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(middleware.ContextProjectID)

	// Parse IDs
	var agentID uuid.UUID
	if req.ID != "" {
		agentID = uuid.MustParse(req.ID)
	} else {
		agentID = uuid.New()
	}

	projectUUID := uuid.MustParse(projectID)
	traceUUID := uuid.MustParse(req.TraceID)
	spanUUID := uuid.MustParse(req.SpanID)

	var parentAgent *uuid.UUID
	if req.ParentAgent != "" {
		p := uuid.MustParse(req.ParentAgent)
		parentAgent = &p
	}

	// Parse times
	startTime := time.Now()
	endTime := time.Now()
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.StartTime); err == nil {
			startTime = t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.EndTime); err == nil {
			endTime = t
		}
	}

	agent := &domain.Agent{
		ID:          agentID,
		ProjectID:   projectUUID,
		TraceID:     traceUUID,
		SpanID:      spanUUID,
		ParentAgent: parentAgent,
		Name:        req.Name,
		Type:        req.Type,
		Role:        req.Role,
		StartTime:   startTime,
		EndTime:     endTime,
		LatencyMs:   req.LatencyMs,
		TotalTokens: req.TotalTokens,
		Cost:        req.Cost,
		Status:      req.Status,
		Metadata:    req.Metadata,
		Tags:        req.Tags,
	}

	if req.Model != "" {
		agent.Model = &req.Model
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = &req.SystemPrompt
	}
	if req.ErrorMessage != "" {
		agent.ErrorMessage = &req.ErrorMessage
	}

	if err := h.agentService.CreateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error("failed to create agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to create agent",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        agent.ID.String(),
		"timestamp": agent.CreatedAt.Format(time.RFC3339Nano),
	})
}

// GetAgent retrieves an agent by ID
func (h *AgentHandler) GetAgent(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	agentID := c.Param("id")

	if _, err := uuid.Parse(agentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "agent ID must be a valid UUID",
		})
		return
	}

	agent, err := h.agentService.GetAgent(c.Request.Context(), projectID, agentID)
	if err != nil {
		h.logger.Error("failed to get agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve agent",
		})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "agent not found",
		})
		return
	}

	c.JSON(http.StatusOK, toAgentResponse(agent))
}

// ListAgents lists agents with filtering and pagination
func (h *AgentHandler) ListAgents(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)

	limit := 50
	offset := 0
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && o >= 0 {
		offset = o
	}

	opts := &service.ListAgentsOptions{
		ProjectID:   projectID,
		TraceID:     c.Query("traceId"),
		AgentType:   c.Query("type"),
		Status:      c.Query("status"),
		ParentAgent: c.Query("parentAgentId"),
		StartTime:   c.Query("startTime"),
		EndTime:     c.Query("endTime"),
		SortBy:      c.DefaultQuery("sortBy", "start_time"),
		SortOrder:   c.DefaultQuery("sortOrder", "DESC"),
		Limit:       limit,
		Offset:      offset,
	}

	agents, total, err := h.agentService.ListAgents(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to list agents",
		})
		return
	}

	responses := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = toAgentResponse(agent)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   responses,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetAgentsByTrace retrieves all agents for a trace
func (h *AgentHandler) GetAgentsByTrace(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	agents, err := h.agentService.GetAgentsByTrace(c.Request.Context(), projectID, traceID)
	if err != nil {
		h.logger.Error("failed to get agents for trace", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve agents",
		})
		return
	}

	responses := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = toAgentResponse(agent)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": len(responses),
	})
}

// GetAgentHierarchy retrieves the agent hierarchy for a trace
func (h *AgentHandler) GetAgentHierarchy(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	hierarchy, err := h.agentService.GetAgentHierarchy(c.Request.Context(), projectID, traceID)
	if err != nil {
		h.logger.Error("failed to get agent hierarchy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve agent hierarchy",
		})
		return
	}

	c.JSON(http.StatusOK, hierarchy)
}

// DetectAgents detects and stores agents from trace spans
func (h *AgentHandler) DetectAgents(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	agents, err := h.agentService.DetectAndStoreAgents(c.Request.Context(), projectID, traceID)
	if err != nil {
		h.logger.Error("failed to detect agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to detect agents",
		})
		return
	}

	responses := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = toAgentResponse(agent)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     responses,
		"detected": len(responses),
	})
}

// ToolCallResponse represents a tool call in API responses
type ToolCallResponse struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"projectId"`
	TraceID      string  `json:"traceId"`
	SpanID       string  `json:"spanId"`
	AgentID      *string `json:"agentId,omitempty"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	Input        string  `json:"input"`
	Output       string  `json:"output"`
	StartTime    string  `json:"startTime"`
	EndTime      string  `json:"endTime"`
	LatencyMs    uint32  `json:"latencyMs"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	RetryCount   int32   `json:"retryCount"`
	Metadata     string  `json:"metadata,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

func toToolCallResponse(tc *domain.ToolCall) *ToolCallResponse {
	resp := &ToolCallResponse{
		ID:           tc.ID.String(),
		ProjectID:    tc.ProjectID.String(),
		TraceID:      tc.TraceID.String(),
		SpanID:       tc.SpanID.String(),
		Name:         tc.Name,
		Description:  tc.Description,
		Input:        tc.Input,
		Output:       tc.Output,
		StartTime:    tc.StartTime.Format(time.RFC3339Nano),
		EndTime:      tc.EndTime.Format(time.RFC3339Nano),
		LatencyMs:    tc.LatencyMs,
		Status:       tc.Status,
		ErrorMessage: tc.ErrorMessage,
		RetryCount:   tc.RetryCount,
		Metadata:     tc.Metadata,
		CreatedAt:    tc.CreatedAt.Format(time.RFC3339Nano),
	}

	if tc.AgentID != nil {
		s := tc.AgentID.String()
		resp.AgentID = &s
	}

	return resp
}

// GetToolCallsByTrace retrieves tool calls for a trace
func (h *AgentHandler) GetToolCallsByTrace(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	toolCalls, err := h.agentService.GetToolCallsByTrace(c.Request.Context(), projectID, traceID)
	if err != nil {
		h.logger.Error("failed to get tool calls", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve tool calls",
		})
		return
	}

	responses := make([]*ToolCallResponse, len(toolCalls))
	for i, tc := range toolCalls {
		responses[i] = toToolCallResponse(tc)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": len(responses),
	})
}

// GetToolCallsByAgent retrieves tool calls for an agent
func (h *AgentHandler) GetToolCallsByAgent(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	agentID := c.Param("id")

	if _, err := uuid.Parse(agentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "agent ID must be a valid UUID",
		})
		return
	}

	toolCalls, err := h.agentService.GetToolCallsByAgent(c.Request.Context(), projectID, agentID)
	if err != nil {
		h.logger.Error("failed to get tool calls for agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve tool calls",
		})
		return
	}

	responses := make([]*ToolCallResponse, len(toolCalls))
	for i, tc := range toolCalls {
		responses[i] = toToolCallResponse(tc)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": len(responses),
	})
}

// GetAgentMessages retrieves messages for a trace
func (h *AgentHandler) GetAgentMessages(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	messages, err := h.agentService.GetAgentMessages(c.Request.Context(), projectID, traceID)
	if err != nil {
		h.logger.Error("failed to get agent messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  messages,
		"total": len(messages),
	})
}

// GetAgentStates retrieves state snapshots for an agent
func (h *AgentHandler) GetAgentStates(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	agentID := c.Param("id")

	if _, err := uuid.Parse(agentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "agent ID must be a valid UUID",
		})
		return
	}

	states, err := h.agentService.GetAgentStates(c.Request.Context(), projectID, agentID)
	if err != nil {
		h.logger.Error("failed to get agent states", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve agent states",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  states,
		"total": len(states),
	})
}

// GetAgentGraph builds and returns the agent graph for a trace
func (h *AgentHandler) GetAgentGraph(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	// Check for simplification option
	maxNodes := 0
	if m, err := strconv.Atoi(c.Query("maxNodes")); err == nil && m > 0 {
		maxNodes = m
	}

	var graph *domain.AgentGraph
	var err error

	if maxNodes > 0 {
		graph, err = h.agentService.GetSimplifiedGraph(c.Request.Context(), projectID, traceID, maxNodes)
	} else {
		graph, err = h.agentService.BuildAgentGraph(c.Request.Context(), projectID, traceID)
	}

	if err != nil {
		h.logger.Error("failed to build agent graph", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to build agent graph",
		})
		return
	}

	c.JSON(http.StatusOK, graph.ToJSON())
}

// GetSubgraph returns a subgraph starting from a specific node
func (h *AgentHandler) GetSubgraph(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	traceID := c.Param("traceId")
	nodeID := c.Param("nodeId")

	if _, err := uuid.Parse(traceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "trace ID must be a valid UUID",
		})
		return
	}

	if _, err := uuid.Parse(nodeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "node ID must be a valid UUID",
		})
		return
	}

	maxDepth := 3
	if d, err := strconv.Atoi(c.DefaultQuery("maxDepth", "3")); err == nil && d > 0 && d <= 10 {
		maxDepth = d
	}

	graph, err := h.agentService.GetSubgraph(c.Request.Context(), projectID, traceID, nodeID, maxDepth)
	if err != nil {
		h.logger.Error("failed to get subgraph", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to get subgraph",
		})
		return
	}

	c.JSON(http.StatusOK, graph.ToJSON())
}

// GetAgentStatistics retrieves aggregated agent statistics
func (h *AgentHandler) GetAgentStatistics(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)

	// Parse time range
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if s := c.Query("startTime"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startTime = t
		}
	}
	if e := c.Query("endTime"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			endTime = t
		}
	}

	stats, err := h.agentService.GetAgentStatistics(c.Request.Context(), projectID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get agent statistics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetToolCallStatistics retrieves aggregated tool call statistics
func (h *AgentHandler) GetToolCallStatistics(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)

	// Parse time range
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if s := c.Query("startTime"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startTime = t
		}
	}
	if e := c.Query("endTime"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			endTime = t
		}
	}

	stats, err := h.agentService.GetToolCallStatistics(c.Request.Context(), projectID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get tool call statistics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  stats,
		"total": len(stats),
	})
}
