package api

import (
	"context"
	"testing"

	"audit-platform/internal/model"
	"audit-platform/internal/service"

	"github.com/google/uuid"
)

// TestStartStreamRTMPURL verifies RTMP URL generation.
func TestStartStreamRTMPURL(t *testing.T) {
	key := "test-stream-key-123"
	url := GenerateRTMPPushURL(key)
	expected := "rtmp://localhost:1935/live/" + key
	if url != expected {
		t.Errorf("RTMP URL = %s, want %s", url, expected)
	}
}

// TestFallbackServiceEndToEnd simulates the full fallback pipeline.
func TestFallbackServiceEndToEnd(t *testing.T) {
	fb := service.NewLocalFallbackService()
	aiSvc := service.NewAIService("", "")
	aiSvc.WithFallback(fb)

	elem := model.ContentElement{
		ID:             uuid.New(),
		ElementContent: "这是一张正常的风景照片",
		ElementKind:    model.ElementCoverImage,
	}

	ctx := context.Background()
	record, err := aiSvc.ReviewElement(ctx, elem)
	if err != nil {
		t.Fatalf("fallback should succeed with empty key: %v", err)
	}
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record.AIScoreAfter == nil {
		t.Fatal("expected non-nil score")
	}
	t.Logf("Fallback review result: score=%d, action=%s", *record.AIScoreAfter, record.Action)
}
