package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// PromptHandler handles prompt-related endpoints
type PromptHandler struct {
	promptService *service.PromptService
	logger        *zap.Logger
}

// NewPromptHandler creates a new prompt handler
func NewPromptHandler(promptService *service.PromptService, logger *zap.Logger) *PromptHandler {
	return &PromptHandler{
		promptService: promptService,
		logger:        logger,
	}
}

// List returns all prompts for a project
func (h *PromptHandler) List(c *gin.Context) {
	projectID := c.GetString("project_id")
	if projectID == "" {
		projectID = "default"
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	prompts, total, err := h.promptService.List(c.Request.Context(), projectID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list prompts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   prompts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create creates a new prompt
func (h *PromptHandler) Create(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Content     string   `json:"content"`
		Tags        []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString("project_id")
	if projectID == "" {
		projectID = "default"
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		projectUUID = uuid.New()
	}

	now := time.Now()
	prompt := &domain.Prompt{
		ID:          uuid.New(),
		ProjectID:   projectUUID,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.promptService.Create(c.Request.Context(), prompt); err != nil {
		h.logger.Error("failed to create prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create prompt",
		})
		return
	}

	c.JSON(http.StatusCreated, prompt)
}

// Get returns a single prompt
func (h *PromptHandler) Get(c *gin.Context) {
	id := c.Param("id")
	prompt, err := h.promptService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to get prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}
	c.JSON(http.StatusOK, prompt)
}

// Update updates a prompt
func (h *PromptHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	prompt, err := h.promptService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}

	if req.Name != "" {
		prompt.Name = req.Name
	}
	prompt.Description = req.Description
	if req.Tags != nil {
		prompt.Tags = req.Tags
	}
	prompt.UpdatedAt = time.Now()

	if err := h.promptService.Update(c.Request.Context(), prompt); err != nil {
		h.logger.Error("failed to update prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update prompt",
		})
		return
	}

	c.JSON(http.StatusOK, prompt)
}

// Delete deletes a prompt
func (h *PromptHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.promptService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete prompt",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Prompt deleted"})
}

// ListVersions returns all versions of a prompt
func (h *PromptHandler) ListVersions(c *gin.Context) {
	id := c.Param("id")

	versions, err := h.promptService.ListVersions(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to list versions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve versions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  versions,
		"total": len(versions),
	})
}

// CreateVersion creates a new version of a prompt
func (h *PromptHandler) CreateVersion(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Content string          `json:"content" binding:"required"`
		Config  json.RawMessage `json:"config,omitempty"`
		Labels  []string        `json:"labels,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Get user ID from context if available
	var createdBy *uuid.UUID
	if userIDStr := c.GetString("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			createdBy = &userID
		}
	}

	version, err := h.promptService.CreateVersion(
		c.Request.Context(),
		id,
		req.Content,
		req.Config,
		req.Labels,
		createdBy,
	)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to create version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create version",
		})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetVersion returns a specific version of a prompt
func (h *PromptHandler) GetVersion(c *gin.Context) {
	promptID := c.Param("id")
	versionStr := c.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid version number",
		})
		return
	}

	promptVersion, err := h.promptService.GetVersion(c.Request.Context(), promptID, version)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Version not found",
			})
			return
		}
		h.logger.Error("failed to get version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve version",
		})
		return
	}

	c.JSON(http.StatusOK, promptVersion)
}

// Compile compiles a prompt template with variables
func (h *PromptHandler) Compile(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Variables map[string]interface{} `json:"variables"`
		Version   int                    `json:"version,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	result, err := h.promptService.CompileTemplate(c.Request.Context(), id, req.Version, req.Variables)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt or version not found",
			})
			return
		}
		h.logger.Error("failed to compile template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to compile template",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        id,
		"compiled":  result.Compiled,
		"variables": result.Variables,
		"missing":   result.Missing,
		"errors":    result.Errors,
	})
}

// Duplicate creates a copy of a prompt
func (h *PromptHandler) Duplicate(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	prompt, err := h.promptService.Duplicate(c.Request.Context(), id, req.Name)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to duplicate prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to duplicate prompt",
		})
		return
	}

	c.JSON(http.StatusCreated, prompt)
}

// CompareVersions compares two versions of a prompt
func (h *PromptHandler) CompareVersions(c *gin.Context) {
	promptID := c.Param("id")
	v1Str := c.Query("v1")
	v2Str := c.Query("v2")

	v1, err := strconv.Atoi(v1Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid v1 version number",
		})
		return
	}

	v2, err := strconv.Atoi(v2Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid v2 version number",
		})
		return
	}

	version1, err := h.promptService.GetVersion(c.Request.Context(), promptID, v1)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": fmt.Sprintf("Version %d not found", v1),
			})
			return
		}
		h.logger.Error("failed to get version 1", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve version",
		})
		return
	}

	version2, err := h.promptService.GetVersion(c.Request.Context(), promptID, v2)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": fmt.Sprintf("Version %d not found", v2),
			})
			return
		}
		h.logger.Error("failed to get version 2", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve version",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"promptId": promptID,
		"v1": gin.H{
			"version":   version1.Version,
			"content":   version1.Content,
			"labels":    version1.Labels,
			"createdAt": version1.CreatedAt,
		},
		"v2": gin.H{
			"version":   version2.Version,
			"content":   version2.Content,
			"labels":    version2.Labels,
			"createdAt": version2.CreatedAt,
		},
	})
}

// UpdateVersionLabels updates labels for a specific version
func (h *PromptHandler) UpdateVersionLabels(c *gin.Context) {
	promptID := c.Param("id")
	versionStr := c.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid version number",
		})
		return
	}

	var req struct {
		Labels []string `json:"labels" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.promptService.UpdateVersionLabels(c.Request.Context(), promptID, version, req.Labels); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Version not found",
			})
			return
		}
		h.logger.Error("failed to update version labels", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update version labels",
		})
		return
	}

	// Return the updated version
	promptVersion, _ := h.promptService.GetVersion(c.Request.Context(), promptID, version)
	c.JSON(http.StatusOK, promptVersion)
}

// ExtractVariables extracts variables from a template content
func (h *PromptHandler) ExtractVariables(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Use regexp to extract variables from {{variable}} syntax
	variables := extractTemplateVariables(req.Content)

	c.JSON(http.StatusOK, gin.H{
		"variables": variables,
	})
}

// extractTemplateVariables extracts variable names from template content
// Supports {{variable}}, {{#if variable}}, {{#each variable}}, etc.
func extractTemplateVariables(content string) []string {
	// Match simple variables: {{variable}}
	simpleVarRe := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match if conditions: {{#if variable}}
	ifVarRe := regexp.MustCompile(`\{\{#if\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match unless conditions: {{#unless variable}}
	unlessVarRe := regexp.MustCompile(`\{\{#unless\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match each loops: {{#each variable}}
	eachVarRe := regexp.MustCompile(`\{\{#each\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match with blocks: {{#with variable}}
	withVarRe := regexp.MustCompile(`\{\{#with\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

	seen := make(map[string]bool)
	var variables []string

	addMatches := func(re *regexp.Regexp) {
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && !seen[match[1]] {
				seen[match[1]] = true
				variables = append(variables, match[1])
			}
		}
	}

	addMatches(simpleVarRe)
	addMatches(ifVarRe)
	addMatches(unlessVarRe)
	addMatches(eachVarRe)
	addMatches(withVarRe)

	return variables
}

// PromoteVersion promotes a version to a specific environment
func (h *PromptHandler) PromoteVersion(c *gin.Context) {
	promptID := c.Param("id")
	versionStr := c.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid version number",
		})
		return
	}

	var req struct {
		Target string `json:"target" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.promptService.PromoteVersion(c.Request.Context(), promptID, version, req.Target); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Version not found",
			})
			return
		}
		h.logger.Error("failed to promote version", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Return the updated version
	promptVersion, _ := h.promptService.GetVersion(c.Request.Context(), promptID, version)
	c.JSON(http.StatusOK, promptVersion)
}

// GetVersionByLabel gets the version with a specific label
func (h *PromptHandler) GetVersionByLabel(c *gin.Context) {
	promptID := c.Param("id")
	label := c.Param("label")

	version, err := h.promptService.GetVersionByLabel(c.Request.Context(), promptID, label)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": fmt.Sprintf("No version found with label '%s'", label),
			})
			return
		}
		h.logger.Error("failed to get version by label", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve version",
		})
		return
	}

	c.JSON(http.StatusOK, version)
}

// GetLinkedTraces returns traces that used this prompt
func (h *PromptHandler) GetLinkedTraces(c *gin.Context) {
	promptID := c.Param("id")

	// Verify prompt exists
	_, err := h.promptService.GetByID(c.Request.Context(), promptID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to get prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}

	// For now, return a placeholder response
	// In a full implementation, this would query ClickHouse for traces with this prompt_id
	c.JSON(http.StatusOK, gin.H{
		"promptId": promptID,
		"traces":   []interface{}{},
		"total":    0,
		"message":  "Traces linked to this prompt will appear here when SDK sends prompt_id with traces",
	})
}

// GetAnalytics returns usage analytics for a prompt
func (h *PromptHandler) GetAnalytics(c *gin.Context) {
	promptID := c.Param("id")

	// Get prompt to verify it exists
	prompt, err := h.promptService.GetByID(c.Request.Context(), promptID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to get prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}

	// Get all versions
	versions, err := h.promptService.ListVersions(c.Request.Context(), promptID)
	if err != nil {
		h.logger.Error("failed to list versions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve versions",
		})
		return
	}

	// Build version stats
	versionStats := make([]gin.H, len(versions))
	var productionVersion, stagingVersion, developmentVersion *int
	for i, v := range versions {
		versionStats[i] = gin.H{
			"version":   v.Version,
			"labels":    v.Labels,
			"createdAt": v.CreatedAt,
		}
		// Track which versions have environment labels
		for _, label := range v.Labels {
			switch label {
			case "production":
				productionVersion = &v.Version
			case "staging":
				stagingVersion = &v.Version
			case "development":
				developmentVersion = &v.Version
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"promptId":           promptID,
		"promptName":         prompt.Name,
		"totalVersions":      len(versions),
		"latestVersion":      len(versions),
		"productionVersion":  productionVersion,
		"stagingVersion":     stagingVersion,
		"developmentVersion": developmentVersion,
		"versions":           versionStats,
		"createdAt":          prompt.CreatedAt,
		"updatedAt":          prompt.UpdatedAt,
	})
}
