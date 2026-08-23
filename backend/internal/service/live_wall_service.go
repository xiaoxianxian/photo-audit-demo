package service

import (
	"context"
	"fmt"
	"time"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

// LiveWallService manages live streams and frame snapshots for the TV wall.
type LiveWallService struct {
	streamRepo *repository.LiveWallRepository
}

// NewLiveWallService creates a new LiveWallService.
func NewLiveWallService(streamRepo *repository.LiveWallRepository) *LiveWallService {
	return &LiveWallService{streamRepo: streamRepo}
}

// StartStream registers a new live stream and sets its status to streaming.
func (s *LiveWallService) StartStream(ctx context.Context, tenantID uuid.UUID, req model.CreateLiveStreamRequest) (*model.LiveStream, error) {
	now := time.Now()
	stream := &model.LiveStream{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ContentID:   req.ContentID,
		StreamKey:   req.StreamKey,
		StreamURL:   req.StreamURL,
		PlayURL:     req.PlayURL,
		Status:      model.LiveStatusStreaming,
		ViewerCount: 0,
		StartedAt:   &now,
	}

	if err := s.streamRepo.CreateLiveStream(ctx, stream); err != nil {
		return nil, fmt.Errorf("start live stream: %w", err)
	}
	return stream, nil
}

// StopStream sets a stream's status to offline.
func (s *LiveWallService) StopStream(ctx context.Context, id uuid.UUID) error {
	return s.streamRepo.UpdateLiveStreamStatus(ctx, id, model.LiveStatusOffline, nil)
}

// GetActiveStreams returns all active (streaming) streams for a tenant with their latest snapshots.
func (s *LiveWallService) GetActiveStreams(ctx context.Context, tenantID string) ([]model.LiveStream, error) {
	streams, err := s.streamRepo.GetActiveStreams(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get active streams: %w", err)
	}

	if len(streams) == 0 {
		return streams, nil
	}

	// Collect stream IDs for snapshot lookup.
	streamIDs := make([]uuid.UUID, 0, len(streams))
	for _, st := range streams {
		streamIDs = append(streamIDs, st.ID)
	}

	snapshots, err := s.streamRepo.GetLatestSnapshots(ctx, streamIDs)
	if err != nil {
		return nil, fmt.Errorf("get latest snapshots: %w", err)
	}

	// Attach latest snapshot to each stream.
	snapMap := make(map[uuid.UUID]model.LiveFrameSnapshot)
	for _, sn := range snapshots {
		snapMap[sn.StreamID] = sn
	}
	for i := range streams {
		if snap, ok := snapMap[streams[i].ID]; ok {
			streams[i].StreamURL = snap.SnapshotURL
			streams[i].ViewerCount = int(snap.AIConfidence * 1000) // rough proxy for demo
		}
	}

	return streams, nil
}

// CreateSnapshot records a new frame snapshot for a live stream.
func (s *LiveWallService) CreateSnapshot(ctx context.Context, req model.CreateSnapshotRequest) (*model.LiveFrameSnapshot, error) {
	snapshot := &model.LiveFrameSnapshot{
		ID:           uuid.New(),
		StreamID:     req.StreamID,
		SnapshotURL:  req.SnapshotURL,
		SnapshotTime: req.SnapshotTime,
		AIRiskScore:  req.AIRiskScore,
		AIRiskTypes:  req.AIRiskTypes,
		AIConfidence: req.AIConfidence,
	}

	if err := s.streamRepo.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	return snapshot, nil
}

// CountActiveStreams returns the number of active streams for a tenant.
func (s *LiveWallService) CountActiveStreams(ctx context.Context, tenantID string) (int64, error) {
	return s.streamRepo.CountActiveStreams(ctx, tenantID)
}
