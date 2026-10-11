package notification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"notification_service/internal/messaging"
)

type recordingSender struct{ sent []Notification }

func (r *recordingSender) Send(_ context.Context, n Notification) error {
	r.sent = append(r.sent, n)
	return nil
}

func envelope(t *testing.T, eventType string, payload map[string]any) messaging.Envelope {
	t.Helper()
	env, err := messaging.NewEnvelope("test", eventType, payload)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestFailedPaymentFlowNotifiesUser(t *testing.T) {
	sender := &recordingSender{}
	svc := NewService(sender)
	ctx := context.Background()
	base := map[string]any{"orderId": "12345678-aaaa", "userId": "u1", "amount": 25000}

	events := []messaging.Envelope{
		envelope(t, "OrderCreated", base),
		envelope(t, "PaymentFailed", map[string]any{"orderId": "12345678-aaaa", "userId": "u1", "reason": "insufficient funds"}),
		envelope(t, "OrderCancelled", map[string]any{"orderId": "12345678-aaaa", "userId": "u1", "reason": "payment failed"}),
	}
	for _, e := range events {
		if err := svc.Handle(ctx, e); err != nil {
			t.Fatalf("Handle(%s) error: %v", e.Type, err)
		}
	}
	// Kafka may redeliver: the user must not get the same message twice.
	_ = svc.Handle(ctx, envelope(t, "PaymentFailed", map[string]any{"orderId": "12345678-aaaa", "userId": "u1", "reason": "insufficient funds"}))

	if len(sender.sent) != 3 {
		t.Fatalf("sent %d notifications, want 3", len(sender.sent))
	}
	if !strings.Contains(sender.sent[0].Message, "250.00") {
		t.Fatalf("OrderCreated message = %q, want total 250.00", sender.sent[0].Message)
	}
	if !strings.Contains(sender.sent[1].Message, "insufficient funds") {
		t.Fatalf("PaymentFailed message = %q", sender.sent[1].Message)
	}
	if got := svc.ListByUser("u1"); len(got) != 3 {
		t.Fatalf("ListByUser = %d, want 3", len(got))
	}
}

func TestIgnoresIrrelevantEvents(t *testing.T) {
	sender := &recordingSender{}
	svc := NewService(sender)

	_ = svc.Handle(context.Background(), envelope(t, "OrderPaid", map[string]any{"orderId": "o", "userId": "u"}))
	bad := messaging.Envelope{Type: "OrderCreated", Payload: json.RawMessage(`not json`)}
	if err := svc.Handle(context.Background(), bad); err != nil {
		t.Fatalf("malformed event must be skipped, got %v", err)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("sent %+v, want nothing", sender.sent)
	}
}

func TestStatusChangesAreNotDeduplicatedAcrossStatuses(t *testing.T) {
	sender := &recordingSender{}
	svc := NewService(sender)
	for _, s := range []string{"SHIPPED", "DELIVERED"} {
		_ = svc.Handle(context.Background(), envelope(t, "OrderStatusChanged", map[string]any{"orderId": "o", "userId": "u", "status": s}))
	}
	if len(sender.sent) != 2 {
		t.Fatalf("sent %d, want 2", len(sender.sent))
	}
}
