package api

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"audit-platform/internal/model"
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// helper to safely extract tenant_id from Locals.
func getTenantID(c *fiber.Ctx) (string, error) {
	v := c.Locals("tenant_id")
	if v == nil {
		return "", errors.New("tenant_id not set in context")
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.New("tenant_id has wrong type in context")
	}
	return s, nil
}

// LiveWallHandler handles HTTP requests for live stream and TV wall operations.
type LiveWallHandler struct {
	liveSvc   *service.LiveWallService
	wsHub     *service.Hub
	scheduler *service.StreamScheduler
	authSvc   *service.AuthService
}

// NewLiveWallHandler creates a new LiveWallHandler.
func NewLiveWallHandler(liveSvc *service.LiveWallService, wsHub *service.Hub, authSvc *service.AuthService) *LiveWallHandler {
	return &LiveWallHandler{liveSvc: liveSvc, wsHub: wsHub, authSvc: authSvc}
}

// WithScheduler attaches the stream snapshot scheduler.
func (h *LiveWallHandler) WithScheduler(s *service.StreamScheduler) {
	h.scheduler = s
}

// createStreamRequest is the DTO for starting a live stream.
type createStreamRequest struct {
	ContentID  string `json:"content_id"`
	StreamName string `json:"stream_name"`
	StreamKey  string `json:"stream_key"`
	StreamURL  string `json:"stream_url"`
	PlayURL    string `json:"play_url"`
}

// createSnapshotRequest is the DTO for recording a live frame snapshot.
type createSnapshotRequest struct {
	StreamID     string   `json:"stream_id"`
	SnapshotURL  string   `json:"snapshot_url"`
	SnapshotTime string   `json:"snapshot_time"`
	AIRiskScore  int      `json:"ai_risk_score"`
	AIRiskTypes  []string `json:"ai_risk_types"`
	AIConfidence float64  `json:"ai_confidence"`
}

// StartStream handles POST /api/v1/live-streams.
// Generates RTMP push URL and optional WebRTC signaling info.
// Registers the stream with the snapshot scheduler for periodic frame capture.
func (h *LiveWallHandler) StartStream(c *fiber.Ctx) error {
	var req createStreamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid request body",
		})
	}

	tenantIDStr, err := getTenantID(c)
	if err != nil {
		return c.JSON(fiber.Map{"code": 500, "message": "tenant context missing"})
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid tenant_id"})
	}

	contentID, err := uuid.Parse(req.ContentID)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid content_id"})
	}

	streamKey := req.StreamKey
	if streamKey == "" {
		streamKey = uuid.New().String()[:12]
	}
	streamName := req.StreamName
	if streamName == "" {
		streamName = fmt.Sprintf("stream-%s", streamKey)
	}

	// Generate RTMP push URL.
	rtmpPushURL := GenerateRTMPPushURL(streamKey)

	// Generate HLS/FLV play URL.
	playURL := req.PlayURL
	if playURL == "" {
		playURL = fmt.Sprintf("http://localhost:8080/hls/%s/index.m3u8", streamKey)
	}

	stream, err := h.liveSvc.StartStream(c.Context(), tenantID, model.CreateLiveStreamRequest{
		ContentID: contentID,
		StreamKey: streamKey,
		StreamURL: rtmpPushURL,
		PlayURL:   playURL,
	})
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to start stream: " + err.Error(),
		})
	}

	// Register with snapshot scheduler if available.
	if h.scheduler != nil {
		h.scheduler.RegisterStream(stream.ID, contentID, tenantIDStr)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"stream":        stream,
			"rtmp_push_url": rtmpPushURL,
			"stream_key":    streamKey,
			"stream_name":   streamName,
		},
		"message": "Live stream started",
	})
}

// StopStream handles DELETE /api/v1/live-streams/:id.
// Unregisters the stream from the snapshot scheduler.
func (h *LiveWallHandler) StopStream(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid stream id"})
	}

	// Unregister from scheduler before stopping.
	if h.scheduler != nil {
		h.scheduler.UnregisterStream(id)
	}

	if err := h.liveSvc.StopStream(c.Context(), id); err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to stop stream: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    fiber.Map{"status": "offline"},
		"message": "Stream stopped",
	})
}

// GetLiveWall handles GET /api/v1/live-wall — list all active streams with snapshots.
func (h *LiveWallHandler) GetLiveWall(c *fiber.Ctx) error {
	tenantID, _ := getTenantID(c)

	streams, err := h.liveSvc.GetActiveStreams(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to get live wall: " + err.Error(),
		})
	}

	if h.wsHub != nil {
		// Convert to fiber.Map for WebSocket broadcast.
		streamMaps := make([]fiber.Map, len(streams))
		for i, s := range streams {
			streamMaps[i] = fiber.Map{
				"id":        s.ID.String(),
				"content_id": s.ContentID.String(),
				"status":    s.Status,
			}
		}
		h.wsHub.BroadcastLiveWallRefresh(tenantID, streamMaps)
	}

	return c.JSON(fiber.Map{
		"data": streams,
	})
}

// CreateSnapshot handles POST /api/v1/live-streams/:id/snapshot.
func (h *LiveWallHandler) CreateSnapshot(c *fiber.Ctx) error {
	streamID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid stream id"})
	}

	var req createSnapshotRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid request body",
		})
	}

	snapshotTime, _ := time.Parse(time.RFC3339, req.SnapshotTime)

	snapshot, err := h.liveSvc.CreateSnapshot(c.Context(), model.CreateSnapshotRequest{
		StreamID:     streamID,
		SnapshotURL:  req.SnapshotURL,
		SnapshotTime: snapshotTime,
		AIRiskScore:  req.AIRiskScore,
		AIRiskTypes:  req.AIRiskTypes,
		AIConfidence: req.AIConfidence,
	})
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to create snapshot: " + err.Error(),
		})
	}

	// Broadcast to TV wall clients (tenant-isolated).
	tenantID, _ := getTenantID(c)
	h.wsHub.BroadcastSnapshot(tenantID, map[string]interface{}{
		"stream_id":     streamID.String(),
		"snapshot_url":  snapshot.SnapshotURL,
		"ai_risk_score": snapshot.AIRiskScore,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    snapshot,
		"message": "Snapshot recorded",
	})
}

// GetStreamCount handles GET /api/v1/live-wall/count — number of active streams.
func (h *LiveWallHandler) GetStreamCount(c *fiber.Ctx) error {
	tenantID, _ := getTenantID(c)

	count, err := h.liveSvc.CountActiveStreams(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to count streams: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{"active_streams": count},
	})
}

// Register handles WebSocket upgrade for live TV wall.
// Accepts token via Authorization header (not query param) to prevent token leakage in logs.
func (h *LiveWallHandler) Register(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if tokenStr == "" {
		tokenStr = c.Query("token") // legacy fallback, deprecated
	}
	if tokenStr == "" {
		return c.JSON(fiber.Map{
			"code":    401,
			"message": "missing authorization token",
		})
	}
	// Strip "Bearer " prefix if present.
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	claims, err := h.authSvc.ValidateToken(tokenStr)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    401,
			"message": "invalid token",
		})
	}

	tenantID, _ := getTenantID(c)
	if tenantID == "" && claims.TenantID != nil {
		tenantID = claims.TenantID.String()
	}

	client := service.NewWSClient(nil, h.wsHub, tenantID)
	h.wsHub.Register(client, claims.UserID.String(), claims.Role, tenantID)

	// Note: WebSocket connection handling is done via the WebSocket() method.
	// This Register handler is for HTTP API calls.
	return c.JSON(fiber.Map{
		"message": "WebSocket registration prepared",
	})
}

// WebSocket handler for live wall real-time updates.
func (h *LiveWallHandler) WebSocket(c *websocket.Conn) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"missing token"}`))
		c.Close()
		return
	}

	claims, err := h.authSvc.ValidateToken(tokenStr)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"invalid token"}`))
		c.Close()
		return
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" && claims.TenantID != nil {
		tenantID = claims.TenantID.String()
	}

	userID := claims.UserID.String()

	client := service.NewWSClient(c, h.wsHub, tenantID)
	h.wsHub.Register(client, userID, claims.Role, tenantID)

	go client.WritePump()
	go client.ReadPump()
}

// GenerateRTMPPushURL generates a standard RTMP push URL from a stream key.
// Format: rtmp://<host>:1935/live/<stream_key>
func GenerateRTMPPushURL(streamKey string) string {
	host := os.Getenv("RTMP_HOST")
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("rtmp://%s:1935/live/%s", host, streamKey)
}
