package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// WSMessage represents a JSON message sent over WebSocket.
type WSMessage struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// WSConnection represents a registered WebSocket client with metadata.
type WSConnection struct {
	Client   *WSClient
	UserID   string
	Role     string
	TenantID string
}

// Hub manages WebSocket connections for both live TV wall and review task distribution.
// It broadcasts snapshot updates to TV wall clients and new review tasks to reviewers.
type Hub struct {
	connections map[*WSClient]*WSConnection // client -> metadata
	register    chan *registerMsg
	unregister  chan *WSClient
	mu          sync.RWMutex
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[*WSClient]*WSConnection),
		register:    make(chan *registerMsg, 100),
		unregister:  make(chan *WSClient, 100),
	}
}

// WSClient represents a single WebSocket connection.
type WSClient struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	tenant string
}

// NewWSClient creates a new WSClient with the given connection and hub.
func NewWSClient(conn *websocket.Conn, hub *Hub, tenant string) *WSClient {
	return &WSClient{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		tenant: tenant,
	}
}

// ReadPump reads messages from the WebSocket connection.
func (c *WSClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump writes messages to the WebSocket connection.
func (c *WSClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message := <-c.send:
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Register adds a client to the hub with role metadata.
func (h *Hub) Register(client *WSClient, userID, role, tenantID string) {
	h.register <- &registerMsg{
		client:   client,
		userID:   userID,
		role:     role,
		tenantID: tenantID,
	}
}

type registerMsg struct {
	client   *WSClient
	userID   string
	role     string
	tenantID string
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *WSClient) {
	h.unregister <- client
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case msg := <-h.register:
			h.mu.Lock()
			h.connections[msg.client] = &WSConnection{
				Client:   msg.client,
				UserID:   msg.userID,
				Role:     msg.role,
				TenantID: msg.tenantID,
			}
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[client]; ok {
				delete(h.connections, client)
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastSnapshot sends a snapshot update only to clients in a specific tenant.
// Previously this broadcast to ALL tenants, which was a cross-tenant data leak.
func (h *Hub) BroadcastSnapshot(tenantID string, payload map[string]interface{}) {
	msg, _ := json.Marshal(WSMessage{
		Type:      "snapshot_update",
		Payload:   payload,
		Timestamp: time.Now(),
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	for client, conn := range h.connections {
		if conn.TenantID == tenantID {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.connections, client)
			}
		}
	}
}

// BroadcastToTenant sends a message only to clients in a specific tenant.
func (h *Hub) BroadcastToTenant(tenantID string, msgType string, payload map[string]interface{}) {
	msg, _ := json.Marshal(WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	for client, conn := range h.connections {
		if conn.TenantID == tenantID {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.connections, client)
			}
		}
	}
}

// BroadcastToReviewers sends a message only to clients with reviewer role in a tenant.
func (h *Hub) BroadcastToReviewers(tenantID string, msgType string, payload map[string]interface{}) {
	msg, _ := json.Marshal(WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	for client, conn := range h.connections {
		if conn.TenantID == tenantID && (conn.Role == "reviewer" || conn.Role == "tenant_admin" || conn.Role == "platform_admin") {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.connections, client)
			}
		}
	}
}

// BroadcastStreamStatus notifies all clients in a tenant about a stream status change.
func (h *Hub) BroadcastStreamStatus(tenantID string, streamID uuid.UUID, status string) {
	payload := map[string]interface{}{
		"stream_id": streamID.String(),
		"status":    status,
	}
	h.BroadcastSnapshot(tenantID, payload)
}

// BroadcastLiveWallRefresh sends a full refresh signal to all TV wall clients in a tenant.
func (h *Hub) BroadcastLiveWallRefresh(tenantID string, streams []fiber.Map) {
	h.BroadcastSnapshot(tenantID, map[string]interface{}{
		"streams": streams,
	})
}

// BroadcastNewTask notifies all reviewers in a tenant about new pending elements.
func (h *Hub) BroadcastNewTask(tenantID string, contentID string, elementCount int) {
	h.BroadcastToReviewers(tenantID, "new_task", map[string]interface{}{
		"content_id":    contentID,
		"element_count": elementCount,
		"message":       fmt.Sprintf("新内容待审：%d 个元素", elementCount),
	})
}

// Stop closes all client connections and stops the hub goroutine.
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.connections {
		close(client.send)
		client.conn.Close()
	}
	h.connections = make(map[*WSClient]*WSConnection)
}
