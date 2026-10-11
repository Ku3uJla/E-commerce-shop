package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	byOrder map[string]*Payment
}

func (r *fakeRepo) Create(_ context.Context, p *Payment) error {
	if _, ok := r.byOrder[p.OrderID]; ok {
		return ErrAlreadyExists
	}
	cp := *p
	r.byOrder[p.OrderID] = &cp
	return nil
}

func (r *fakeRepo) Update(_ context.Context, p *Payment) error {
	cp := *p
	r.byOrder[p.OrderID] = &cp
	return nil
}

func (r *fakeRepo) GetByOrderID(_ context.Context, orderID string) (*Payment, error) {
	if p, ok := r.byOrder[orderID]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, ErrPaymentNotFound
}

type published struct {
	eventType string
	payload   ResultPayload
}

type fakePublisher struct {
	events []published
	fail   bool
}

func (f *fakePublisher) Publish(_ context.Context, eventType, _ string, payload any) error {
	if f.fail {
		return errors.New("kafka down")
	}
	f.events = append(f.events, published{eventType, payload.(ResultPayload)})
	return nil
}

type countingGateway struct {
	*SimulatedGateway
	charges int
	refunds int
}

func (g *countingGateway) Charge(ctx context.Context, userID string, amount int64) error {
	g.charges++
	return g.SimulatedGateway.Charge(ctx, userID, amount)
}

func (g *countingGateway) Refund(ctx context.Context, id string, amount int64) error {
	g.refunds++
	return g.SimulatedGateway.Refund(ctx, id, amount)
}

func setup() (*paymentSrv, *fakeRepo, *fakePublisher, *countingGateway) {
	repo := &fakeRepo{byOrder: map[string]*Payment{}}
	pub := &fakePublisher{}
	gw := &countingGateway{SimulatedGateway: NewSimulatedGateway(10000, 0)} // limit 100.00
	return NewService(repo, gw, pub), repo, pub, gw
}

func orderCreated(amount int64) OrderCreatedPayload {
	return OrderCreatedPayload{OrderID: uuid.NewString(), UserID: uuid.NewString(), Amount: amount}
}

func TestPaymentCompleted(t *testing.T) {
	srv, repo, pub, _ := setup()
	e := orderCreated(5000)

	if err := srv.HandleOrderCreated(context.Background(), e); err != nil {
		t.Fatalf("HandleOrderCreated() error: %v", err)
	}
	if p := repo.byOrder[e.OrderID]; p.Status != StatusCompleted {
		t.Fatalf("status = %s, want COMPLETED", p.Status)
	}
	if len(pub.events) != 1 || pub.events[0].eventType != EventPaymentCompleted {
		t.Fatalf("published = %+v, want PaymentCompleted", pub.events)
	}
}

func TestPaymentDeclinedOverLimit(t *testing.T) {
	srv, repo, pub, _ := setup()
	e := orderCreated(20000)

	if err := srv.HandleOrderCreated(context.Background(), e); err != nil {
		t.Fatalf("a decline is a business outcome, not an error: %v", err)
	}
	if p := repo.byOrder[e.OrderID]; p.Status != StatusFailed || p.Reason == "" {
		t.Fatalf("payment = %+v, want FAILED with reason", p)
	}
	if len(pub.events) != 1 || pub.events[0].eventType != EventPaymentFailed {
		t.Fatalf("published = %+v, want PaymentFailed", pub.events)
	}
}

func TestRedeliveryDoesNotChargeTwice(t *testing.T) {
	srv, _, pub, gw := setup()
	ctx := context.Background()
	e := orderCreated(5000)

	// First attempt: charged and stored, but publishing fails -> error -> retry.
	pub.fail = true
	if err := srv.HandleOrderCreated(ctx, e); err == nil {
		t.Fatal("expected error when publishing fails")
	}
	pub.fail = false
	if err := srv.HandleOrderCreated(ctx, e); err != nil {
		t.Fatalf("redelivery error: %v", err)
	}

	if gw.charges != 1 {
		t.Fatalf("customer charged %d times, want 1", gw.charges)
	}
	if len(pub.events) != 1 || pub.events[0].eventType != EventPaymentCompleted {
		t.Fatalf("published = %+v, want one PaymentCompleted", pub.events)
	}
}

func TestCancelPaidOrderRefunds(t *testing.T) {
	srv, repo, pub, gw := setup()
	ctx := context.Background()
	e := orderCreated(5000)
	_ = srv.HandleOrderCreated(ctx, e)

	cancel := OrderCancelledPayload{OrderID: e.OrderID, UserID: e.UserID, Reason: "cancelled by request"}
	if err := srv.HandleOrderCancelled(ctx, cancel); err != nil {
		t.Fatalf("HandleOrderCancelled() error: %v", err)
	}
	_ = srv.HandleOrderCancelled(ctx, cancel) // duplicate

	if p := repo.byOrder[e.OrderID]; p.Status != StatusRefunded {
		t.Fatalf("status = %s, want REFUNDED", p.Status)
	}
	if gw.refunds != 1 {
		t.Fatalf("refunded %d times, want 1", gw.refunds)
	}
	if last := pub.events[len(pub.events)-1]; last.eventType != EventPaymentRefunded {
		t.Fatalf("last event = %s, want PaymentRefunded", last.eventType)
	}
}

func TestCancelAfterFailedPaymentIsNoop(t *testing.T) {
	srv, _, pub, gw := setup()
	ctx := context.Background()
	e := orderCreated(20000)
	_ = srv.HandleOrderCreated(ctx, e)

	// Order cancels itself in reaction to PaymentFailed; nothing to refund.
	if err := srv.HandleOrderCancelled(ctx, OrderCancelledPayload{OrderID: e.OrderID}); err != nil {
		t.Fatalf("HandleOrderCancelled() error: %v", err)
	}
	if gw.refunds != 0 || len(pub.events) != 1 {
		t.Fatalf("refunds = %d, events = %+v; want no refund", gw.refunds, pub.events)
	}
}
