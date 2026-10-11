package product

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	items map[string]*Product
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]*Product{}}
}

func (r *fakeRepo) Create(_ context.Context, p *Product) error {
	r.items[p.ID().String()] = p
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, id ProductID) (*Product, error) {
	if p, ok := r.items[id.String()]; ok {
		return p, nil
	}
	return nil, ErrProductNotFound
}

func (r *fakeRepo) List(_ context.Context, limit, offset int) ([]*Product, error) {
	all := make([]*Product, 0, len(r.items))
	for _, p := range r.items {
		all = append(all, p)
	}
	if offset >= len(all) {
		return []*Product{}, nil
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (r *fakeRepo) GetByIDs(_ context.Context, ids []ProductID) ([]*Product, error) {
	out := make([]*Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.items[id.String()]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.items)), nil
}

func (r *fakeRepo) Update(_ context.Context, p *Product) error {
	if _, ok := r.items[p.ID().String()]; !ok {
		return ErrProductNotFound
	}
	r.items[p.ID().String()] = p
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, id ProductID) error {
	if _, ok := r.items[id.String()]; !ok {
		return ErrProductNotFound
	}
	delete(r.items, id.String())
	return nil
}

func mustName(t *testing.T, value string) Name {
	t.Helper()
	name, err := NewName(value)
	if err != nil {
		t.Fatalf("NewName(%q) error: %v", value, err)
	}
	return name
}

func mustDescription(t *testing.T, value string) Description {
	t.Helper()
	description, err := NewDescription(value)
	if err != nil {
		t.Fatalf("NewDescription(%q) error: %v", value, err)
	}
	return description
}

func mustPrice(t *testing.T, value int64) Price {
	t.Helper()
	price, err := NewPrice(value)
	if err != nil {
		t.Fatalf("NewPrice(%d) error: %v", value, err)
	}
	return price
}

func mustStock(t *testing.T, value int) Stock {
	t.Helper()
	stock, err := NewStock(value)
	if err != nil {
		t.Fatalf("NewStock(%d) error: %v", value, err)
	}
	return stock
}

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	srv := NewService(newFakeRepo())

	created, err := srv.Create(ctx, mustName(t, "Widget"), mustDescription(t, "A widget"), mustPrice(t, 1999), mustStock(t, 10))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := srv.Get(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name().String() != "Widget" {
		t.Fatalf("Get().Name() = %q, want %q", got.Name().String(), "Widget")
	}
	if got.Price().Int64() != 1999 {
		t.Fatalf("Get().Price() = %d, want 1999", got.Price().Int64())
	}
}

func TestGetNotFound(t *testing.T) {
	ctx := context.Background()
	srv := NewService(newFakeRepo())

	if _, err := srv.Get(ctx, GenerateProductID()); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrProductNotFound)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	srv := NewService(newFakeRepo())

	for i := 0; i < 3; i++ {
		if _, err := srv.Create(ctx, mustName(t, "P"), mustDescription(t, ""), mustPrice(t, 100), mustStock(t, i)); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	items, err := srv.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(items))
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	srv := NewService(newFakeRepo())

	created, err := srv.Create(ctx, mustName(t, "Old"), mustDescription(t, ""), mustPrice(t, 100), mustStock(t, 1))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	updated, err := srv.Update(ctx, created.ID(), mustName(t, "New"), mustDescription(t, "desc"), mustPrice(t, 500), mustStock(t, 7))
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.Name().String() != "New" || updated.Price().Int64() != 500 || updated.Stock().Int() != 7 {
		t.Fatalf("Update() did not apply changes: %+v", updated)
	}
	if updated.UpdatedAt() == nil {
		t.Fatal("Update() did not set updatedAt")
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	srv := NewService(newFakeRepo())

	created, err := srv.Create(ctx, mustName(t, "Gone"), mustDescription(t, ""), mustPrice(t, 100), mustStock(t, 1))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := srv.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, err := srv.Get(ctx, created.ID()); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want %v", err, ErrProductNotFound)
	}
}
