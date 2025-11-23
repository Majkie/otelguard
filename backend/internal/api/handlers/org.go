package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// OrgHandler handles organization and project endpoints
type OrgHandler struct {
	orgService *service.OrgService
	logger     *zap.Logger
}

// NewOrgHandler creates a new organization handler
func NewOrgHandler(orgService *service.OrgService, logger *zap.Logger) *OrgHandler {
	return &OrgHandler{
		orgService: orgService,
		logger:     logger,
	}
}

// OrganizationResponse represents an organization in API responses
type OrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// MemberResponse represents a member in API responses
type MemberResponse struct {
	ID        string        `json:"id"`
	UserID    string        `json:"userId"`
	Role      string        `json:"role"`
	CreatedAt string        `json:"createdAt"`
	User      *UserResponse `json:"user,omitempty"`
}

// SessionResponse represents a session in API responses
type SessionResponse struct {
	ID           string `json:"id"`
	UserAgent    string `json:"userAgent,omitempty"`
	IPAddress    string `json:"ipAddress,omitempty"`
	LastActiveAt string `json:"lastActiveAt"`
	CreatedAt    string `json:"createdAt"`
	ExpiresAt    string `json:"expiresAt"`
}

// ListOrganizations lists organizations for the current user
func (h *OrgHandler) ListOrganizations(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orgs, total, err := h.orgService.ListUserOrganizations(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list organizations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list organizations",
		})
		return
	}

	response := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		response[i] = OrganizationResponse{
			ID:        org.ID.String(),
			Name:      org.Name,
			Slug:      org.Slug,
			CreatedAt: org.CreatedAt.Format(time.RFC3339),
			UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateOrganization creates a new organization
func (h *OrgHandler) CreateOrganization(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	org, err := h.orgService.CreateOrganization(c.Request.Context(), userID, req.Name)
	if err != nil {
		h.logger.Error("failed to create organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create organization",
		})
		return
	}

	c.JSON(http.StatusCreated, OrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

// GetOrganization retrieves an organization by ID
func (h *OrgHandler) GetOrganization(c *gin.Context) {
	orgID := c.Param("orgId")

	org, err := h.orgService.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Organization not found",
		})
		return
	}

	c.JSON(http.StatusOK, OrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

// UpdateOrganization updates an organization
func (h *OrgHandler) UpdateOrganization(c *gin.Context) {
	orgID := c.Param("orgId")

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	org, err := h.orgService.UpdateOrganization(c.Request.Context(), orgID, req.Name)
	if err != nil {
		h.logger.Error("failed to update organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update organization",
		})
		return
	}

	c.JSON(http.StatusOK, OrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
		UpdatedAt: org.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteOrganization deletes an organization
func (h *OrgHandler) DeleteOrganization(c *gin.Context) {
	orgID := c.Param("orgId")

	if err := h.orgService.DeleteOrganization(c.Request.Context(), orgID); err != nil {
		h.logger.Error("failed to delete organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete organization",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted"})
}

// ListMembers lists members of an organization
func (h *OrgHandler) ListMembers(c *gin.Context) {
	orgID := c.Param("orgId")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	members, total, err := h.orgService.ListOrganizationMembers(c.Request.Context(), orgID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list members",
		})
		return
	}

	response := make([]MemberResponse, len(members))
	for i, m := range members {
		response[i] = MemberResponse{
			ID:        m.ID.String(),
			UserID:    m.UserID.String(),
			Role:      m.Role,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AddMember adds a user to an organization
func (h *OrgHandler) AddMember(c *gin.Context) {
	orgID := c.Param("orgId")

	var req struct {
		UserID string `json:"userId" binding:"required"`
		Role   string `json:"role" binding:"required,oneof=admin member viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.orgService.AddOrganizationMember(c.Request.Context(), orgID, req.UserID, req.Role); err != nil {
		h.logger.Error("failed to add member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to add member",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Member added"})
}

// RemoveMember removes a user from an organization
func (h *OrgHandler) RemoveMember(c *gin.Context) {
	orgID := c.Param("orgId")
	userID := c.Param("userId")

	if err := h.orgService.RemoveOrganizationMember(c.Request.Context(), orgID, userID); err != nil {
		h.logger.Error("failed to remove member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to remove member",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// ListProjects lists projects for the current user
func (h *OrgHandler) ListProjects(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	projects, total, err := h.orgService.ListUserProjects(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list projects", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list projects",
		})
		return
	}

	response := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		response[i] = ProjectResponse{
			ID:             p.ID.String(),
			OrganizationID: p.OrganizationID.String(),
			Name:           p.Name,
			Slug:           p.Slug,
			CreatedAt:      p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateProject creates a new project
func (h *OrgHandler) CreateProject(c *gin.Context) {
	var req struct {
		OrganizationID string `json:"organizationId" binding:"required"`
		Name           string `json:"name" binding:"required,min=1,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	project, err := h.orgService.CreateProject(c.Request.Context(), req.OrganizationID, req.Name)
	if err != nil {
		h.logger.Error("failed to create project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create project",
		})
		return
	}

	c.JSON(http.StatusCreated, ProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		Slug:           project.Slug,
		CreatedAt:      project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      project.UpdatedAt.Format(time.RFC3339),
	})
}

// GetProject retrieves a project by ID
func (h *OrgHandler) GetProject(c *gin.Context) {
	projectID := c.Param("projectId")

	project, err := h.orgService.GetProject(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		Slug:           project.Slug,
		CreatedAt:      project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      project.UpdatedAt.Format(time.RFC3339),
	})
}

// UpdateProject updates a project
func (h *OrgHandler) UpdateProject(c *gin.Context) {
	projectID := c.Param("projectId")

	var req struct {
		Name     string `json:"name"`
		Settings []byte `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	project, err := h.orgService.UpdateProject(c.Request.Context(), projectID, req.Name, req.Settings)
	if err != nil {
		h.logger.Error("failed to update project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update project",
		})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		Slug:           project.Slug,
		CreatedAt:      project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      project.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteProject deletes a project
func (h *OrgHandler) DeleteProject(c *gin.Context) {
	projectID := c.Param("projectId")

	if err := h.orgService.DeleteProject(c.Request.Context(), projectID); err != nil {
		h.logger.Error("failed to delete project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete project",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// RequestPasswordReset initiates a password reset flow
func (h *OrgHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Don't reveal if email exists
	token, _ := h.orgService.RequestPasswordReset(c.Request.Context(), req.Email)

	// In production, send email with reset link
	// For development, log the token
	if token != "" {
		h.logger.Info("password reset token generated",
			zap.String("email", req.Email),
			zap.String("token", token),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account exists with this email, a reset link has been sent",
	})
}

// ResetPassword resets a user's password using a reset token
func (h *OrgHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.orgService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "reset_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}

// ListSessions lists active sessions for the current user
func (h *OrgHandler) ListSessions(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, total, err := h.orgService.ListUserSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list sessions",
		})
		return
	}

	response := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		response[i] = SessionResponse{
			ID:           s.ID.String(),
			UserAgent:    s.UserAgent,
			IPAddress:    s.IPAddress,
			LastActiveAt: s.LastActiveAt.Format(time.RFC3339),
			CreatedAt:    s.CreatedAt.Format(time.RFC3339),
			ExpiresAt:    s.ExpiresAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// RevokeSession revokes a specific session
func (h *OrgHandler) RevokeSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	if err := h.orgService.RevokeSession(c.Request.Context(), sessionID); err != nil {
		h.logger.Error("failed to revoke session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to revoke session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

// RevokeAllSessions revokes all sessions for the current user
func (h *OrgHandler) RevokeAllSessions(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	if err := h.orgService.RevokeAllUserSessions(c.Request.Context(), userID); err != nil {
		h.logger.Error("failed to revoke sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to revoke sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked"})
}
