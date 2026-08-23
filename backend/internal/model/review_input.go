package model

import (
	"errors"

	"github.com/google/uuid"
)

// UploadInput is the DTO used by IngestionService to create content and elements.
type UploadInput struct {
	ContentType    ContentType   `json:"content_type"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	ReviewPolicy   ReviewPolicy  `json:"review_policy"`
	FileURLs       []string      `json:"file_urls"`
	TenantID       uuid.UUID     `json:"tenant_id"`
	CreatorID      uuid.UUID     `json:"creator_id"`
}

// ErrAlreadyReviewed is returned when a handler tries to review an already-reviewed element.
var ErrAlreadyReviewed = errors.New("element has already been reviewed")
