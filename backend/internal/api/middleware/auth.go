package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

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
	ContextProjectID      ContextKey = "project_id"
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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			return
		}

		// Check for Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid authorization header format",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

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
		c.Set(string(ContextProjectID), claims.ProjectID.String())
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
