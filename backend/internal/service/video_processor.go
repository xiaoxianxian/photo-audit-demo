package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

var videoLog = logger.New("video_processor")

// VideoProcessor handles video-specific preprocessing: frame extraction and ASR transcription.
type VideoProcessor struct {
	elementRepo *repository.ElementRepository
	// ffmpegPath is the path to the ffmpeg binary. Defaults to "ffmpeg" (in PATH).
	ffmpegPath string
	// asrEndpoint is the ASR API endpoint. Empty means use mock ASR.
	asrEndpoint string
	asrAPIKey   string
	// MaxFrames limits how many frames are extracted (avoids element bloat).
	maxFrames int
	// FrameIntervalSeconds defines the gap between consecutive frames.
	frameInterval int
	// fingerprintSvc handles video fingerprinting for deduplication.
	fingerprintSvc *FingerprintService
}

// NewVideoProcessor creates a VideoProcessor with the given dependencies.
func NewVideoProcessor(elementRepo *repository.ElementRepository) *VideoProcessor {
	return &VideoProcessor{
		elementRepo:   elementRepo,
		ffmpegPath:    "ffmpeg",
		maxFrames:     10,
		frameInterval: 5,
	}
}

// SetFFmpegPath overrides the ffmpeg binary path.
func (p *VideoProcessor) SetFFmpegPath(path string) {
	p.ffmpegPath = path
}

// SetFingerprintService attaches a fingerprint service for video deduplication.
func (p *VideoProcessor) SetFingerprintService(fps *FingerprintService) {
	p.fingerprintSvc = fps
}

// ExtractFrames runs ffmpeg to extract keyframes from a video at regular intervals.
// Returns a slice of frame image bytes and their timestamps.
func (p *VideoProcessor) ExtractFrames(ctx context.Context, videoData []byte, totalDurationSec int) ([][]byte, []float64, error) {
	if totalDurationSec <= 0 {
		totalDurationSec = 60 // default assumption for 1-minute clips
	}

	// Limit frames: don't exceed maxFrames.
	interval := p.frameInterval
	frameCount := totalDurationSec / interval
	if frameCount > p.maxFrames {
		interval = totalDurationSec / p.maxFrames
		if interval < 1 {
			interval = 1
		}
		frameCount = p.maxFrames
	}

	// Write video to a temp file for ffmpeg.
	tmpFile, err := writeTempFile(videoData, ".mp4")
	if err != nil {
		return nil, nil, fmt.Errorf("write temp video: %w", err)
	}
	defer cleanupTemp(tmpFile)

	// ffmpeg command: extract one frame every N seconds.
	// -vf "select='eq(n\,0)+eq(mod(n\,%d),0)'" selects frame 0 + every Nth frame.
	// -frames:v 1 ensures only one frame per selection.
	cmd := exec.CommandContext(
		ctx,
		p.ffmpegPath,
		"-i", tmpFile,
		"-vf", fmt.Sprintf("select='eq(n\\,0)+eq(mod(n\\,%d),0)',scale=320:-1", interval),
		"-vsync", "vfr",
		"-frames:v", "1",
		"-q:v", "2",
		"-f", "image2pipe",
		"-pix_fmt", "rgb24",
		"pipe:1",
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		if isFFmpegNotFound(err) {
			return nil, nil, fmt.Errorf("ffmpeg not found in PATH — video frame extraction disabled (mock frames will be used)")
		}
		return nil, nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("ffmpeg extract: %w", err)
	}

	frameData := stdout.Bytes()
	if len(frameData) == 0 {
		return nil, nil, fmt.Errorf("ffmpeg produced empty output")
	}

	// Generate timestamps for each frame.
	timestamps := make([]float64, 0, frameCount)
	for i := 0; i < frameCount && float64(i)*float64(interval) <= float64(totalDurationSec); i++ {
		timestamps = append(timestamps, float64(i)*float64(interval))
	}

	return [][]byte{frameData}, timestamps, nil
}

// ExtractFramesBatch extracts multiple frames from a video.
// Each frame is written as a separate JPEG.
func (p *VideoProcessor) ExtractFramesBatch(ctx context.Context, videoData []byte, totalDurationSec int) ([][]byte, []float64, error) {
	if totalDurationSec <= 0 {
		totalDurationSec = 60
	}

	tmpFile, err := writeTempFile(videoData, ".mp4")
	if err != nil {
		return nil, nil, fmt.Errorf("write temp video: %w", err)
	}
	defer cleanupTemp(tmpFile)

	// Calculate frame count and interval.
	interval := p.frameInterval
	frameCount := totalDurationSec / interval
	if frameCount > p.maxFrames {
		interval = totalDurationSec / p.maxFrames
		if interval < 1 {
			interval = 1
		}
		frameCount = p.maxFrames
	}

	// Use ffmpeg to extract frames at specific timestamps.
	var allFrames [][]byte
	var timestamps []float64

	for i := 0; i < frameCount; i++ {
		ts := float64(i) * float64(interval)
		if ts > float64(totalDurationSec) {
			break
		}

		cmd := exec.CommandContext(
			ctx,
			p.ffmpegPath,
			"-ss", fmt.Sprintf("%.1f", ts),
			"-i", tmpFile,
			"-frames:v", "1",
			"-vf", "scale=320:-1",
			"-q:v", "2",
			"-f", "jpeg",
			"pipe:1",
		)

		out, err := cmd.Output()
		if err != nil {
			// Skip failed frames (e.g., near end of video).
			continue
		}
		if len(out) == 0 {
			continue
		}

		allFrames = append(allFrames, out)
		timestamps = append(timestamps, ts)
	}

	if len(allFrames) == 0 {
		return nil, nil, fmt.Errorf("ffmpeg extracted 0 frames")
	}

	return allFrames, timestamps, nil
}

// TranscribeASR sends audio from a video to an ASR service and returns the transcript text.
// If no ASR endpoint is configured, returns a mock transcript.
func (p *VideoProcessor) TranscribeASR(ctx context.Context, videoData []byte, durationSec int) (string, error) {
	if p.asrEndpoint == "" || p.asrAPIKey == "" {
		// Mock ASR: return a placeholder transcript.
		return fmt.Sprintf("[ASR 转写模拟] 视频时长 %d 秒 — 此处将接入真实 ASR API (设置 ASR_ENDPOINT + ASR_API_KEY 环境变量启用)", durationSec), nil
	}

	// Extract audio from video for ASR.
	tmpFile, err := writeTempFile(videoData, ".mp4")
	if err != nil {
		return "", fmt.Errorf("write temp video: %w", err)
	}
	defer cleanupTemp(tmpFile)

	tmpAudio, err := writeTempFile(nil, ".wav")
	if err != nil {
		return "", fmt.Errorf("write temp audio: %w", err)
	}
	defer cleanupTemp(tmpAudio)

	// Extract audio stream.
	audioCmd := exec.CommandContext(
		ctx,
		p.ffmpegPath,
		"-i", tmpFile,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		tmpAudio,
	)
	if err := audioCmd.Run(); err != nil {
		return "", fmt.Errorf("extract audio: %w", err)
	}

	// Read audio data.
	audioData, err := readTempFile(tmpAudio)
	if err != nil {
		return "", fmt.Errorf("read audio: %w", err)
	}

	// Send to ASR endpoint (multipart form).
	resp, err := p.callASREndpoint(ctx, audioData)
	if err != nil {
		return "", fmt.Errorf("ASR call: %w", err)
	}

	return resp, nil
}

// callASREndpoint sends audio data to the ASR API and returns the transcript.
func (p *VideoProcessor) callASREndpoint(ctx context.Context, audioData []byte) (string, error) {
	// Placeholder: in production, this would send audioData to the ASR endpoint
	// and parse the JSON response containing the transcript.
	_ = ctx
	_ = audioData
	return fmt.Sprintf("[ASR 转写] 已发送到 %s (待接入真实 API)", p.asrEndpoint), nil
}

// ProcessVideo performs the full video preprocessing pipeline:
// 1. Detect video duration (via ffprobe).
// 2. Extract keyframes.
// 3. Run ASR transcription.
// 4. Return elements to be inserted.
func (p *VideoProcessor) ProcessVideo(ctx context.Context, videoData []byte, originalFilename string) ([]model.ContentElement, error) {
	elements := make([]model.ContentElement, 0)

	// Step 1: Get video duration via ffprobe.
	durationSec := 60 // default assumption
	if dur, err := p.getDuration(ctx, videoData); err == nil {
		durationSec = dur
	}

	// Step 2: Extract frames.
	frames, timestamps, err := p.ExtractFramesBatch(ctx, videoData, durationSec)
	if err != nil {
		// Frame extraction failed — continue with empty frames (ASR still works).
		videoLog.Warn("frame extraction failed: %v, continuing without frames", err)
	}

	// Step 3: Generate video_frame elements from extracted frames.
	for i := range frames {
		frameElem := model.ContentElement{
			ID:             uuid.New(),
			ElementKind:    model.ElementVideoFrame,
			ElementContent: fmt.Sprintf("frame_%d:%.1fs", i+1, timestamps[i]),
			AIRiskScore:    0,
			AIStatus:       model.ElementPendingAI,
			HumanStatus:    model.ElementPendingHuman,
			CreatedAt:      time.Now(),
		}
		elements = append(elements, frameElem)
	}

	// Step 4: ASR transcription.
	asrText, err := p.TranscribeASR(ctx, videoData, durationSec)
	if err != nil {
		videoLog.Warn("ASR transcription failed: %v", err)
		asrText = fmt.Sprintf("[ASR 转写失败] %v", err)
	}

	asrElem := model.ContentElement{
		ID:             uuid.New(),
		ElementKind:    model.ElementASRText,
		ElementContent: asrText,
		AIRiskScore:    0,
		AIStatus:       model.ElementPendingAI,
		HumanStatus:    model.ElementPendingHuman,
		CreatedAt:      time.Now(),
	}
	elements = append(elements, asrElem)

	// Step 5: Compute video fingerprint for deduplication.
	if p.fingerprintSvc != nil && len(frames) > 0 {
		fp, err := p.fingerprintSvc.FingerprintVideo(frames[0], videoData)
		if err != nil {
			videoLog.Warn("fingerprint computation failed: %v", err)
		} else {
			// Store fingerprint as a hex string in a dedicated element.
			fpElem := model.ContentElement{
				ID:             uuid.New(),
				ElementKind:    "video_fingerprint",
				ElementContent: fmt.Sprintf("%016x", fp),
				AIRiskScore:    0,
				AIStatus:       model.ElementAIPassed,
				HumanStatus:    model.ElementPendingHuman,
				CreatedAt:      time.Now(),
			}
			elements = append(elements, fpElem)
		}
	}

	return elements, nil
}

// getDuration uses ffprobe to get video duration in seconds.
func (p *VideoProcessor) getDuration(ctx context.Context, videoData []byte) (int, error) {
	tmpFile, err := writeTempFile(videoData, ".mp4")
	if err != nil {
		return 0, err
	}
	defer cleanupTemp(tmpFile)

	cmd := exec.CommandContext(
		ctx,
		p.ffmpegPath,
		"-i", tmpFile,
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	duration := 0
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &duration)
	if duration <= 0 {
		return 0, fmt.Errorf("ffprobe returned invalid duration: %s", string(out))
	}

	return duration, nil
}

// isFFmpegNotFound checks if an error indicates ffmpeg binary not found.
func isFFmpegNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "executable file not found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "No such file or directory")
}

// --- Temp file helpers ---

func writeTempWithData(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func writeTempFile(data []byte, ext string) (string, error) {
	cmd := exec.Command("mktemp", "/tmp/audit-XXXXXX"+ext)
	f, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(f))

	if len(data) > 0 {
		if err := writeTempWithData(path, data); err != nil {
			return "", err
		}
	}
	return path, nil
}

func readTempFile(path string) ([]byte, error) {
	cmd := exec.Command("cat", path)
	return cmd.Output()
}

func cleanupTemp(path string) {
	exec.Command("rm", "-f", path).Run()
}
