package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *pgxpool.Pool, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Services map[string]string `json:"services,omitempty"`
}

// Health handles the health check endpoint
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
	})
}

// Ready handles the readiness check endpoint
// @Summary Readiness check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	services := make(map[string]string)
	allHealthy := true

	// Check PostgreSQL connection
	if h.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := h.db.Ping(ctx); err != nil {
			h.logger.Error("postgres health check failed", zap.Error(err))
			services["postgres"] = "unhealthy"
			allHealthy = false
		} else {
			services["postgres"] = "healthy"
		}
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:   status,
		Version:  "1.0.0",
		Services: services,
	})
}
