package http

import (
	"context"
	"net/http/httptest"

	"catalog_service/internal/domain/product"
	"catalog_service/internal/transport/http/handler"
)

// memRepo is an in-memory product.Repository used for transport tests.
type memRepo struct {
	items map[string]*product.Product
}

func newMemRepo() *memRepo {
	return &memRepo{items: map[string]*product.Product{}}
}

func (r *memRepo) Create(_ context.Context, p *product.Product) error {
	r.items[p.ID().String()] = p
	return nil
}

func (r *memRepo) GetByID(_ context.Context, id product.ProductID) (*product.Product, error) {
	if p, ok := r.items[id.String()]; ok {
		return p, nil
	}
	return nil, product.ErrProductNotFound
}

func (r *memRepo) List(_ context.Context, limit, offset int) ([]*product.Product, error) {
	all := make([]*product.Product, 0, len(r.items))
	for _, p := range r.items {
		all = append(all, p)
	}
	if offset >= len(all) {
		return []*product.Product{}, nil
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (r *memRepo) GetByIDs(_ context.Context, ids []product.ProductID) ([]*product.Product, error) {
	out := make([]*product.Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.items[id.String()]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *memRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.items)), nil
}

func (r *memRepo) Update(_ context.Context, p *product.Product) error {
	if _, ok := r.items[p.ID().String()]; !ok {
		return product.ErrProductNotFound
	}
	r.items[p.ID().String()] = p
	return nil
}

func (r *memRepo) Delete(_ context.Context, id product.ProductID) error {
	if _, ok := r.items[id.String()]; !ok {
		return product.ErrProductNotFound
	}
	delete(r.items, id.String())
	return nil
}

func newTestServer() (*httptest.Server, string) {
	const instanceID = "instance-1"
	svc := product.NewService(newMemRepo())
	router := NewRouter(handler.New(svc, instanceID), instanceID)
	return httptest.NewServer(router), instanceID
}
