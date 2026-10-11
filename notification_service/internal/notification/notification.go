// Package notification turns saga events into user notifications.
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"notification_service/internal/messaging"
)

const (
	TopicOrderEvents   = "order.events"
	TopicPaymentEvents = "payment.events"
)

// Notification is a message delivered to a user.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	OrderID   string    `json:"orderId"`
	Event     string    `json:"event"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

// Sender delivers a notification through a channel (email, SMS, push...).
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// LogSender simulates e-mail delivery by writing to the log.
type LogSender struct{}

func (LogSender) Send(_ context.Context, n Notification) error {
	log.Printf("EMAIL to user %s: %s", n.UserID, n.Message)
	return nil
}

// eventPayload is the union of the fields used from order/payment events.
type eventPayload struct {
	OrderID string `json:"orderId"`
	UserID  string `json:"userId"`
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}

// Service builds notifications from events and keeps the latest ones in
// memory so they can be inspected over HTTP.
type Service struct {
	sender Sender
	limit  int

	mu    sync.RWMutex
	items []Notification
	seen  map[string]struct{} // event type + order id: duplicates are delivered once
}

func NewService(sender Sender) *Service {
	return &Service{sender: sender, limit: 1000, seen: map[string]struct{}{}}
}

// Handle is the Kafka handler for both order.events and payment.events.
func (s *Service) Handle(ctx context.Context, env messaging.Envelope) error {
	var p eventPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil || p.UserID == "" {
		return nil // nothing to notify about
	}
	message, ok := render(env.Type, p)
	if !ok {
		return nil
	}

	// OrderStatusChanged may legitimately repeat with a different status.
	dedupKey := env.Type + "/" + p.OrderID + "/" + p.Status
	s.mu.RLock()
	_, dup := s.seen[dedupKey]
	s.mu.RUnlock()
	if dup {
		return nil
	}

	n := Notification{
		ID:        uuid.NewString(),
		UserID:    p.UserID,
		OrderID:   p.OrderID,
		Event:     env.Type,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.sender.Send(ctx, n); err != nil {
		return err // retried by the consumer
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[dedupKey] = struct{}{}
	s.items = append(s.items, n)
	if len(s.items) > s.limit {
		s.items = s.items[len(s.items)-s.limit:]
	}
	return nil
}

// ListByUser returns the user's notifications, newest first.
func (s *Service) ListByUser(userID string) []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Notification{}
	for _, n := range s.items {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func render(eventType string, p eventPayload) (string, bool) {
	short := p.OrderID
	if len(short) > 8 {
		short = short[:8]
	}
	money := fmt.Sprintf("%.2f", float64(p.Amount)/100)

	switch eventType {
	case "OrderCreated":
		return fmt.Sprintf("Order %s received (total %s). Waiting for payment.", short, money), true
	case "PaymentCompleted":
		return fmt.Sprintf("Payment of %s for order %s succeeded.", money, short), true
	case "PaymentFailed":
		return fmt.Sprintf("Payment for order %s failed: %s.", short, p.Reason), true
	case "OrderCancelled":
		return fmt.Sprintf("Order %s was cancelled: %s.", short, p.Reason), true
	case "PaymentRefunded":
		return fmt.Sprintf("%s for order %s has been refunded.", money, short), true
	case "OrderStatusChanged":
		return fmt.Sprintf("Order %s is now %s.", short, p.Status), true
	default: // e.g. OrderPaid duplicates PaymentCompleted for the user
		return "", false
	}
}
