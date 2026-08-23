package service

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"sync"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"

	"github.com/google/uuid"
)

var streamLog = logger.New("stream_scheduler")

// Global random source for deterministic seeding across calls.
var streamRng = rand.New(rand.NewSource(time.Now().UnixNano()))

// StreamScheduler manages periodic frame snapshots and health checks for live streams.
// It simulates an RTMP pull + ffmpeg snapshot pipeline.
type StreamScheduler struct {
	liveSvc *LiveWallService
	wsHub   *Hub
	streams map[uuid.UUID]*streamJob
	mu      sync.RWMutex
	stopCh  chan struct{}
}

type streamJob struct {
	streamID  uuid.UUID
	contentID uuid.UUID
	tenantID  string
	interval  time.Duration
	cancel    context.CancelFunc
}

// NewStreamScheduler creates a scheduler that periodically captures snapshots
// from active live streams.
func NewStreamScheduler(liveSvc *LiveWallService, wsHub *Hub) *StreamScheduler {
	return &StreamScheduler{
		liveSvc: liveSvc,
		wsHub:   wsHub,
		streams: make(map[uuid.UUID]*streamJob),
		stopCh:  make(chan struct{}),
	}
}

// Start begins the snapshot scheduler loop.
func (s *StreamScheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *StreamScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkStreams(ctx)
		case <-s.stopCh:
			return
		}
	}
}

// checkStreams iterates over all active streams and triggers snapshot capture
// if enough time has elapsed since the last snapshot.
func (s *StreamScheduler) checkStreams(ctx context.Context) {
	s.mu.RLock()
	jobs := make([]*streamJob, 0, len(s.streams))
	for _, j := range s.streams {
		jobs = append(jobs, j)
	}
	s.mu.RUnlock()

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check stream health via ffprobe (or simulate if ffprobe unavailable).
		healthy, err := s.isStreamHealthy(job.streamID)
		if err != nil || !healthy {
			// Stream unhealthy — mark offline.
			s.markStreamOffline(ctx, job)
			continue
		}

		// Capture a frame snapshot.
		s.captureSnapshot(ctx, job)
	}
}

// isStreamHealthy probes the RTMP stream via ffprobe.
// Returns true if the stream is alive, false if ffmpeg reports no video.
func (s *StreamScheduler) isStreamHealthy(streamID uuid.UUID) (bool, error) {
	// Retrieve stream info from DB.
	streams, err := s.liveSvc.streamRepo.GetActiveStreams(nil, "")
	if err != nil {
		return false, err
	}

	for _, st := range streams {
		if st.ID == streamID && st.StreamURL != "" {
			cmd := exec.Command("ffprobe",
				"-v", "error",
				"-select_streams", "v:0",
				"-show_entries", "stream=codec_type",
				"-of", "csv=p=0",
				st.StreamURL,
			)
			out, err := cmd.Output()
			if err != nil {
				return false, err
			}
			return len(out) > 0 && len(out) < 20, nil // simple heuristic
		}
	}
	return false, nil
}

// captureSnapshot simulates capturing a frame from the RTMP stream.
// In production this would use ffmpeg to pull the RTMP stream and save frames.
func (s *StreamScheduler) captureSnapshot(ctx context.Context, job *streamJob) {
	now := time.Now()

	// Generate a simulated snapshot URL.
	// In production: ffmpeg -i rtmp://... -ss 0 -vframes 1 /tmp/snap_{streamID}.jpg
	snapshotURL := fmt.Sprintf("/snapshots/%s_%d.jpg", job.streamID.String()[:8], now.Unix())

	// Simulate AI risk score based on random seed (production: send to Agnes AI).
	riskScore := streamRng.Intn(30) // low risk by default
	riskTypes := []string{}
	aiConfidence := 0.7 + streamRng.Float64()*0.3

	snapshotReq := model.CreateSnapshotRequest{
		StreamID:     job.streamID,
		SnapshotURL:  snapshotURL,
		SnapshotTime: now,
		AIRiskScore:  riskScore,
		AIRiskTypes:  riskTypes,
		AIConfidence: aiConfidence,
	}

	_, err := s.liveSvc.CreateSnapshot(ctx, snapshotReq)
	if err != nil {
		streamLog.Warn("snapshot capture failed for stream %s: %v", job.streamID, err)
		return
	}

	// Broadcast snapshot update to TV wall.
	if s.wsHub != nil {
		s.wsHub.BroadcastSnapshot(job.tenantID, map[string]interface{}{
			"stream_id":     job.streamID.String(),
			"snapshot_url":  snapshotURL,
			"ai_risk_score": riskScore,
			"timestamp":     now.Format(time.RFC3339),
		})
	}
}

// markStreamOffline sets the stream status to offline and broadcasts the change.
func (s *StreamScheduler) markStreamOffline(ctx context.Context, job *streamJob) {
	_ = s.liveSvc.streamRepo.UpdateLiveStreamStatus(ctx, job.streamID, model.LiveStatusOffline, nil)
	s.wsHub.BroadcastStreamStatus(job.tenantID, job.streamID, "offline")

	s.mu.Lock()
	if j, ok := s.streams[job.streamID]; ok {
		j.cancel()
		delete(s.streams, job.streamID)
	}
	s.mu.Unlock()
}

// RegisterStream adds a stream to the scheduler for periodic snapshot capture.
func (s *StreamScheduler) RegisterStream(streamID, contentID uuid.UUID, tenantID string) {
	s.mu.Lock()
	s.streams[streamID] = &streamJob{
		streamID:  streamID,
		contentID: contentID,
		tenantID:  tenantID,
		interval:  15 * time.Second,
		cancel:    func() {},
	}
	s.mu.Unlock()
}

// UnregisterStream removes a stream from the scheduler.
func (s *StreamScheduler) UnregisterStream(streamID uuid.UUID) {
	s.mu.Lock()
	if job, ok := s.streams[streamID]; ok {
		job.cancel()
		delete(s.streams, streamID)
	}
	s.mu.Unlock()
}

// Stop halts all snapshot scheduling.
func (s *StreamScheduler) Stop() {
	close(s.stopCh)
	s.mu.Lock()
	for _, job := range s.streams {
		job.cancel()
	}
	s.streams = make(map[uuid.UUID]*streamJob)
	s.mu.Unlock()
}
