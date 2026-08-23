package service

import (
	"context"
	"fmt"
	"log"
)

// NotificationType identifies the kind of notification being sent.
type NotificationType string

const (
	NotificationAppealSubmitted  NotificationType = "appeal_submitted"
	NotificationAppealResolved   NotificationType = "appeal_resolved"
	NotificationAppealApproved   NotificationType = "appeal_approved"
	NotificationAppealMaintained NotificationType = "appeal_maintained"
)

// NotificationPayload carries the data needed to send a notification.
type NotificationPayload struct {
	Type       NotificationType
	UserID     string // recipient user ID
	Title      string
	Message    string
	AppealID   string
	ContentID  string
}

// Notifier is the interface for sending notifications.
type Notifier interface {
	Notify(ctx context.Context, payload NotificationPayload) error
}

// ConsoleNotifier logs notifications to stdout — useful for development.
type ConsoleNotifier struct{}

func (n *ConsoleNotifier) Notify(ctx context.Context, payload NotificationPayload) error {
	msg := fmt.Sprintf("[NOTIFY] type=%s user=%s title=%q message=%q",
		payload.Type, payload.UserID, payload.Title, payload.Message)
	log.Println(msg)
	return nil
}

// MultiNotifier dispatches to multiple Notifiers in sequence.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (n *MultiNotifier) Notify(ctx context.Context, payload NotificationPayload) error {
	for _, notifier := range n.notifiers {
		if err := notifier.Notify(ctx, payload); err != nil {
			return fmt.Errorf("multi notifier: %w", err)
		}
	}
	return nil
}
