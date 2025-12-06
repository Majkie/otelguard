package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

// APIKeyClaims represents claims extracted from an API key
type APIKeyClaims struct {
	ProjectID uuid.UUID
	Scopes    []string
}

// ContextKey type for context keys
type ContextKey string

const (
	// Context keys
	ContextUserID         ContextKey = "user_id"
	ContextOrganizationID ContextKey = "organization_id"
	ContextProjectID      string     = "project_id"
	ContextEmail          ContextKey = "email"
	ContextRole           ContextKey = "role"
	ContextScopes         ContextKey = "scopes"
	ContextAuthType       ContextKey = "auth_type"
)

// AuthType constants
const (
	AuthTypeJWT    = "jwt"
	AuthTypeAPIKey = "api_key"
)

// APIKeyValidator is a function type for validating API keys
type APIKeyValidator func(keyHash string) (*APIKeyClaims, error)

// JWTAuth returns middleware for JWT authentication
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fall back to cookie
			var err error
			tokenString, err = c.Cookie("access_token")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "unauthorized",
					"message": "missing authentication token",
				})
				return
			}
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
			})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid token claims",
			})
			return
		}

		// Set claims in context
		c.Set(string(ContextUserID), claims.UserID)
		c.Set(string(ContextOrganizationID), claims.OrganizationID)
		c.Set(string(ContextEmail), claims.Email)
		c.Set(string(ContextRole), claims.Role)
		c.Set(string(ContextAuthType), AuthTypeJWT)

		c.Next()
	}
}

// APIKeyAuth returns middleware for API key authentication
func APIKeyAuth(salt string, validator APIKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try Authorization header with "Api-Key" prefix
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Api-Key ") {
				apiKey = strings.TrimPrefix(authHeader, "Api-Key ")
			}
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing API key",
			})
			return
		}

		// Hash the API key
		keyHash := HashAPIKey(apiKey, salt)

		// Validate with the provided validator
		claims, err := validator(keyHash)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid API key",
			})
			return
		}

		// Set claims in context
		c.Set(ContextProjectID, claims.ProjectID.String())
		c.Set(string(ContextScopes), claims.Scopes)
		c.Set(string(ContextAuthType), AuthTypeAPIKey)

		c.Next()
	}
}

// CombinedAuth allows either JWT or API key authentication
func CombinedAuth(jwtSecret, apiKeySalt string, apiKeyValidator APIKeyValidator) gin.HandlerFunc {
	jwtAuth := JWTAuth(jwtSecret)
	apiKeyAuth := APIKeyAuth(apiKeySalt, apiKeyValidator)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		apiKeyHeader := c.GetHeader("X-API-Key")

		// Check for API key first
		if apiKeyHeader != "" || strings.HasPrefix(authHeader, "Api-Key ") {
			apiKeyAuth(c)
			return
		}

		// Fall back to JWT
		jwtAuth(c)
	}
}

// SetProjectContext extracts project_id from query parameters and sets it in context
func SetProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Query("projectId")
		if projectID != "" {
			c.Set(ContextProjectID, projectID)
		}
		c.Next()
	}
}

// RequireScope checks if the current auth has the required scope
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString(string(ContextAuthType))

		// JWT users have all scopes
		if authType == AuthTypeJWT {
			c.Next()
			return
		}

		// Check API key scopes
		scopes, exists := c.Get(string(ContextScopes))
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
			return
		}

		scopeList, ok := scopes.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "invalid scopes",
			})
			return
		}

		// Check if scope is present or if wildcard scope is present
		for _, s := range scopeList {
			if s == scope || s == "*" {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "insufficient permissions for scope: " + scope,
		})
	}
}

// HashAPIKey creates a SHA-256 hash of an API key with salt
func HashAPIKey(key, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + key))
	return hex.EncodeToString(h.Sum(nil))
}

// SecureCompare performs a constant-time comparison
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateCSRFToken generates a random CSRF token
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CSRFProtection middleware validates CSRF tokens for state-changing operations
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF check for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get CSRF token from header
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			// Also check for token in form data for compatibility
			csrfToken = c.PostForm("csrf_token")
		}

		if csrfToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_required",
				"message": "CSRF token required for this operation",
			})
			return
		}

		// For now, we'll store CSRF tokens in session-like cookies
		// In production, you'd want to store them in Redis or similar
		expectedToken, err := c.Cookie("csrf_token")
		if err != nil || !SecureCompare(csrfToken, expectedToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_invalid",
				"message": "Invalid CSRF token",
			})
			return
		}

		c.Next()
	}
}

// AutoRefreshAuth returns middleware that automatically refreshes tokens when close to expiry
func AutoRefreshAuth(secret string, refreshThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fall back to cookie
			var err error
			tokenString, err = c.Cookie("access_token")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "unauthorized",
					"message": "missing authentication token",
				})
				return
			}
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			// Try to refresh the token automatically
			if refreshErr := refreshTokenIfNeeded(c, secret); refreshErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "unauthorized",
					"message": "invalid or expired token",
				})
				return
			}

			// Re-parse the token after refresh
			tokenString, _ = c.Cookie("access_token")
			token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "unauthorized",
					"message": "token refresh failed",
				})
				return
			}
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid token claims",
			})
			return
		}

		// Check if token needs refresh (proactive refresh)
		if time.Until(claims.ExpiresAt.Time) < refreshThreshold {
			// Try to refresh in background (don't block the request)
			go func() {
				// Note: This is a simplified version. In production, you'd want to use a worker queue
				// or some other mechanism to handle concurrent refreshes properly
				_ = refreshTokenIfNeeded(c.Copy(), secret)
			}()
		}

		// Set claims in context
		c.Set(string(ContextUserID), claims.UserID)
		c.Set(string(ContextOrganizationID), claims.OrganizationID)
		c.Set(string(ContextEmail), claims.Email)
		c.Set(string(ContextRole), claims.Role)
		c.Set(string(ContextAuthType), AuthTypeJWT)

		c.Next()
	}
}

// refreshTokenIfNeeded attempts to refresh the access token using the refresh token
func refreshTokenIfNeeded(c *gin.Context, secret string) error {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		return err
	}

	// Validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return jwt.ErrTokenInvalidClaims
	}

	// Generate new tokens
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // Default expiration, should come from config

	newClaims := &Claims{
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		Email:          claims.Email,
		Role:           claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "otelguard",
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := newToken.SignedString([]byte(secret))
	if err != nil {
		return err
	}

	// Set new access token cookie
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    tokenString,
		MaxAge:   int(24 * time.Hour.Seconds()),
		Path:     "/",
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}
