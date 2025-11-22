package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sqlx.DB, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services,omitempty"`
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
		if err := h.db.Ping(); err != nil {
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
