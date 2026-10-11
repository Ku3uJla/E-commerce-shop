package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeRepo stores orders in memory and keeps every event that would have
// gone to the outbox.
type fakeRepo struct {
	items  map[string]*Order
	outbox []Event
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]*Order{}}
}

func (r *fakeRepo) Create(_ context.Context, o *Order, events []Event) error {
	r.items[o.ID()] = o
	r.outbox = append(r.outbox, events...)
	return nil
}

func (r *fakeRepo) Update(_ context.Context, o *Order, events []Event) error {
	r.items[o.ID()] = o
	r.outbox = append(r.outbox, events...)
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (*Order, error) {
	if o, ok := r.items[id]; ok {
		return o, nil
	}
	return nil, ErrOrderNotFound
}

func (r *fakeRepo) ListByUser(_ context.Context, userID string) ([]*Order, error) {
	out := []*Order{}
	for _, o := range r.items {
		if o.UserID() == userID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *fakeRepo) eventTypes() []string {
	types := make([]string, 0, len(r.outbox))
	for _, e := range r.outbox {
		types = append(types, e.Type)
	}
	return types
}

// fakeCatalog prices every known product.
type fakeCatalog map[string]int64

func (c fakeCatalog) Prices(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		if p, ok := c[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

var (
	p1 = uuid.NewString()
	p2 = uuid.NewString()
)

func newTestService() (*orderSrv, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo, fakeCatalog{p1: 1000, p2: 250}), repo
}

func mustItem(t *testing.T, productID string, qty int) Item {
	t.Helper()
	it, err := NewItem(productID, qty)
	if err != nil {
		t.Fatalf("NewItem() error: %v", err)
	}
	return it
}

func placeOrder(t *testing.T, srv *orderSrv) *Order {
	t.Helper()
	o, err := srv.Place(context.Background(), uuid.NewString(), []Item{mustItem(t, p1, 1)})
	if err != nil {
		t.Fatalf("Place() error: %v", err)
	}
	return o
}

func assertEvents(t *testing.T, repo *fakeRepo, want ...string) {
	t.Helper()
	got := repo.eventTypes()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestPlacePricesMergesAndStartsSaga(t *testing.T) {
	srv, repo := newTestService()

	o, err := srv.Place(context.Background(), uuid.NewString(), []Item{
		mustItem(t, p1, 1), mustItem(t, p2, 2), mustItem(t, p1, 3),
	})
	if err != nil {
		t.Fatalf("Place() error: %v", err)
	}
	if o.Status() != StatusPending {
		t.Fatalf("status = %s, want PENDING", o.Status())
	}
	if len(o.Items()) != 2 || o.Items()[0].Quantity() != 4 {
		t.Fatalf("items not merged: %+v", o.Items())
	}
	if want := int64(4*1000 + 2*250); o.Total() != want {
		t.Fatalf("total = %d, want %d", o.Total(), want)
	}
	assertEvents(t, repo, EventOrderCreated)
	if p := repo.outbox[0].Payload.(OrderCreatedPayload); p.Amount != o.Total() || p.OrderID != o.ID() {
		t.Fatalf("OrderCreated payload = %+v", p)
	}
}

func TestPlaceValidation(t *testing.T) {
	srv, repo := newTestService()
	ctx := context.Background()

	if _, err := srv.Place(ctx, uuid.NewString(), nil); !errors.Is(err, ErrEmptyOrder) {
		t.Fatalf("Place(no items) error = %v, want %v", err, ErrEmptyOrder)
	}
	if _, err := srv.Place(ctx, "bad", []Item{mustItem(t, p1, 1)}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Place(bad user) error = %v, want %v", err, ErrInvalidID)
	}
	if _, err := srv.Place(ctx, uuid.NewString(), []Item{mustItem(t, uuid.NewString(), 1)}); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("Place(unknown product) error = %v, want %v", err, ErrProductNotFound)
	}
	if _, err := NewItem(p1, 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("NewItem(qty 0) error = %v, want %v", err, ErrInvalidQuantity)
	}
	assertEvents(t, repo) // nothing published for rejected orders
}

func TestSagaPaymentCompleted(t *testing.T) {
	srv, repo := newTestService()
	o := placeOrder(t, srv)

	if err := srv.HandlePaymentCompleted(context.Background(), o.ID(), "pay-1"); err != nil {
		t.Fatalf("HandlePaymentCompleted() error: %v", err)
	}
	got, _ := srv.Get(context.Background(), o.ID())
	if got.Status() != StatusPaid || got.PaymentID() != "pay-1" {
		t.Fatalf("order = %s/%s, want PAID/pay-1", got.Status(), got.PaymentID())
	}
	assertEvents(t, repo, EventOrderCreated, EventOrderPaid)
}

func TestSagaPaymentFailedCompensates(t *testing.T) {
	srv, repo := newTestService()
	o := placeOrder(t, srv)

	if err := srv.HandlePaymentFailed(context.Background(), o.ID(), "insufficient funds"); err != nil {
		t.Fatalf("HandlePaymentFailed() error: %v", err)
	}
	got, _ := srv.Get(context.Background(), o.ID())
	if got.Status() != StatusCancelled {
		t.Fatalf("status = %s, want CANCELLED", got.Status())
	}
	if got.CancelReason() != "payment failed: insufficient funds" {
		t.Fatalf("cancel reason = %q", got.CancelReason())
	}
	assertEvents(t, repo, EventOrderCreated, EventOrderCancelled)
}

func TestSagaIsIdempotent(t *testing.T) {
	srv, repo := newTestService()
	ctx := context.Background()
	o := placeOrder(t, srv)

	// Kafka delivers at least once: duplicates and late results are no-ops.
	_ = srv.HandlePaymentCompleted(ctx, o.ID(), "pay-1")
	_ = srv.HandlePaymentCompleted(ctx, o.ID(), "pay-1")
	_ = srv.HandlePaymentFailed(ctx, o.ID(), "late failure")

	got, _ := srv.Get(ctx, o.ID())
	if got.Status() != StatusPaid {
		t.Fatalf("status = %s, want PAID", got.Status())
	}
	assertEvents(t, repo, EventOrderCreated, EventOrderPaid)

	if err := srv.HandlePaymentCompleted(ctx, uuid.NewString(), "pay-x"); err != nil {
		t.Fatalf("unknown order must be skipped, got %v", err)
	}
}

func TestManualStatusChanges(t *testing.T) {
	srv, repo := newTestService()
	ctx := context.Background()
	o := placeOrder(t, srv)

	if _, err := srv.ChangeStatus(ctx, o.ID(), StatusPaid); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("PENDING -> PAID must be reserved to the saga, got %v", err)
	}

	_ = srv.HandlePaymentCompleted(ctx, o.ID(), "pay-1")
	for _, s := range []Status{StatusShipped, StatusDelivered} {
		if _, err := srv.ChangeStatus(ctx, o.ID(), s); err != nil {
			t.Fatalf("ChangeStatus(%s) error: %v", s, err)
		}
	}
	if _, err := srv.ChangeStatus(ctx, o.ID(), StatusCancelled); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("DELIVERED -> CANCELLED error = %v, want %v", err, ErrInvalidTransition)
	}
	assertEvents(t, repo, EventOrderCreated, EventOrderPaid, EventOrderStatusChanged, EventOrderStatusChanged)
}

func TestCancelPaidOrderRequestsRefund(t *testing.T) {
	srv, repo := newTestService()
	ctx := context.Background()
	o := placeOrder(t, srv)
	_ = srv.HandlePaymentCompleted(ctx, o.ID(), "pay-1")

	if _, err := srv.ChangeStatus(ctx, o.ID(), StatusCancelled); err != nil {
		t.Fatalf("ChangeStatus(CANCELLED) error: %v", err)
	}
	last := repo.outbox[len(repo.outbox)-1]
	p, ok := last.Payload.(OrderCancelledPayload)
	if !ok || last.Type != EventOrderCancelled || p.PaymentID != "pay-1" {
		t.Fatalf("last event = %+v, want OrderCancelled with paymentId for refund", last)
	}
}

func TestListByUser(t *testing.T) {
	srv, _ := newTestService()
	ctx := context.Background()
	alice, bob := uuid.NewString(), uuid.NewString()

	for _, u := range []string{alice, alice, bob} {
		if _, err := srv.Place(ctx, u, []Item{mustItem(t, p1, 1)}); err != nil {
			t.Fatalf("Place() error: %v", err)
		}
	}
	orders, err := srv.ListByUser(ctx, alice)
	if err != nil {
		t.Fatalf("ListByUser() error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("ListByUser() = %d orders, want 2", len(orders))
	}
}
