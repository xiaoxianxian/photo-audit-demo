package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

var aiLog = logger.New("ai_service")

// AIService handles interactions with external AI review models.
type AIService struct {
	agnesAPIKey    string
	deepseekAPIKey string
	httpClient     *http.Client
	endpoint       string
	fallback       *LocalFallbackService
	aiConfigRepo   *repository.AIConfigRepository
}

// aiReviewRequest is the JSON payload sent to Agnes AI.
type aiReviewRequest struct {
	Model  string                 `json:"model"`
	Input  map[string]interface{} `json:"input"`
	Task   string                 `json:"task"`
}

// aiReviewResponse is the JSON response from Agnes AI.
type aiReviewResponse struct {
	RiskScore  int      `json:"risk_score"`
	RiskTypes  []string `json:"risk_types"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
	IsSafe     bool     `json:"is_safe"`
}

// NewAIService creates an AI service backed by the given API keys.
// The Agnes endpoint defaults to AGNES_API_ENDPOINT env var, falling back to https://api.agnes.ai/v1/review.
func NewAIService(agnesKey, deepseekKey string) *AIService {
	endpoint := os.Getenv("AGNES_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.agnes.ai/v1/review"
	}

	return &AIService{
		agnesAPIKey:    agnesKey,
		deepseekAPIKey: deepseekKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		endpoint: endpoint,
	}
}

// WithFallback attaches a local rule-based fallback service.
// When the external AI returns quota/rate-limit errors, this fallback
// is invoked automatically to produce a risk score.
func (s *AIService) WithFallback(fb *LocalFallbackService) {
	s.fallback = fb
}

// WithAIConfigRepo sets the AI config repository for dynamic key retrieval.
func (s *AIService) WithAIConfigRepo(repo *repository.AIConfigRepository) {
	s.aiConfigRepo = repo
}

// ReviewElement sends a content element to Agnes AI for multimodal review
// and returns a structured AuditRecord with review_type="ai_primary".
// If no API key is configured, falls back to local rule-based review.
func (s *AIService) ReviewElement(ctx context.Context, element model.ContentElement) (*model.AuditRecord, error) {
	// No API key configured — use local fallback immediately.
	if s.agnesAPIKey == "" && s.fallback != nil {
		return s.fallback.ReviewElement(element), nil
	}
	if s.agnesAPIKey == "" {
		return nil, fmt.Errorf("AGNES_API_KEY not configured")
	}

	inputType := "image"
	inputData := map[string]interface{}{
		"type": inputType,
	}

	switch element.ElementKind {
	case model.ElementCoverImage, model.ElementVideoFrame, model.ElementLiveSnapshot, model.ElementUserAvatar:
		inputData["url"] = element.ElementContent
	case model.ElementTitle, model.ElementComment, model.ElementASRText, model.ElementNickname, model.ElementDescription:
		inputType = "text"
		inputData["type"] = inputType
		inputData["text"] = element.ElementContent
	default:
		inputData["text"] = element.ElementContent
		inputData["type"] = inputType
	}

	reqBody, err := json.Marshal(aiReviewRequest{
		Model: "agnes-multimodal-v1",
		Input: inputData,
		Task:  "content_review",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal AI review request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create AI review request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.agnesAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send AI review request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if serr := s.DetectQuotaError(resp, body); serr != nil {
			// Quota/rate-limit error — trigger local fallback if available.
			if s.fallback != nil {
				aiLog.Info("AI quota error (%s), triggering local fallback: %v", serr.Error(), serr)
				return s.fallback.ReviewElement(element), nil
			}
			return nil, fmt.Errorf("context: %w", serr)
		}
		return nil, fmt.Errorf("AI review failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read AI review response: %w", err)
	}

	result, err := s.ParseAIResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse AI response: %w", err)
	}

	record := &model.AuditRecord{
		ID:             uuid.Nil, // to be set by repository
		TaskID:         element.ID,
		ElementID:      element.ID,
		ReviewerID:     nil, // AI reviewer
		ReviewType:     model.ReviewTypeAIPrimary,
		Action:         model.ActionApprove,
		AIScoreBefore:  nil,
		AIScoreAfter:   &result.Score,
		IsConflict:     false,
		CreatedAt:      time.Now(),
	}

	if result.Score >= 60 {
		record.Action = model.ActionReject
	}

	if len(result.Types) > 0 {
		reason := model.RejectReason(result.Types[0])
		record.Reason = &reason
	}

	if result.Reason != "" {
		record.Comment = &result.Reason
	}

	return record, nil
}

// JudgeReview sends a primary AI review result to DeepSeek for consistency checking.
// Returns an AuditRecord with review_type="ai_judge". If the score difference exceeds 20,
// IsConflict is set to true.
func (s *AIService) JudgeReview(ctx context.Context, primaryRecord *model.AuditRecord) (*model.AuditRecord, error) {
	if primaryRecord.AIScoreAfter == nil {
		return nil, fmt.Errorf("judge review: primary record has no ai_score_after")
	}

	primaryJSON, err := json.Marshal(map[string]interface{}{
		"score":  *primaryRecord.AIScoreAfter,
		"action": string(primaryRecord.Action),
		"reason": primaryRecord.Comment,
		"types":  primaryRecord.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal primary review for judging: %w", err)
	}

	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": fmt.Sprintf("Judge this review result: %s", string(primaryJSON)),
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal judge request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.deepseek.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create judge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.deepseekAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send judge request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("judge review failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read judge response: %w", err)
	}

	judgeResult, err := parseJudgeResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse judge response: %w", err)
	}

	diff := *primaryRecord.AIScoreAfter - judgeResult
	if diff < 0 {
		diff = -diff
	}

	record := &model.AuditRecord{
		ID:             uuid.Nil,
		TaskID:         primaryRecord.TaskID,
		ElementID:      primaryRecord.ElementID,
		ReviewerID:     nil,
		ReviewType:     model.ReviewTypeAIPJudge,
		Action:         primaryRecord.Action,
		AIScoreBefore:  primaryRecord.AIScoreAfter,
		AIScoreAfter:   &judgeResult,
		IsConflict:     diff > 20,
		CreatedAt:      time.Now(),
	}

	if record.IsConflict {
		record.Comment = stringPtr(fmt.Sprintf("裁判分歧: 主审 %d, 裁判 %d", *primaryRecord.AIScoreAfter, judgeResult))
	}

	return record, nil
}

// ParseAIResponse parses the JSON body returned by Agnes AI into a structured result.
// Validates that risk_score is in [0, 100].
func (s *AIService) ParseAIResponse(body []byte) (*struct {
	Score      int
	Types      []string
	Confidence float64
	Reason     string
}, error) {
	var resp aiReviewResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal AI response: %w", err)
	}

	if resp.RiskScore < 0 || resp.RiskScore > 100 {
		return nil, fmt.Errorf("risk_score out of range: %d", resp.RiskScore)
	}

	if resp.Confidence < 0 || resp.Confidence > 1 {
		return nil, fmt.Errorf("confidence out of range: %.2f", resp.Confidence)
	}

	return &struct {
		Score      int
		Types      []string
		Confidence float64
		Reason     string
	}{
		Score:      resp.RiskScore,
		Types:      resp.RiskTypes,
		Confidence: resp.Confidence,
		Reason:     resp.Reason,
	}, nil
}

// DetectQuotaError inspects an HTTP response and its body to determine whether
// the error is quota/limit related (402, 429, or specific keyword matches).
// Returns a typed error when quota limits are hit.
func (s *AIService) DetectQuotaError(resp *http.Response, body []byte) error {
	status := resp.StatusCode
	bodyStr := string(body)

	quotaKeywords := []string{"quota", "billing", "rate_limit", "insufficient"}

	isQuota := false

	switch status {
	case http.StatusPaymentRequired, http.StatusTooManyRequests:
		isQuota = true
	default:
		for _, kw := range quotaKeywords {
			if strings.Contains(strings.ToLower(bodyStr), kw) {
				isQuota = true
				break
			}
		}
	}

	if !isQuota {
		return nil
	}

	switch status {
	case http.StatusPaymentRequired:
		return fmt.Errorf("ai_quota_exhausted: %s", bodyStr)
	case http.StatusTooManyRequests:
		return fmt.Errorf("ai_rate_limited: %s", bodyStr)
	default:
		return fmt.Errorf("ai_quota_error: %s", bodyStr)
	}
}

// parseJudgeResponse extracts the judge's score from a DeepSeek JSON response.
// It looks for a "score" integer in the parsed JSON under choices[0].message.content.
// Uses strict regex to only match explicit "score" keys, avoiding accidental digit
// extraction from arbitrary numbers like "3 violations".
func parseJudgeResponse(body []byte) (int, error) {
	var outer struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &outer); err != nil {
		return 0, fmt.Errorf("unmarshal judge response: %w", err)
	}
	if len(outer.Choices) == 0 {
		return 0, errors.New("judge response has no choices")
	}

	content := outer.Choices[0].Message.Content

	// Look for explicit score markers: "score": 42, "score":42, score=42, score: 42
	scoreRegex := regexp.MustCompile(`(?i)"?score"?\s*[:=]\s*(\d+)`)
	if matches := scoreRegex.FindStringSubmatch(content); len(matches) > 1 {
		s := 0
		for _, ch := range matches[1] {
			s = s*10 + int(ch-'0')
			if s > 100 {
				s = 100
				break
			}
		}
		return s, nil
	}

	// Fallback: no explicit score found; return 0 to indicate unclear result.
	return 0, nil
}

// GetAIConfigs retrieves all AI configuration items for a tenant.
// API keys are masked before being returned to prevent exposure.
func (s *AIService) GetAIConfigs(ctx context.Context, tenantID string) ([]model.AIConfig, error) {
	if s.aiConfigRepo == nil {
		return nil, fmt.Errorf("AI config repo not initialized")
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}
	config, err := s.aiConfigRepo.GetByTenant(ctx, tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("get ai configs: %w", err)
	}
	if config == nil {
		return []model.AIConfig{}, nil
	}

	// Mask API keys in response to prevent exposure.
	result := []model.AIConfig{*config}
	result[0].AgnesAPIKey = maskAPIKey(result[0].AgnesAPIKey)
	result[0].DeepSeekAPIKey = maskAPIKey(result[0].DeepSeekAPIKey)
	return result, nil
}

// maskAPIKey masks an API key, showing only the last 4 characters.
// Returns "***" if the key is empty or too short.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "***"
	}
	return "****" + key[len(key)-4:]
}
