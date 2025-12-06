package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/config"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
	orgService  *service.OrgService
	cfg         *config.AuthConfig
	logger      *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, orgService *service.OrgService, cfg *config.AuthConfig, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		orgService:  orgService,
		cfg:         cfg,
		logger:      logger,
	}
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required,min=1"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	ExpiresAt int64         `json:"expiresAt"`
	User      *UserResponse `json:"user"`
	CSRFToken string        `json:"csrfToken,omitempty"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		h.logger.Error("registration failed", zap.Error(err), zap.String("email", req.Email))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "registration_failed",
			"message": err.Error(),
		})
		return
	}

	// Generate tokens and set cookies
	expiresAt, err := h.setAuthCookies(c, user.ID.String(), "", user.Email, "member")
	if err != nil {
		h.logger.Error("token generation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate authentication tokens",
		})
		return
	}

	// Get CSRF token for response
	csrfToken, _ := c.Cookie("csrf_token")

	c.JSON(http.StatusCreated, AuthResponse{
		ExpiresAt: expiresAt,
		User: &UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			Name:      user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
		CSRFToken: csrfToken,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Warn("login failed", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Get default organization for the user (if any)
	orgID := ""
	role := "member"

	// Get user's organizations
	orgs, _, err := h.orgService.ListUserOrganizations(c.Request.Context(), user.ID.String(), 1, 0)
	if err != nil {
		h.logger.Warn("failed to get user organizations", zap.Error(err))
	} else if len(orgs) > 0 {
		orgID = orgs[0].ID.String()
		// Get the user's role in this organization
		if userRole, err := h.orgService.GetUserOrgRole(c.Request.Context(), orgID, user.ID.String()); err == nil {
			role = userRole
		}
	}

	// Generate tokens and set cookies
	expiresAt, err := h.setAuthCookies(c, user.ID.String(), orgID, user.Email, role)
	if err != nil {
		h.logger.Error("token generation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate authentication tokens",
		})
		return
	}

	// Get CSRF token for response
	csrfToken, _ := c.Cookie("csrf_token")

	c.JSON(http.StatusOK, AuthResponse{
		ExpiresAt: expiresAt,
		User: &UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			Name:      user.Name,
			AvatarURL: user.AvatarURL,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
		CSRFToken: csrfToken,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_token",
			"message": "No refresh token provided",
		})
		return
	}

	// Validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		// Clear invalid cookies
		h.clearAuthCookies(c)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_token",
			"message": "Invalid or expired refresh token",
		})
		return
	}

	claims, ok := token.Claims.(*middleware.Claims)
	if !ok {
		// Clear invalid cookies
		h.clearAuthCookies(c)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_token",
			"message": "Invalid token claims",
		})
		return
	}

	// Generate new tokens and set cookies
	expiresAt, err := h.setAuthCookies(c, claims.UserID, claims.OrganizationID, claims.Email, claims.Role)
	if err != nil {
		h.logger.Error("token refresh failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to refresh tokens",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"expiresAt": expiresAt,
	})
}

// Me returns the current user's profile
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	user, err := h.authService.UpdateProfile(c.Request.Context(), userID, req.Name, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update profile",
		})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

// ChangePassword changes the current user's password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "password_change_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// Stub handlers for organizations and projects
func (h *AuthHandler) ListOrganizations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
}

func (h *AuthHandler) CreateOrganization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) GetOrganization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) UpdateOrganization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) DeleteOrganization(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) ListMembers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
}

func (h *AuthHandler) AddMember(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) RemoveMember(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) ListProjects(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
}

func (h *AuthHandler) CreateProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) GetProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) UpdateProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

func (h *AuthHandler) DeleteProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// CreateAPIKeyRequest represents the request to create an API key
type CreateAPIKeyRequest struct {
	Name      string   `json:"name" binding:"required"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expiresAt"`
}

func (h *AuthHandler) ListAPIKeys(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	apiKeys, err := h.authService.ListAPIKeys(c.Request.Context(), projectID)
	if err != nil {
		h.logger.Error("failed to list API keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  apiKeys,
		"total": len(apiKeys),
	})
}

func (h *AuthHandler) CreateAPIKey(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default scopes if not provided
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"trace:write", "prompt:read", "guardrail:evaluate"}
	}

	// Parse expiration time if provided
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiration time format"})
			return
		}
		expiresAt = &t
	}

	apiKey, rawKey, err := h.authService.CreateAPIKey(c.Request.Context(), projectID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		h.logger.Error("failed to create API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        apiKey.ID.String(),
		"key":       rawKey, // Only shown once
		"keyPrefix": apiKey.KeyPrefix,
		"message":   "Save this key securely. It will not be shown again.",
	})
}

func (h *AuthHandler) RevokeAPIKey(c *gin.Context) {
	keyIDStr := c.Param("keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key ID"})
		return
	}

	if err := h.authService.RevokeAPIKey(c.Request.Context(), keyID); err != nil {
		h.logger.Error("failed to revoke API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// Logout handles user logout by clearing cookies
func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// setAuthCookies generates tokens and sets them as secure HTTP-only cookies
func (h *AuthHandler) setAuthCookies(c *gin.Context, userID, orgID, email, role string) (int64, error) {
	now := time.Now()
	expiresAt := now.Add(h.cfg.JWTExpiration)

	// Access token
	claims := &middleware.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "otelguard",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return 0, err
	}

	// Refresh token (longer expiry)
	refreshClaims := &middleware.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(h.cfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "otelguard",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return 0, err
	}

	// Generate CSRF token
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		return 0, err
	}

	// Set secure cookies with proper attributes
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// Access token cookie (short-lived, HTTP-only)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    tokenString,
		MaxAge:   int(h.cfg.JWTExpiration.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Refresh token cookie (long-lived, HTTP-only)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		MaxAge:   int(h.cfg.RefreshTokenExpiry.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// CSRF token cookie (same lifetime as session, readable by JS)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		MaxAge:   int(h.cfg.JWTExpiration.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: false, // Allow JS to read for CSRF protection
		SameSite: http.SameSiteLaxMode,
	})

	return expiresAt.Unix(), nil
}

// clearAuthCookies removes authentication cookies
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// Clear access token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear refresh token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear CSRF token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

// generateTokens creates JWT access and refresh tokens (deprecated - use setAuthCookies instead)
func (h *AuthHandler) generateTokens(userID, orgID, email, role string) (string, string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(h.cfg.JWTExpiration)

	// Access token
	claims := &middleware.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "otelguard",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return "", "", 0, err
	}

	// Refresh token (longer expiry)
	refreshClaims := &middleware.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(h.cfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "otelguard",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return "", "", 0, err
	}

	return tokenString, refreshTokenString, expiresAt.Unix(), nil
}
