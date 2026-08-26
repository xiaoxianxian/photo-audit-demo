package api

import (
	"time"

	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
)

// SignalingHandler implements WHIP (publish) / WHEP (view) signaling endpoints.
// Media flows P2P between publisher and viewer; these endpoints only relay
// SDP offer/answer bodies.
type SignalingHandler struct {
	hub *service.SignalingHub
}

// NewSignalingHandler creates the handler with a fresh hub (5min session TTL).
func NewSignalingHandler() *SignalingHandler {
	return &SignalingHandler{hub: service.NewSignalingHub(5 * time.Minute)}
}

// WhipPublish handles POST /api/v1/webrtc/whip/:streamKey — a publisher
// submits its SDP offer and blocks (up to 60s) until a viewer's answer
// arrives, which is returned as the response body.
func (h *SignalingHandler) WhipPublish(c *fiber.Ctx) error {
	streamKey := c.Params("streamKey")
	if streamKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": 400, "message": "missing streamKey"})
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 500, "message": "tenant context missing"})
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": 400, "message": "empty SDP offer"})
	}

	answer, err := h.hub.Publish(c.Context(), streamKey, tenantID, body, 60*time.Second)
	if err != nil {
		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"code":    504,
			"message": "no viewer answered in time",
		})
	}
	c.Set("Content-Type", "application/sdp")
	return c.Send(answer)
}

// WhepView handles POST /api/v1/webrtc/whep/:streamKey — a viewer submits
// its SDP answer and receives the publisher's pending offer. Two-step flow:
// clients may first GET /whep/:streamKey to read the offer, produce an
// answer locally, then POST it here.
func (h *SignalingHandler) WhepView(c *fiber.Ctx) error {
	streamKey := c.Params("streamKey")
	if streamKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": 400, "message": "missing streamKey"})
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 500, "message": "tenant context missing"})
	}

	body := c.Body()

	offer, err := h.hub.View(streamKey, tenantID, body)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": 404, "message": "no active publisher for this stream"})
	}
	c.Set("Content-Type", "application/sdp")
	return c.Send(offer)
}

// WhepPeek handles GET /api/v1/webrtc/whep/:streamKey — returns the pending
// SDP offer without consuming it (step one of the two-step WHEP flow).
func (h *SignalingHandler) WhepPeek(c *fiber.Ctx) error {
	streamKey := c.Params("streamKey")
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 500, "message": "tenant context missing"})
	}
	offer, err := h.hub.Peek(streamKey, tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": 404, "message": "no active publisher for this stream"})
	}
	c.Set("Content-Type", "application/sdp")
	return c.Send(offer)
}

// WhipDelete handles DELETE /api/v1/webrtc/whip/:streamKey — publisher ends
// the session.
func (h *SignalingHandler) WhipDelete(c *fiber.Ctx) error {
	h.hub.End(c.Params("streamKey"))
	return c.SendStatus(fiber.StatusOK)
}
