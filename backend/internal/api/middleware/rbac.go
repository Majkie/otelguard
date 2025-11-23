package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
)

// RBACMiddleware provides role-based access control middleware
type RBACMiddleware struct {
	orgService *service.OrgService
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(orgService *service.OrgService) *RBACMiddleware {
	return &RBACMiddleware{orgService: orgService}
}

// RequireOrgRole ensures the user has at least the specified role in the organization
func (m *RBACMiddleware) RequireOrgRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(string(ContextUserID))
		orgID := c.Param("orgId")

		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		if orgID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "organization ID required",
			})
			return
		}

		role, err := m.orgService.GetUserOrgRole(c.Request.Context(), orgID, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "not a member of this organization",
			})
			return
		}

		// Check if user's role is in the allowed roles
		allowed := false
		for _, r := range roles {
			if role == r {
				allowed = true
				break
			}
		}

		// Also allow higher roles (owner > admin > member > viewer)
		if !allowed {
			roleHierarchy := map[string]int{
				domain.RoleOwner:  4,
				domain.RoleAdmin:  3,
				domain.RoleMember: 2,
				domain.RoleViewer: 1,
			}
			userLevel := roleHierarchy[role]
			for _, r := range roles {
				if userLevel >= roleHierarchy[r] {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
			return
		}

		// Store role in context for later use
		c.Set(string(ContextRole), role)
		c.Set(string(ContextOrganizationID), orgID)
		c.Next()
	}
}

// RequireProjectPermission ensures the user has the specified permission for the project
func (m *RBACMiddleware) RequireProjectPermission(perm domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(string(ContextUserID))
		projectID := c.Param("projectId")

		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		if projectID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "project ID required",
			})
			return
		}

		hasPermission, err := m.orgService.CanUserAccessProject(c.Request.Context(), projectID, userID, perm)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "failed to check permissions",
			})
			return
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
			return
		}

		c.Set(string(ContextProjectID), projectID)
		c.Next()
	}
}

// RequireOrgPermission ensures the user has the specified permission in the organization
func (m *RBACMiddleware) RequireOrgPermission(perm domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(string(ContextUserID))
		orgID := c.Param("orgId")

		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			return
		}

		if orgID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "organization ID required",
			})
			return
		}

		role, err := m.orgService.GetUserOrgRole(c.Request.Context(), orgID, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "not a member of this organization",
			})
			return
		}

		if !domain.HasPermission(role, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
			return
		}

		c.Set(string(ContextRole), role)
		c.Set(string(ContextOrganizationID), orgID)
		c.Next()
	}
}

// RequireRole ensures the user has at least one of the specified roles
// This is a simpler version that works with roles set in the JWT token
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString(string(ContextRole))

		if userRole == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "no role assigned",
			})
			return
		}

		// Check if user's role is in the allowed roles
		allowed := false
		for _, r := range roles {
			if userRole == r {
				allowed = true
				break
			}
		}

		// Also allow higher roles
		if !allowed {
			roleHierarchy := map[string]int{
				domain.RoleOwner:  4,
				domain.RoleAdmin:  3,
				domain.RoleMember: 2,
				domain.RoleViewer: 1,
			}
			userLevel := roleHierarchy[userRole]
			for _, r := range roles {
				if userLevel >= roleHierarchy[r] {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}
