package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// DashboardHandler handles dashboard-related HTTP requests
type DashboardHandler struct {
	dashboardService *service.DashboardService
	logger           *zap.Logger
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(dashboardService *service.DashboardService, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		logger:           logger,
	}
}

// CreateDashboardRequest represents a dashboard creation request
type CreateDashboardRequest struct {
	ProjectID   string          `json:"projectId" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Layout      json.RawMessage `json:"layout"`
}

// UpdateDashboardRequest represents a dashboard update request
type UpdateDashboardRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Layout      json.RawMessage `json:"layout"`
	IsPublic    bool            `json:"isPublic"`
}

// AddWidgetRequest represents a widget creation request
type AddWidgetRequest struct {
	WidgetType string          `json:"widgetType" binding:"required"`
	Title      string          `json:"title" binding:"required"`
	Config     json.RawMessage `json:"config" binding:"required"`
	Position   json.RawMessage `json:"position" binding:"required"`
}

// UpdateWidgetRequest represents a widget update request
type UpdateWidgetRequest struct {
	WidgetType string          `json:"widgetType"`
	Title      string          `json:"title"`
	Config     json.RawMessage `json:"config"`
	Position   json.RawMessage `json:"position"`
}

// UpdateLayoutRequest represents a layout update request
type UpdateLayoutRequest struct {
	Widgets map[string]json.RawMessage `json:"widgets" binding:"required"`
}

// CreateShareRequest represents a share creation request
type CreateShareRequest struct {
	ExpiresInHours *int `json:"expiresInHours"`
}

// CloneDashboardRequest represents a dashboard clone request
type CloneDashboardRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateDashboard creates a new dashboard
// POST /v1/dashboards
func (h *DashboardHandler) CreateDashboard(c *gin.Context) {
	var req CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_project_id",
			"message": "Invalid project ID format",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	createdBy, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Invalid user ID",
		})
		return
	}

	dashboard, err := h.dashboardService.CreateDashboard(c.Request.Context(), projectID, createdBy, req.Name, req.Description, req.Layout)
	if err != nil {
		h.logger.Error("failed to create dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create dashboard",
		})
		return
	}

	c.JSON(http.StatusCreated, dashboard)
}

// GetDashboard retrieves a dashboard by ID
// GET /v1/dashboards/:id
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	dashboard, widgets, err := h.dashboardService.GetDashboardWithWidgets(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to get dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve dashboard",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dashboard": dashboard,
		"widgets":   widgets,
	})
}

// ListDashboards lists dashboards for a project
// GET /v1/dashboards?projectId=xxx&includeTemplates=true
func (h *DashboardHandler) ListDashboards(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_project_id",
			"message": "Project ID is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_project_id",
			"message": "Invalid project ID format",
		})
		return
	}

	includeTemplates := c.DefaultQuery("includeTemplates", "false") == "true"

	dashboards, err := h.dashboardService.ListDashboards(c.Request.Context(), projectID, includeTemplates)
	if err != nil {
		h.logger.Error("failed to list dashboards", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve dashboards",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dashboards,
	})
}

// UpdateDashboard updates a dashboard
// PUT /v1/dashboards/:id
func (h *DashboardHandler) UpdateDashboard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	var req UpdateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err = h.dashboardService.UpdateDashboard(c.Request.Context(), id, req.Name, req.Description, req.Layout, req.IsPublic)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to update dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update dashboard",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard updated successfully",
	})
}

// DeleteDashboard deletes a dashboard
// DELETE /v1/dashboards/:id
func (h *DashboardHandler) DeleteDashboard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	err = h.dashboardService.DeleteDashboard(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to delete dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete dashboard",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard deleted successfully",
	})
}

// AddWidget adds a widget to a dashboard
// POST /v1/dashboards/:id/widgets
func (h *DashboardHandler) AddWidget(c *gin.Context) {
	dashboardID, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	var req AddWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	widget, err := h.dashboardService.AddWidget(c.Request.Context(), dashboardID, req.WidgetType, req.Title, req.Config, req.Position)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to add widget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to add widget",
		})
		return
	}

	c.JSON(http.StatusCreated, widget)
}

// UpdateWidget updates a widget
// PUT /v1/dashboards/:dashboardId/widgets/:widgetId
func (h *DashboardHandler) UpdateWidget(c *gin.Context) {
	widgetID, err := uuid.Parse(c.Param("widgetId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid widget ID",
		})
		return
	}

	var req UpdateWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err = h.dashboardService.UpdateWidget(c.Request.Context(), widgetID, req.WidgetType, req.Title, req.Config, req.Position)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Widget not found",
			})
			return
		}
		h.logger.Error("failed to update widget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update widget",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Widget updated successfully",
	})
}

// DeleteWidget deletes a widget
// DELETE /v1/dashboards/:dashboardId/widgets/:widgetId
func (h *DashboardHandler) DeleteWidget(c *gin.Context) {
	widgetID, err := uuid.Parse(c.Param("widgetId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid widget ID",
		})
		return
	}

	err = h.dashboardService.DeleteWidget(c.Request.Context(), widgetID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Widget not found",
			})
			return
		}
		h.logger.Error("failed to delete widget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete widget",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Widget deleted successfully",
	})
}

// UpdateLayout updates widget positions (for drag-and-drop)
// PUT /v1/dashboards/:id/layout
func (h *DashboardHandler) UpdateLayout(c *gin.Context) {
	var req UpdateLayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Convert string keys to UUIDs
	updates := make(map[uuid.UUID]json.RawMessage)
	for idStr, position := range req.Widgets {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_widget_id",
				"message": "Invalid widget ID: " + idStr,
			})
			return
		}
		updates[id] = position
	}

	err := h.dashboardService.UpdateWidgetLayout(c.Request.Context(), updates)
	if err != nil {
		h.logger.Error("failed to update layout", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update layout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Layout updated successfully",
	})
}

// CreateShare creates a shareable link for a dashboard
// POST /v1/dashboards/:id/share
func (h *DashboardHandler) CreateShare(c *gin.Context) {
	dashboardID, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid user ID",
		})
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresInHours != nil {
		duration := time.Duration(*req.ExpiresInHours) * time.Hour
		expiresIn = &duration
	}

	share, err := h.dashboardService.CreateShare(c.Request.Context(), dashboardID, userID, expiresIn)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to create share", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create share",
		})
		return
	}

	c.JSON(http.StatusCreated, share)
}

// GetSharedDashboard retrieves a dashboard using a share token
// GET /v1/dashboards/shared/:token
func (h *DashboardHandler) GetSharedDashboard(c *gin.Context) {
	token := c.Param("token")

	dashboard, err := h.dashboardService.GetDashboardByShare(c.Request.Context(), token)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Shared dashboard not found or expired",
			})
			return
		}
		h.logger.Error("failed to get shared dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve shared dashboard",
		})
		return
	}

	widgets, err := h.dashboardService.GetWidgets(c.Request.Context(), dashboard.ID)
	if err != nil {
		h.logger.Error("failed to get widgets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve widgets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dashboard": dashboard,
		"widgets":   widgets,
	})
}

// ListShares lists all shares for a dashboard
// GET /v1/dashboards/:id/shares
func (h *DashboardHandler) ListShares(c *gin.Context) {
	dashboardID, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	shares, err := h.dashboardService.ListShares(c.Request.Context(), dashboardID)
	if err != nil {
		h.logger.Error("failed to list shares", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve shares",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": shares,
	})
}

// DeleteShare deletes a share
// DELETE /v1/dashboards/:dashboardId/shares/:shareId
func (h *DashboardHandler) DeleteShare(c *gin.Context) {
	shareID, err := uuid.Parse(c.Param("shareId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid share ID",
		})
		return
	}

	err = h.dashboardService.DeleteShare(c.Request.Context(), shareID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Share not found",
			})
			return
		}
		h.logger.Error("failed to delete share", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete share",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Share deleted successfully",
	})
}

// CloneDashboard clones a dashboard
// POST /v1/dashboards/:id/clone
func (h *DashboardHandler) CloneDashboard(c *gin.Context) {
	sourceID, err := uuid.Parse(c.Param("dashboardId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid dashboard ID",
		})
		return
	}

	var req CloneDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Get project and user from context
	projectIDStr := c.GetString("project_id")
	userIDStr := c.GetString("user_id")

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_project_id",
			"message": "Invalid project ID",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid user ID",
		})
		return
	}

	dashboard, err := h.dashboardService.CloneDashboard(c.Request.Context(), sourceID, projectID, userID, req.Name)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dashboard not found",
			})
			return
		}
		h.logger.Error("failed to clone dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to clone dashboard",
		})
		return
	}

	c.JSON(http.StatusCreated, dashboard)
}
