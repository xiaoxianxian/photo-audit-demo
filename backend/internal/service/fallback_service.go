package service

import (
	"audit-platform/internal/model"
	"strings"
)

// LocalFallbackService provides rule-based content review as a fallback
// when the external AI API is unavailable (quota exhausted, rate limited, etc.).
// It uses simple heuristics on element content to produce a risk score.
type LocalFallbackService struct {
	// HighRiskWords contains words that trigger immediate high risk scores.
	HighRiskWords map[string]int // word -> risk score
}

// NewLocalFallbackService creates a fallback service with default rules.
func NewLocalFallbackService() *LocalFallbackService {
	return &LocalFallbackService{
		HighRiskWords: map[string]int{
			"违法":   90,
			"暴力":   90,
			"色情":   95,
			"赌博":   85,
			"毒品":   90,
			"恐怖":   90,
			"诈骗":   85,
			"侵权":   70,
			"抄袭":   65,
			"广告":   40,
			"引流":   50,
			"微商":   55,
			"二维码":  35,
			"微信号":  40,
		},
	}
}

// ReviewElement applies local rule-based review to an element.
// Returns an AuditRecord with review_type="ai_fallback".
func (s *LocalFallbackService) ReviewElement(element model.ContentElement) *model.AuditRecord {
	riskScore := 5 // baseline low risk
	riskTypes := make([]string, 0)
	comment := ""

	content := element.ElementContent

	// Check high-risk words.
	for word, score := range s.HighRiskWords {
		if strings.Contains(content, word) {
			if score > riskScore {
				riskScore = score
			}
			riskTypes = append(riskTypes, word)
			if comment == "" {
				comment = "命中敏感词" + word
			} else {
				comment += "; 命中敏感词" + word
			}
		}
	}

	// Check for excessive special characters (spam indicator).
	if strings.Count(content, "http") > 2 {
		riskScore = max(riskScore, 50)
		riskTypes = append(riskTypes, "spam")
		if comment == "" {
			comment = "疑似垃圾链接"
		} else {
			comment += "; 疑似垃圾链接"
		}
	}

	// Check for excessive repetition (noise indicator).
	if len(content) > 0 {
		runes := []rune(content)
		duplicateCount := 0
		for i := 1; i < len(runes); i++ {
			if runes[i] == runes[i-1] {
				duplicateCount++
			}
		}
		if duplicateCount > len(runes)/2 && len(runes) > 10 {
			riskScore = max(riskScore, 40)
			riskTypes = append(riskTypes, "noise")
		}
	}

	// Clamp score.
	if riskScore > 100 {
		riskScore = 100
	}

	action := model.ActionApprove
	if riskScore >= 60 {
		action = model.ActionReject
	}

	reason := model.RejectReason("")
	if len(riskTypes) > 0 {
		reason = model.RejectReason(riskTypes[0])
	}

	return &model.AuditRecord{
		ID:           element.ID,
		TaskID:       element.ID,
		ElementID:    element.ID,
		ReviewType:   model.ReviewTypeAIPrimary,
		Action:       action,
		AIScoreAfter: &riskScore,
		IsConflict:   false,
		Comment:      &comment,
		Reason:       &reason,
	}
}

// ReviewBatch applies fallback review to multiple elements.
func (s *LocalFallbackService) ReviewBatch(elements []model.ContentElement) []*model.AuditRecord {
	records := make([]*model.AuditRecord, 0, len(elements))
	for _, elem := range elements {
		records = append(records, s.ReviewElement(elem))
	}
	return records
}
