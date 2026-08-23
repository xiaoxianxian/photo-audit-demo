package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

var ingestionLog = logger.New("ingestion_service")

// ErrContentNotFound is returned when a content item cannot be found.
var ErrContentNotFound = fmt.Errorf("content not found")

// IngestionService orchestrates content creation, element splitting, and
// triggering AI review for newly ingested items.
type IngestionService struct {
	contentRepo  *repository.ContentRepository
	elementRepo  *repository.ElementRepository
	aiSvc        *AIService
	videoProc    *VideoProcessor
	wsHub        *Hub
	contentMu    sync.Map // contentID -> *sync.Mutex for per-content serialization
}

// NewIngestionService creates a new IngestionService.
func NewIngestionService(contentRepo *repository.ContentRepository, elementRepo *repository.ElementRepository, aiSvc *AIService, wsHub *Hub) *IngestionService {
	return &IngestionService{
		contentRepo: contentRepo,
		elementRepo: elementRepo,
		aiSvc:       aiSvc,
		wsHub:       wsHub,
	}
}

// WithVideoProcessor attaches a VideoProcessor for video-specific preprocessing.
func (s *IngestionService) WithVideoProcessor(vp *VideoProcessor) {
	s.videoProc = vp
}

// VideoProcessor returns the attached VideoProcessor, or nil if not configured.
func (s *IngestionService) VideoProcessor() *VideoProcessor {
	return s.videoProc
}

// WithHub attaches the WebSocket hub for task notifications.
func (s *IngestionService) WithHub(hub *Hub) {
	s.wsHub = hub
}

// UploadContent creates a Content record with the given metadata, generates
// ContentElements from the file URLs, and returns both the content and its
// elements. Elements are inserted via a transactional bulk call.
//
// For short_video content, if a VideoProcessor is attached, it also processes
// the video to generate video_frame and asr_text elements.
func (s *IngestionService) UploadContent(ctx context.Context, input model.UploadInput) (*model.Content, []model.ContentElement, error) {
	// 1. Create the Content record.
	content := &model.Content{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ContentType:  input.ContentType,
		ReviewPolicy: input.ReviewPolicy,
		AIRiskScore:  0,
		Status:       "pending",
		CreatorID:    input.CreatorID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, nil, fmt.Errorf("upload content: %w", err)
	}

	// 2. Split file URLs into content elements.
	elements := make([]model.ContentElement, 0, len(input.FileURLs)+2)

	switch input.ContentType {
	case model.ContentTypeShortVideo:
		// For video content, use the first URL as cover/thumbnail.
		if len(input.FileURLs) > 0 {
			cover := model.ContentElement{
				ID:             uuid.New(),
				ContentID:      content.ID,
				ElementKind:    model.ElementCoverImage,
				ElementContent: input.FileURLs[0],
				AIRiskScore:    0,
				AIStatus:       model.ElementPendingAI,
				HumanStatus:    model.ElementPendingHuman,
				CreatedAt:      time.Now(),
			}
			elements = append(elements, cover)
		}

		// If a video processor is available, extract frames and ASR from the raw video data.
		// The raw video bytes are expected to be in the second URL (original_url).
		if s.videoProc != nil && len(input.FileURLs) > 1 {
			videoElements, err := s.processVideoFromURL(ctx, input.FileURLs[1], content.ID)
			if err != nil {
				ingestionLog.Warn("video preprocessing failed: %v (continuing with basic elements)", err)
			} else {
				elements = append(elements, videoElements...)
			}
		}

	default:
		// Cover/thumbnail element from the first URL.
		if len(input.FileURLs) > 0 {
			cover := model.ContentElement{
				ID:             uuid.New(),
				ContentID:      content.ID,
				ElementKind:    model.ElementCoverImage,
				ElementContent: input.FileURLs[0],
				AIRiskScore:    0,
				AIStatus:       model.ElementPendingAI,
				HumanStatus:    model.ElementPendingHuman,
				CreatedAt:      time.Now(),
			}
			elements = append(elements, cover)
		}

		// Additional file URLs become review elements.
		for i, url := range input.FileURLs {
			if i == 0 {
				continue // already covered
			}
			kind := model.ElementCoverImage
			switch input.ContentType {
			case model.ContentTypeShortVideo:
				kind = model.ElementVideoFrame
			case model.ContentTypeLiveStream:
				kind = model.ElementLiveSnapshot
			}

			elem := model.ContentElement{
				ID:             uuid.New(),
				ContentID:      content.ID,
				ElementKind:    kind,
				ElementContent: url,
				AIRiskScore:    0,
				AIStatus:       model.ElementPendingAI,
				HumanStatus:    model.ElementPendingHuman,
				CreatedAt:      time.Now(),
			}
			elements = append(elements, elem)
		}
	}

	// Title element if provided.
	if input.Title != "" {
		titleElem := model.ContentElement{
			ID:             uuid.New(),
			ContentID:      content.ID,
			ElementKind:    model.ElementTitle,
			ElementContent: input.Title,
			AIRiskScore:    0,
			AIStatus:       model.ElementPendingAI,
			HumanStatus:    model.ElementPendingHuman,
			CreatedAt:      time.Now(),
		}
		elements = append(elements, titleElem)
	}

	// Description element if provided.
	if input.Description != "" {
		descElem := model.ContentElement{
			ID:             uuid.New(),
			ContentID:      content.ID,
			ElementKind:    model.ElementDescription,
			ElementContent: input.Description,
			AIRiskScore:    0,
			AIStatus:       model.ElementPendingAI,
			HumanStatus:    model.ElementPendingHuman,
			CreatedAt:      time.Now(),
		}
		elements = append(elements, descElem)
	}

	// 3. Bulk-insert elements within a transaction (atomic).
	if len(elements) > 0 {
		ptrs := make([]*model.ContentElement, len(elements))
		for i := range elements {
			ptrs[i] = &elements[i]
		}
		if err := s.elementRepo.CreateBulk(ctx, ptrs); err != nil {
			return nil, nil, fmt.Errorf("upload content bulk insert elements: %w", err)
		}
	}

	return content, elements, nil
}

// isPrivateIP checks whether the given IP address is a private/local address
// that should be blocked to prevent SSRF attacks.
func isPrivateIP(addr net.IP) bool {
	// Block loopback, link-local, private, and unspecified addresses.
	if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
		return true
	}
	privatePrefixes := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("100.64.0.0"), Mask: net.CIDRMask(10, 32)}, // carrier-grade NAT
	}
	for _, prefix := range privatePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// validateURL checks that the URL is safe to fetch (prevents SSRF).
// Only https/http schemes are allowed, and the target must not resolve to
// a private or local IP address. Redirects are blocked.
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q (allowed: http, https)", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname in URL %q", rawURL)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("no IP addresses resolved for %q", host)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("blocked private IP %s for URL %q (SSRF prevention)", ip, rawURL)
		}
	}
	return nil
}

// processVideoFromURL fetches a video from a URL, preprocesses it (frames + ASR),
// and returns ContentElements. The raw video bytes are cached in memory.
func (s *IngestionService) processVideoFromURL(ctx context.Context, videoURL string, contentID uuid.UUID) ([]model.ContentElement, error) {
	// Validate URL to prevent SSRF attacks.
	if err := validateURL(videoURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	// Create an HTTP client that does NOT follow redirects to prevent redirect-based SSRF.
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("redirects are not allowed (SSRF prevention)")
		},
	}

	// Fetch the video file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch video: status %d", resp.StatusCode)
	}

	// Limit response body size to 500MB to prevent memory exhaustion.
	limitReader := io.LimitReader(resp.Body, 500*1024*1024)

	videoData, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("read video: %w", err)
	}

	// Run video preprocessing.
	elements, err := s.videoProc.ProcessVideo(ctx, videoData, "video.mp4")
	if err != nil {
		return nil, fmt.Errorf("process video: %w", err)
	}

	// Assign content_id to all generated elements.
	for i := range elements {
		elements[i].ContentID = contentID
	}

	return elements, nil
}

// TriggerAIReview sends all pending elements to Agnes AI asynchronously.
// If AI is unavailable, elements remain in pending_ai status for manual review.
// After AI review completes, if elements are now pending human review, a WebSocket
// notification is broadcast to online reviewers in the tenant.
func (s *IngestionService) TriggerAIReview(ctx context.Context, contentID uuid.UUID, tenantID string) {
	if s.aiSvc == nil {
		return
	}

	elements, err := s.elementRepo.FindByContentID(ctx, contentID)
	if err != nil {
		ingestionLog.Warn("trigger AI review: find elements: %v", err)
		return
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(elements))

	// Bounded concurrency: max 10 concurrent AI review goroutines.
	sem := make(chan struct{}, 10)

	for i := range elements {
		elem := &elements[i]
		if elem.AIStatus != model.ElementPendingAI {
			continue
		}
		wg.Add(1)
		go func(e *model.ContentElement) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			// Mark as processing.
			if err := s.elementRepo.UpdateStatus(ctx, e.ID, string(model.ElementAIProcessing), string(e.HumanStatus)); err != nil {
				ingestionLog.Warn("AI review mark processing: %v", err)
				return
			}

			// Call Agnes AI.
			aiRecord, err := s.aiSvc.ReviewElement(ctx, *e)
			if err != nil {
				// AI failed — fall back to pending human review.
				ingestionLog.Warn("AI review failed for element %s: %v, falling back to human review", e.ID, err)
				_ = s.elementRepo.UpdateStatus(ctx, e.ID, string(model.ElementPendingHuman), string(e.HumanStatus))
				return
			}

			// Update element with AI results.
			aiStatus := model.ElementAIPassed
			if aiRecord.Action == model.ActionReject {
				aiStatus = model.ElementAIRejected
			}

			// Convert reason to risk types slice.
			riskTypes := make([]string, 0)
			if aiRecord.Reason != nil {
				riskTypes = append(riskTypes, string(*aiRecord.Reason))
			}

			if err := s.elementRepo.UpdateAIRisk(ctx, e.ID, *aiRecord.AIScoreAfter, riskTypes, 0.9); err != nil {
				ingestionLog.Warn("AI update element risk: %v", err)
				return
			}
			if err := s.elementRepo.UpdateStatus(ctx, e.ID, string(aiStatus), string(e.HumanStatus)); err != nil {
				ingestionLog.Warn("AI update element status: %v", err)
				return
			}

			// Send to judge model for consistency check.
			judgeRecord, err := s.aiSvc.JudgeReview(ctx, aiRecord)
			if err != nil {
				ingestionLog.Warn("AI judge review failed: %v", err)
				return
			}

			if judgeRecord.IsConflict {
				// Mark conflict on the element.
				if err := s.elementRepo.MarkConflict(ctx, e.ID, true); err != nil {
					ingestionLog.Warn("AI mark conflict: %v", err)
				}
				errCh <- fmt.Errorf("conflict detected: primary=%d, judge=%d", *aiRecord.AIScoreAfter, *judgeRecord.AIScoreAfter)
			}
		}(elem)
	}

	wg.Wait()
	close(errCh)

	// Collect errors (non-blocking — just log).
	var conflicts []error
	for err := range errCh {
		if err != nil {
			conflicts = append(conflicts, err)
		}
	}
	if len(conflicts) > 0 {
		ingestionLog.Info("AI review completed for content %s: %d conflicts", contentID, len(conflicts))
	}

	// After AI review, notify online reviewers if elements need human review.
	if s.wsHub != nil && tenantID != "" {
		go func() {
			stats, err := s.elementRepo.CountByFilters(ctx, string(model.ElementAIPassed), string(model.ElementPendingHuman), "", 0, 100)
			if err == nil && stats.PendingHuman > 0 {
				s.wsHub.BroadcastNewTask(tenantID, contentID.String(), int(stats.PendingHuman))
			}
		}()
	}
}

// TriggerContentDecision evaluates all elements of a content item and determines
// the overall content-level status using a multi-dimensional decision engine:
//
//  1. Mandatory reject: any single element with human_rejected AND risk_score >= 70
//  2. Conflict escalation: any element with is_conflict=true → force "under_review"
//  3. Voting mechanism: majority human_rejected → reject
//  4. AI risk threshold: average AI risk_score > 60 → reject (even without human review)
//  5. Weighted scoring: cover_image and live_snapshot count 2x toward decision
//  6. Default: if all human-done and no rejection → approve; otherwise "pending"
func (s *IngestionService) TriggerContentDecision(ctx context.Context, contentID uuid.UUID) error {
	// Per-content lock to prevent concurrent decision races.
	muIface, _ := s.contentMu.LoadOrStore(contentID, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	elements, err := s.elementRepo.FindByContentID(ctx, contentID)
	if err != nil {
		return fmt.Errorf("trigger decision: %w", err)
	}

	if len(elements) == 0 {
		return nil
	}

	// --- Phase 1: mandatory reject checks (single element overrides) ---

	// Rule 1: any human_rejected element with high AI risk forces content rejection.
	for _, e := range elements {
		if e.HumanStatus == model.ElementHumanRejected && e.AIRiskScore >= 70 {
			return s.contentRepo.UpdateStatus(ctx, contentID, "rejected")
		}
	}

	// Rule 2: any unresolved conflict forces content into in_human_review.
	hasUnresolvedConflict := false
	for _, e := range elements {
		if e.IsConflict && e.HumanStatus != model.ElementHumanPassed && e.HumanStatus != model.ElementHumanRejected {
			hasUnresolvedConflict = true
		}
	}
	if hasUnresolvedConflict {
		return s.contentRepo.UpdateStatus(ctx, contentID, "in_human_review")
	}

	// --- Phase 2: collect reviewable elements (human_done + AI scored) ---

	type weightedElement struct {
		elem       model.ContentElement
		weight     int // 1 = normal, 2 = high-weight element type
		rejected   bool
		passed     bool
		aiRisk     int
	}

	reviewable := make([]weightedElement, 0)
	totalWeight := 0

	for _, e := range elements {
		// Skip elements not yet reviewed by humans.
		if e.HumanStatus != model.ElementHumanPassed && e.HumanStatus != model.ElementHumanRejected {
			continue
		}

		w := 1
		switch e.ElementKind {
		case model.ElementCoverImage, model.ElementLiveSnapshot:
			w = 2 // cover and live snapshots carry more weight
		}

		reviewable = append(reviewable, weightedElement{
			elem:     e,
			weight:   w,
			rejected: e.HumanStatus == model.ElementHumanRejected,
			passed:   e.HumanStatus == model.ElementHumanPassed,
			aiRisk:   e.AIRiskScore,
		})
		totalWeight += w
	}

	// --- Phase 3: voting mechanism (weighted majority) ---

	if len(reviewable) > 0 {
		rejectWeight := 0
		for _, we := range reviewable {
			if we.rejected {
				rejectWeight += we.weight
			}
		}
		// More than half the weighted votes rejected → force reject.
		if rejectWeight > totalWeight/2 {
			return s.contentRepo.UpdateStatus(ctx, contentID, "rejected")
		}
	}

	// --- Phase 4: AI risk threshold (no human review needed) ---

	if len(elements) > 0 {
		totalAIRisk := 0
		counted := 0
		for _, e := range elements {
			if e.AIStatus == model.ElementAIPassed || e.AIStatus == model.ElementAIRejected {
				totalAIRisk += e.AIRiskScore
				counted++
			}
		}
		if counted > 0 {
			avgAIRisk := totalAIRisk / counted
			if avgAIRisk > 60 {
				return s.contentRepo.UpdateStatus(ctx, contentID, "rejected")
			}
		}
	}

	// --- Phase 5: all human done, no rejection → approve; otherwise still pending ---

	allHumanDone := true
	for _, e := range elements {
		if e.HumanStatus != model.ElementHumanPassed && e.HumanStatus != model.ElementHumanRejected {
			allHumanDone = false
			break
		}
	}

	newStatus := "pending"
	if allHumanDone {
		newStatus = "approved"
	}

	return s.contentRepo.UpdateStatus(ctx, contentID, newStatus)
}
