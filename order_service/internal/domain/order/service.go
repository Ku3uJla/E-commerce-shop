package order

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists orders.
type Repository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	ListByUser(ctx context.Context, userID string) ([]*Order, error)
}

// Service exposes the ordering use-cases to the transport layer.
type Service interface {
	Place(ctx context.Context, userID string, items []Item) (*Order, error)
	Get(ctx context.Context, id string) (*Order, error)
	ListByUser(ctx context.Context, userID string) ([]*Order, error)
}

type orderSrv struct {
	repo Repository
}

func NewService(repo Repository) *orderSrv {
	return &orderSrv{repo: repo}
}

func (srv *orderSrv) Place(ctx context.Context, userID string, items []Item) (*Order, error) {
	o, err := New(userID, items)
	if err != nil {
		return nil, err
	}
	if err := srv.repo.Create(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (srv *orderSrv) Get(ctx context.Context, id string) (*Order, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidID
	}
	return srv.repo.GetByID(ctx, id)
}

func (srv *orderSrv) ListByUser(ctx context.Context, userID string) ([]*Order, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidID
	}
	return srv.repo.ListByUser(ctx, userID)
}
