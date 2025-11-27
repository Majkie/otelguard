package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketEvent represents an event sent over WebSocket
type WebSocketEvent struct {
	Type      string      `json:"type"`      // 'trace_created', 'trace_updated', 'span_created', 'agent_detected', etc.
	ProjectID uuid.UUID   `json:"project_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID            string
	ProjectID     uuid.UUID
	Conn          *websocket.Conn
	Send          chan *WebSocketEvent
	Hub           *WebSocketHub
	mu            sync.Mutex
	subscriptions map[string]bool // subscription filters: trace_id, session_id, etc.
}

// InitSubscriptions initializes the subscriptions map
func (c *Client) InitSubscriptions() {
	c.subscriptions = make(map[string]bool)
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	// Registered clients by project ID
	clients map[uuid.UUID]map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast events
	broadcast chan *WebSocketEvent

	// Mutex for thread-safe operations
	mu sync.RWMutex

	logger *zap.Logger
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *zap.Logger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[uuid.UUID]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WebSocketEvent, 256),
		logger:     logger,
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case event := <-h.broadcast:
			h.broadcastEvent(event)

		case <-ticker.C:
			h.cleanupStaleConnections()

		case <-ctx.Done():
			h.logger.Info("websocket hub shutting down")
			h.closeAllConnections()
			return
		}
	}
}

// registerClient registers a new client
func (h *WebSocketHub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ProjectID]; !exists {
		h.clients[client.ProjectID] = make(map[string]*Client)
	}

	h.clients[client.ProjectID][client.ID] = client

	h.logger.Info("client connected",
		zap.String("client_id", client.ID),
		zap.String("project_id", client.ProjectID.String()),
	)
}

// unregisterClient unregisters a client
func (h *WebSocketHub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if projectClients, exists := h.clients[client.ProjectID]; exists {
		if _, exists := projectClients[client.ID]; exists {
			delete(projectClients, client.ID)
			close(client.Send)

			if len(projectClients) == 0 {
				delete(h.clients, client.ProjectID)
			}

			h.logger.Info("client disconnected",
				zap.String("client_id", client.ID),
				zap.String("project_id", client.ProjectID.String()),
			)
		}
	}
}

// broadcastEvent broadcasts an event to all relevant clients
func (h *WebSocketHub) broadcastEvent(event *WebSocketEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	projectClients, exists := h.clients[event.ProjectID]
	if !exists {
		return
	}

	for _, client := range projectClients {
		// Check if client should receive this event based on subscriptions
		if client.shouldReceiveEvent(event) {
			select {
			case client.Send <- event:
				// Event sent successfully
			default:
				// Client's send channel is full, skip this event
				h.logger.Warn("client send channel full, dropping event",
					zap.String("client_id", client.ID),
					zap.String("event_type", event.Type),
				)
			}
		}
	}
}

// cleanupStaleConnections removes connections that are no longer active
func (h *WebSocketHub) cleanupStaleConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for projectID, projectClients := range h.clients {
		for clientID, client := range projectClients {
			// Try to ping the client
			client.mu.Lock()
			err := client.Conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(10*time.Second),
			)
			client.mu.Unlock()

			if err != nil {
				h.logger.Warn("client ping failed, removing",
					zap.String("client_id", clientID),
					zap.Error(err),
				)
				delete(projectClients, clientID)
				close(client.Send)
			}
		}

		if len(projectClients) == 0 {
			delete(h.clients, projectID)
		}
	}
}

// closeAllConnections closes all WebSocket connections
func (h *WebSocketHub) closeAllConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for projectID, projectClients := range h.clients {
		for clientID, client := range projectClients {
			close(client.Send)
			client.Conn.Close()
			h.logger.Info("closed client connection",
				zap.String("client_id", clientID),
				zap.String("project_id", projectID.String()),
			)
		}
	}

	h.clients = make(map[uuid.UUID]map[string]*Client)
}

// BroadcastEvent broadcasts an event to all clients in a project
func (h *WebSocketHub) BroadcastEvent(event *WebSocketEvent) {
	select {
	case h.broadcast <- event:
		// Event queued successfully
	default:
		// Broadcast channel is full
		h.logger.Warn("broadcast channel full, dropping event",
			zap.String("event_type", event.Type),
		)
	}
}

// shouldReceiveEvent determines if a client should receive an event
func (c *Client) shouldReceiveEvent(event *WebSocketEvent) bool {
	// If no subscriptions, receive all events
	if len(c.subscriptions) == 0 {
		return true
	}

	// Check if event matches any subscription filters
	// This can be extended with more sophisticated filtering
	for filter := range c.subscriptions {
		if filter == event.Type {
			return true
		}
	}

	return false
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.Send:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			err := c.Conn.WriteJSON(event)
			c.mu.Unlock()

			if err != nil {
				c.Hub.logger.Error("write error", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()

			if err != nil {
				return
			}
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("unexpected close error", zap.Error(err))
			}
			break
		}

		// Handle subscription messages
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "subscribe":
				c.handleSubscribe(msg)
			case "unsubscribe":
				c.handleUnsubscribe(msg)
			case "ping":
				c.handlePing()
			}
		}
	}
}

// handleSubscribe handles subscription requests
func (c *Client) handleSubscribe(msg map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if filters, ok := msg["filters"].([]interface{}); ok {
		for _, filter := range filters {
			if filterStr, ok := filter.(string); ok {
				c.subscriptions[filterStr] = true
			}
		}

		c.Hub.logger.Info("client subscribed to filters",
			zap.String("client_id", c.ID),
			zap.Any("filters", filters),
		)
	}
}

// handleUnsubscribe handles unsubscription requests
func (c *Client) handleUnsubscribe(msg map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if filters, ok := msg["filters"].([]interface{}); ok {
		for _, filter := range filters {
			if filterStr, ok := filter.(string); ok {
				delete(c.subscriptions, filterStr)
			}
		}

		c.Hub.logger.Info("client unsubscribed from filters",
			zap.String("client_id", c.ID),
			zap.Any("filters", filters),
		)
	}
}

// handlePing handles ping messages from client
func (c *Client) handlePing() {
	c.mu.Lock()
	defer c.mu.Unlock()

	pongMsg := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now(),
	}

	data, err := json.Marshal(pongMsg)
	if err != nil {
		return
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

// TraceEventPublisher publishes trace events to WebSocket clients
type TraceEventPublisher struct {
	hub    *WebSocketHub
	logger *zap.Logger
}

// NewTraceEventPublisher creates a new trace event publisher
func NewTraceEventPublisher(hub *WebSocketHub, logger *zap.Logger) *TraceEventPublisher {
	return &TraceEventPublisher{
		hub:    hub,
		logger: logger,
	}
}

// PublishTraceCreated publishes a trace created event
func (p *TraceEventPublisher) PublishTraceCreated(projectID uuid.UUID, trace interface{}) {
	event := &WebSocketEvent{
		Type:      "trace_created",
		ProjectID: projectID,
		Timestamp: time.Now(),
		Data:      trace,
	}

	p.hub.BroadcastEvent(event)
}

// PublishTraceUpdated publishes a trace updated event
func (p *TraceEventPublisher) PublishTraceUpdated(projectID uuid.UUID, trace interface{}) {
	event := &WebSocketEvent{
		Type:      "trace_updated",
		ProjectID: projectID,
		Timestamp: time.Now(),
		Data:      trace,
	}

	p.hub.BroadcastEvent(event)
}

// PublishSpanCreated publishes a span created event
func (p *TraceEventPublisher) PublishSpanCreated(projectID uuid.UUID, span interface{}) {
	event := &WebSocketEvent{
		Type:      "span_created",
		ProjectID: projectID,
		Timestamp: time.Now(),
		Data:      span,
	}

	p.hub.BroadcastEvent(event)
}

// PublishAgentDetected publishes an agent detected event
func (p *TraceEventPublisher) PublishAgentDetected(projectID uuid.UUID, agent interface{}) {
	event := &WebSocketEvent{
		Type:      "agent_detected",
		ProjectID: projectID,
		Timestamp: time.Now(),
		Data:      agent,
	}

	p.hub.BroadcastEvent(event)
}

// PublishAlertFired publishes an alert fired event
func (p *TraceEventPublisher) PublishAlertFired(projectID uuid.UUID, alert interface{}) {
	event := &WebSocketEvent{
		Type:      "alert_fired",
		ProjectID: projectID,
		Timestamp: time.Now(),
		Data:      alert,
	}

	p.hub.BroadcastEvent(event)
}
