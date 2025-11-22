package middleware

import (
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a gin middleware for structured logging
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID
		reqID := requestid.Get(c)

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		}

		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("server error", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("client error", fields...)
		default:
			logger.Info("request completed", fields...)
		}
	}
}
