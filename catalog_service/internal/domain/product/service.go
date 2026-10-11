package product

import (
	"context"
	"errors"
)

// ErrProductNotFound is returned when a product does not exist.
var ErrProductNotFound = errors.New("product not found")

// Service exposes the catalog use-cases to the transport layer.
type Service interface {
	Create(ctx context.Context, name Name, description Description, price Price, stock Stock) (*Product, error)
	Get(ctx context.Context, id ProductID) (*Product, error)
	GetMany(ctx context.Context, ids []ProductID) ([]*Product, error)
	List(ctx context.Context, limit, offset int) ([]*Product, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, id ProductID, name Name, description Description, price Price, stock Stock) (*Product, error)
	Delete(ctx context.Context, id ProductID) error
}

type productSrv struct {
	repo Repository
}

func NewService(repo Repository) *productSrv {
	return &productSrv{repo: repo}
}

func (srv *productSrv) Create(ctx context.Context, name Name, description Description, price Price, stock Stock) (*Product, error) {
	p := New(name, description, price, stock)
	if err := srv.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (srv *productSrv) Get(ctx context.Context, id ProductID) (*Product, error) {
	return srv.repo.GetByID(ctx, id)
}

// GetMany returns the products with the given ids; unknown ids are skipped.
func (srv *productSrv) GetMany(ctx context.Context, ids []ProductID) ([]*Product, error) {
	if len(ids) == 0 {
		return []*Product{}, nil
	}
	return srv.repo.GetByIDs(ctx, ids)
}

func (srv *productSrv) Count(ctx context.Context) (int64, error) {
	return srv.repo.Count(ctx)
}

func (srv *productSrv) List(ctx context.Context, limit, offset int) ([]*Product, error) {
	return srv.repo.List(ctx, limit, offset)
}

func (srv *productSrv) Update(ctx context.Context, id ProductID, name Name, description Description, price Price, stock Stock) (*Product, error) {
	p, err := srv.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Update(name, description, price, stock)
	if err := srv.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (srv *productSrv) Delete(ctx context.Context, id ProductID) error {
	return srv.repo.Delete(ctx, id)
}
