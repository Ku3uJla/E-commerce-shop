package order

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
)

// Repository persists orders. Create and Update must store the order and the
// given events (transactional outbox) in a single transaction.
type Repository interface {
	Create(ctx context.Context, order *Order, events []Event) error
	Update(ctx context.Context, order *Order, events []Event) error
	GetByID(ctx context.Context, id string) (*Order, error)
	ListByUser(ctx context.Context, userID string) ([]*Order, error)
}

// PriceProvider returns current catalog prices (cents) for the given products.
// Unknown products are absent from the result.
type PriceProvider interface {
	Prices(ctx context.Context, productIDs []string) (map[string]int64, error)
}

// Service exposes the ordering use-cases to the transport layer.
type Service interface {
	Place(ctx context.Context, userID string, items []Item) (*Order, error)
	Get(ctx context.Context, id string) (*Order, error)
	ListByUser(ctx context.Context, userID string) ([]*Order, error)
	ChangeStatus(ctx context.Context, id string, status Status) (*Order, error)

	// Saga reactions to payment.events.
	HandlePaymentCompleted(ctx context.Context, orderID, paymentID string) error
	HandlePaymentFailed(ctx context.Context, orderID, reason string) error
}

type orderSrv struct {
	repo    Repository
	catalog PriceProvider
}

func NewService(repo Repository, catalog PriceProvider) *orderSrv {
	return &orderSrv{repo: repo, catalog: catalog}
}

// Place prices the items with the catalog, stores a PENDING order and an
// OrderCreated event that starts the saga.
func (srv *orderSrv) Place(ctx context.Context, userID string, items []Item) (*Order, error) {
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ProductID())
	}
	prices, err := srv.catalog.Prices(ctx, ids)
	if err != nil {
		return nil, err
	}

	o, err := New(userID, items, prices)
	if err != nil {
		return nil, err
	}
	if err := srv.repo.Create(ctx, o, o.PullEvents()); err != nil {
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

func (srv *orderSrv) ChangeStatus(ctx context.Context, id string, status Status) (*Order, error) {
	o, err := srv.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.ChangeStatus(status); err != nil {
		return nil, err
	}
	if err := srv.repo.Update(ctx, o, o.PullEvents()); err != nil {
		return nil, err
	}
	return o, nil
}

func (srv *orderSrv) HandlePaymentCompleted(ctx context.Context, orderID, paymentID string) error {
	return srv.react(ctx, orderID, func(o *Order) bool { return o.MarkPaid(paymentID) })
}

// HandlePaymentFailed runs the compensating transaction: cancel the order.
func (srv *orderSrv) HandlePaymentFailed(ctx context.Context, orderID, reason string) error {
	return srv.react(ctx, orderID, func(o *Order) bool { return o.CancelByPayment(reason) })
}

// react loads the order, applies a saga step and saves it. Events are
// delivered at least once, so a step that no longer applies is skipped, and a
// concurrent modification is retried with fresh state.
func (srv *orderSrv) react(ctx context.Context, orderID string, apply func(*Order) bool) error {
	for attempt := 0; attempt < 3; attempt++ {
		o, err := srv.repo.GetByID(ctx, orderID)
		if errors.Is(err, ErrOrderNotFound) {
			log.Printf("saga: order %s not found, event skipped", orderID)
			return nil
		}
		if err != nil {
			return err
		}
		if !apply(o) {
			log.Printf("saga: order %s is %s, event ignored", orderID, o.Status())
			return nil
		}
		err = srv.repo.Update(ctx, o, o.PullEvents())
		if !errors.Is(err, ErrConcurrentUpdate) {
			return err
		}
	}
	return ErrConcurrentUpdate
}
