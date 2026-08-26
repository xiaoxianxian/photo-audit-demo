// Package search provides Elasticsearch-backed full-text search over audit
// records (Phase 2). Records are indexed asynchronously after each review
// action; search failures never break the review flow.
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
)

var esLog = logger.New("es_search")

// Index holding searchable audit records.
const AuditIndex = "audit_records"

// Client wraps the ES client with audit-record specific operations.
type Client struct {
	es *elasticsearch.Client
}

// New connects to Elasticsearch and ensures the index mapping exists.
// Returns an error only if the cluster is unreachable — callers decide
// whether that is fatal (it usually is not: search is best-effort).
func New(url string) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{url}})
	if err != nil {
		return nil, fmt.Errorf("es client: %w", err)
	}
	c := &Client{es: es}
	if err := c.ensureIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("es ensure index: %w", err)
	}
	return c, nil
}

// ensureIndex creates the audit index with explicit mappings if missing.
func (c *Client) ensureIndex(ctx context.Context) error {
	res, err := c.es.Indices.Exists([]string{AuditIndex},
		c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return nil // already exists
	}

	mapping := map[string]any{
		"settings": map[string]any{
			"number_of_shards": 1, "number_of_replicas": 0,
			"analysis": map[string]any{
				"analyzer": map[string]any{
					"edge_ngram_analyzer": map[string]any{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"edge_ngram"},
					},
				},
				"filter": map[string]any{
					"edge_ngram": map[string]any{
						"type":     "edge_ngram",
						"minGram":  1,
						"maxGram":  10,
						"token_type": "word",
					},
				},
			},
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"id":                 map[string]any{"type": "keyword"},
				"element_id":         map[string]any{"type": "keyword"},
				"reviewer_id":        map[string]any{"type": "keyword"},
				"tenant_id":          map[string]any{"type": "keyword"}, // filter field
				"review_type":        map[string]any{"type": "keyword"},
				"action":             map[string]any{"type": "keyword"},
				"penalty_level_code": map[string]any{"type": "keyword"},
				"reason":             map[string]any{"type": "keyword"},
				"comment": map[string]any{
					"type":     "text",
					"analyzer": "edge_ngram_analyzer",
				},
				"ai_score_before": map[string]any{"type": "integer"},
				"ai_score_after":  map[string]any{"type": "integer"},
				"is_conflict":     map[string]any{"type": "boolean"},
				"created_at":      map[string]any{"type": "date"},
			},
		},
	}
	body, _ := json.Marshal(mapping)
	res2, err := c.es.Indices.Create(AuditIndex,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(bytes.NewReader(body)))
	if err != nil {
		return err
	}
	defer res2.Body.Close()
	if res2.IsError() {
		return fmt.Errorf("create index: %s", res2.String())
	}
	esLog.Info("index %s created", AuditIndex)
	return nil
}

// doc converts an AuditRecord into the indexable document (adds tenantID).
func doc(r *model.AuditRecord, tenantID string) map[string]any {
	d := map[string]any{
		"id":          r.ID.String(),
		"task_id":     r.TaskID.String(),
		"element_id":  r.ElementID.String(),
		"tenant_id":   tenantID,
		"review_type": string(r.ReviewType),
		"action":      string(r.Action),
		"is_conflict": r.IsConflict,
		"created_at":  r.CreatedAt.Format(time.RFC3339),
	}
	if r.ReviewerID != nil {
		d["reviewer_id"] = r.ReviewerID.String()
	}
	if r.PenaltyLevel != nil {
		d["penalty_level_code"] = *r.PenaltyLevel
	}
	if r.Reason != nil {
		d["reason"] = string(*r.Reason)
	}
	if r.Comment != nil {
		d["comment"] = *r.Comment
	}
	if r.AIScoreBefore != nil {
		d["ai_score_before"] = *r.AIScoreBefore
	}
	if r.AIScoreAfter != nil {
		d["ai_score_after"] = *r.AIScoreAfter
	}
	return d
}

// IndexRecord indexes one audit record (idempotent by record ID).
func (c *Client) IndexRecord(ctx context.Context, r *model.AuditRecord, tenantID string) error {
	body, err := json.Marshal(doc(r, tenantID))
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}
	res, err := c.es.Index(AuditIndex, bytes.NewReader(body),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithDocumentID(r.ID.String()),
		c.es.Index.WithRefresh("false"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("index record: %s", res.String())
	}
	return nil
}

// SearchQuery filters for the audit-log search endpoint.
type SearchQuery struct {
	TenantID   string `json:"-"`
	Text       string // free-text match on comment/reason
	Action     string
	ReviewType string
	ReviewerID string
	Conflict   *bool
	Page       int
	PageSize   int
}

// SearchHit mirrors model.AuditRecord plus tenant_id for API responses.
type SearchHit struct {
	model.AuditRecord
	TenantID string `json:"tenant_id"`
}

// SearchResult is a page of hits with total count.
type SearchResult struct {
	Items []SearchHit `json:"items"`
	Total int64       `json:"total"`
}

// Search runs a filtered full-text query over audit records.
func (c *Client) Search(ctx context.Context, q SearchQuery) (*SearchResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	filter := []any{map[string]any{"term": map[string]any{"tenant_id": q.TenantID}}}
	if q.Action != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"action": q.Action}})
	}
	if q.ReviewType != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"review_type": q.ReviewType}})
	}
	if q.ReviewerID != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"reviewer_id": q.ReviewerID}})
	}
	if q.Conflict != nil {
		filter = append(filter, map[string]any{"term": map[string]any{"is_conflict": *q.Conflict}})
	}

	queryPart := any(map[string]any{"match_all": map[string]any{}})
	if q.Text != "" {
		queryPart = map[string]any{
			"multi_match": map[string]any{
				"query":  q.Text,
				"fields": []string{"comment", "reason"},
			},
		}
	}

	body := map[string]any{
		"query": map[string]any{"bool": map[string]any{
			"must":   queryPart,
			"filter": filter,
		}},
		"sort": []any{map[string]any{"created_at": "desc"}},
		"from": (q.Page - 1) * q.PageSize,
		"size": q.PageSize,
	}
	raw, _ := json.Marshal(body)

	res, err := esapi.SearchRequest{
		Index: []string{AuditIndex},
		Body:  bytes.NewReader(raw),
	}.Do(ctx, c.es)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("search: %s", res.String())
	}

	var out struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	result := &SearchResult{Total: out.Hits.Total.Value, Items: make([]SearchHit, 0, len(out.Hits.Hits))}
	for _, h := range out.Hits.Hits {
		src := h.Source
		hit := SearchHit{}
		hit.TenantID, _ = src["tenant_id"].(string)
		hit.ID = parseUUID(src["id"])
		hit.TaskID = parseUUID(src["task_id"])
		hit.ElementID = parseUUID(src["element_id"])
		if v, ok := src["reviewer_id"].(string); ok && v != "" {
			u := parseUUID(v)
			hit.ReviewerID = &u
		}
		if v, ok := src["review_type"].(string); ok {
			hit.ReviewType = model.ReviewType(v)
		}
		if v, ok := src["action"].(string); ok {
			hit.Action = model.ReviewAction(v)
		}
		if v, ok := src["penalty_level_code"].(string); ok {
			hit.PenaltyLevel = &v
		}
		if v, ok := src["reason"].(string); ok {
			r := model.RejectReason(v)
			hit.Reason = &r
		}
		if v, ok := src["comment"].(string); ok {
			hit.Comment = &v
		}
		if v, ok := src["ai_score_before"].(float64); ok {
			i := int(v)
			hit.AIScoreBefore = &i
		}
		if v, ok := src["ai_score_after"].(float64); ok {
			i := int(v)
			hit.AIScoreAfter = &i
		}
		if v, ok := src["is_conflict"].(bool); ok {
			hit.IsConflict = v
		}
		if v, ok := src["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				hit.CreatedAt = t
			}
		}
		result.Items = append(result.Items, hit)
	}
	return result, nil
}

func parseUUID(v any) uuid.UUID {
	s, _ := v.(string)
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
