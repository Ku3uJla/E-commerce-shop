package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	items map[string]*Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]*Order{}}
}

func (r *fakeRepo) Create(_ context.Context, o *Order) error {
	r.items[o.ID()] = o
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

func mustItem(t *testing.T, productID string, qty int) Item {
	t.Helper()
	it, err := NewItem(productID, qty)
	if err != nil {
		t.Fatalf("NewItem() error: %v", err)
	}
	return it
}

func TestPlaceMergesDuplicateProducts(t *testing.T) {
	srv := NewService(newFakeRepo())
	p1, p2 := uuid.NewString(), uuid.NewString()

	o, err := srv.Place(context.Background(), uuid.NewString(), []Item{
		mustItem(t, p1, 1), mustItem(t, p2, 2), mustItem(t, p1, 3),
	})
	if err != nil {
		t.Fatalf("Place() error: %v", err)
	}
	if len(o.Items()) != 2 {
		t.Fatalf("Place() items = %d, want 2", len(o.Items()))
	}
	if o.Items()[0].Quantity() != 4 {
		t.Fatalf("merged quantity = %d, want 4", o.Items()[0].Quantity())
	}
}

func TestPlaceValidation(t *testing.T) {
	srv := NewService(newFakeRepo())
	ctx := context.Background()

	if _, err := srv.Place(ctx, uuid.NewString(), nil); !errors.Is(err, ErrEmptyOrder) {
		t.Fatalf("Place(no items) error = %v, want %v", err, ErrEmptyOrder)
	}
	if _, err := srv.Place(ctx, "bad", []Item{mustItem(t, uuid.NewString(), 1)}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Place(bad user) error = %v, want %v", err, ErrInvalidID)
	}
	if _, err := NewItem(uuid.NewString(), 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("NewItem(qty 0) error = %v, want %v", err, ErrInvalidQuantity)
	}
}

func TestListByUser(t *testing.T) {
	srv := NewService(newFakeRepo())
	ctx := context.Background()
	alice, bob := uuid.NewString(), uuid.NewString()

	for _, u := range []string{alice, alice, bob} {
		if _, err := srv.Place(ctx, u, []Item{mustItem(t, uuid.NewString(), 1)}); err != nil {
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
