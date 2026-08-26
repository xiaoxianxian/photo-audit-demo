package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// whipSession is one pending WebRTC signaling exchange for a stream key.
// A publisher posts an SDP offer (WHIP), viewers fetch it and post an SDP
// answer; the offer/answer pair is then delivered back to both sides.
type whipSession struct {
	id        uuid.UUID
	streamKey string
	tenantID  string
	offer     []byte // SDP offer body from publisher
	answerCh  chan []byte
	createdAt time.Time
}

// SignalingHub coordinates WHIP (publish) / WHEP (view) SDP exchanges.
// In-memory only: sessions live until the publisher disconnects or the TTL
// expires. Media never flows through the server — this is pure signaling.
type SignalingHub struct {
	mu       sync.RWMutex
	sessions map[string]*whipSession // streamKey -> session
	ttl      time.Duration
}

// NewSignalingHub creates a signaling hub with the given session TTL.
func NewSignalingHub(ttl time.Duration) *SignalingHub {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &SignalingHub{
		sessions: make(map[string]*whipSession),
		ttl:      ttl,
	}
}

// Publish stores a WHIP offer and waits up to waitTimeout for a WHEP answer.
// Returns the viewer's SDP answer once it arrives.
func (h *SignalingHub) Publish(ctx context.Context, streamKey, tenantID string, offer []byte, waitTimeout time.Duration) ([]byte, error) {
	if waitTimeout <= 0 {
		waitTimeout = 30 * time.Second
	}
	h.mu.Lock()
	if h.sessions == nil {
		h.sessions = make(map[string]*whipSession)
	}
	sess := &whipSession{
		id:        uuid.New(),
		streamKey: streamKey,
		tenantID:  tenantID,
		offer:     append([]byte(nil), offer...),
		answerCh:  make(chan []byte, 1),
		createdAt: time.Now(),
	}
	h.sessions[streamKey] = sess
	h.mu.Unlock()

	ctx2, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()
	select {
	case ans := <-sess.answerCh:
		return ans, nil
	case <-ctx2.Done():
		// Clean up our own session on timeout so stale offers don't linger.
		h.mu.Lock()
		if cur, ok := h.sessions[streamKey]; ok && cur == sess {
			delete(h.sessions, streamKey)
		}
		h.mu.Unlock()
		return nil, ctx2.Err()
	}
}

// View fetches the pending offer for a stream key and registers a WHEP
// answer. Returns ErrNoOffer when no publisher is waiting or the tenant
// does not match.
func (h *SignalingHub) View(streamKey, tenantID string, answer []byte) ([]byte, error) {
	h.mu.RLock()
	sess, ok := h.sessions[streamKey]
	h.mu.RUnlock()
	if !ok || time.Since(sess.createdAt) > h.ttl || sess.offer == nil {
		return nil, ErrNoOffer
	}
	if sess.tenantID != tenantID {
		return nil, ErrNoOffer
	}
	select {
	case sess.answerCh <- answer:
	default:
		return nil, ErrNoOffer
	}
	return sess.offer, nil
}

// Peek returns the pending offer without consuming it (used by WHEP clients
// that need the offer before producing an answer).
func (h *SignalingHub) Peek(streamKey, tenantID string) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sess, ok := h.sessions[streamKey]
	if !ok || time.Since(sess.createdAt) > h.ttl || sess.offer == nil {
		return nil, ErrNoOffer
	}
	if sess.tenantID != tenantID {
		return nil, ErrNoOffer
	}
	return sess.offer, nil
}

// Answer delivers a WHEP answer to the waiting publisher without needing to
// fetch the offer first (two-step WHEP flow).
func (h *SignalingHub) Answer(streamKey, tenantID string, answer []byte) error {
	h.mu.RLock()
	sess, ok := h.sessions[streamKey]
	h.mu.RUnlock()
	if !ok || time.Since(sess.createdAt) > h.ttl {
		return ErrNoOffer
	}
	if sess.tenantID != tenantID {
		return ErrNoOffer
	}
	select {
	case sess.answerCh <- answer:
		return nil
	default:
		return ErrNoOffer
	}
}

// End removes a session (publisher stopped).
func (h *SignalingHub) End(streamKey string) {
	h.mu.Lock()
	delete(h.sessions, streamKey)
	h.mu.Unlock()
}

// Errors for signaling flows.
var (
	ErrNoOffer = errors.New("no pending offer for stream key")
)
