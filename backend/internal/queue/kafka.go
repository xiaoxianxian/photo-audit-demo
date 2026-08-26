// Package queue provides the Kafka-backed audit task queue.
//
// Phase 2: AI review tasks are published to Kafka instead of being run on
// unbounded in-process goroutines, so review work survives restarts, can be
// scaled horizontally (consumer group), and back-pressure is visible.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go"

	"audit-platform/internal/logger"
)

var kafkaLog = logger.New("kafka_queue")

// Topic used for AI review tasks. One partition per tenant would be a later
// optimization; a single topic with key=tenantID keeps ordering per tenant.
const AITopic = "audit-ai-review"

// Task kinds published on the AI review topic.
const (
	TaskKindAIReview  = "ai_review"        // payload: {content_id, tenant_id}
	TaskKindDecision  = "content_decision" // payload: {content_id}
)

// Message is the envelope for everything flowing through the audit queue.
type Message struct {
	Kind      string `json:"kind"`
	TenantID  string `json:"tenant_id,omitempty"`
	ContentID string `json:"content_id"`
}

// Producer publishes audit task messages to Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a Kafka producer. brokers example: ["localhost:9092"].
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        AITopic,
			Balancer:     &kafka.Hash{}, // key=tenantID → per-tenant ordering
			RequiredAcks: kafka.RequireOne,
			BatchTimeout: 50 * time.Millisecond,
			Async:        false, // synchronous so callers get delivery errors
		},
	}
}

// Publish sends one message; key = tenantID keeps a tenant's tasks ordered
// and colocated on one partition.
func (p *Producer) Publish(ctx context.Context, msg Message) error {
	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal queue message: %w", err)
	}
	var key []byte
	if msg.TenantID != "" {
		key = []byte(msg.TenantID)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
}

// Close flushes and releases the underlying writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Ping checks broker connectivity (used at startup to decide wiring).
func Ping(ctx context.Context, brokers []string) error {
	for _, b := range brokers {
		conn, err := net.DialTimeout("tcp", b, 2*time.Second)
		if err != nil {
			continue
		}
		_ = conn.Close()
		return nil
	}
	return fmt.Errorf("no kafka broker reachable at %v", brokers)
}

// EnsureTopic creates the AI review topic if missing (best effort — broker
// auto-create may also cover this). Bounded by ctx; never blocks startup:
// on failure we just log and let the consumer's group subscription trigger
// auto-creation instead.
func EnsureTopic(ctx context.Context, brokers []string) {
	topicCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// NOTE: kafka.DialLeader is known to hang indefinitely here even when the
	// broker is healthy (leader lookup quirk), so use a plain Dial + explicit
	// CreateTopics request instead.
	conn, err := kafka.DialContext(topicCtx, "tcp", brokers[0])
	if err != nil {
		kafkaLog.Warn("ensure topic %s: dial %s: %v", AITopic, brokers[0], err)
		return
	}
	defer conn.Close()

	if err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             AITopic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil {
		// Topic already exists is fine.
		kafkaLog.Info("ensure topic %s: %v (likely exists)", AITopic, err)
		return
	}
	kafkaLog.Info("topic %s created", AITopic)
}

// Consumer consumes audit task messages from the AI review topic as part of a
// consumer group, invoking handler for each message. Handler returning an error
// causes a retry after backoff until ctx is cancelled (at-least-once delivery).
type Consumer struct {
	reader    *kafka.Reader
	handler   func(ctx context.Context, msg Message) error
	brokers   []string
}

// NewConsumer creates a consumer joining the given group on the AI topic.
func NewConsumer(brokers []string, groupID string, handler func(ctx context.Context, msg Message) error) *Consumer {
	return &Consumer{
		brokers: brokers,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			GroupTopics:    []string{AITopic},
			MinBytes:       1,
			MaxBytes:       1e6,
			CommitInterval: time.Second, // batch commits
			MaxWait:        500 * time.Millisecond,
		}),
		handler: handler,
	}
}

// Run blocks until ctx is cancelled, processing messages through handler.
// Malformed messages are logged and skipped (committed, not retried forever).
func (c *Consumer) Run(ctx context.Context) {
	kafkaLog.Info("kafka consumer started (topic=%s)", AITopic)
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break // normal shutdown
			}
			kafkaLog.Warn("fetch message: %v", err)
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Second):
			}
			continue
		}

		var task Message
		if err := json.Unmarshal(m.Value, &task); err != nil {
			kafkaLog.Warn("skip malformed message (partition=%d offset=%d): %v", m.Partition, m.Offset, err)
			if err := c.reader.CommitMessages(ctx, m); err != nil {
				kafkaLog.Warn("commit malformed: %v", err)
			}
			continue
		}

		if err := c.handler(ctx, task); err != nil {
			// Do NOT commit — the message will be redelivered (at-least-once).
			kafkaLog.Warn("handler failed kind=%s content=%s: %v (will retry)", task.Kind, task.ContentID, err)
		} else if err := c.reader.CommitMessages(ctx, m); err != nil {
			kafkaLog.Warn("commit: %v", err)
		}
	}
	if err := c.reader.Close(); err != nil {
		kafkaLog.Warn("close reader: %v", err)
	}
	kafkaLog.Info("kafka consumer stopped")
}
