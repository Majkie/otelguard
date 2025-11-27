package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub    *service.WebSocketHub
	logger *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *service.WebSocketHub, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// ServeWS handles WebSocket requests from clients
// @Summary WebSocket endpoint for real-time updates
// @Tags websocket
// @Param projectId query string true "Project ID"
// @Router /ws [get]
func (h *WebSocketHandler) ServeWS(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_project_id",
			Message: "project_id query parameter is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Invalid project ID format",
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	// Create client
	client := &service.Client{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Conn:      conn,
		Send:      make(chan *service.WebSocketEvent, 256),
		Hub:       h.hub,
	}
	client.InitSubscriptions()

	// Register client
	h.hub.register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}
