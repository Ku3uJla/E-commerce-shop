package payment

import (
	"context"
	"errors"
	"log"
	"time"
)

// Repository persists payments. Create returns ErrAlreadyExists when the
// order already has a payment (unique order_id).
type Repository interface {
	Create(ctx context.Context, p *Payment) error
	Update(ctx context.Context, p *Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
}

// Publisher sends an event to payment.events keyed by order id.
type Publisher interface {
	Publish(ctx context.Context, eventType, key string, payload any) error
}

// Service implements the payment step of the saga.
type Service interface {
	HandleOrderCreated(ctx context.Context, e OrderCreatedPayload) error
	HandleOrderCancelled(ctx context.Context, e OrderCancelledPayload) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
}

type paymentSrv struct {
	repo      Repository
	gateway   Gateway
	publisher Publisher
}

func NewService(repo Repository, gateway Gateway, publisher Publisher) *paymentSrv {
	return &paymentSrv{repo: repo, gateway: gateway, publisher: publisher}
}

// HandleOrderCreated charges the customer and reports the outcome.
//
// Idempotency: the payment is stored before the result is published. If the
// event is redelivered (e.g. publishing failed or the service crashed), the
// stored payment is found and its outcome is re-published, so the customer is
// never charged twice.
func (srv *paymentSrv) HandleOrderCreated(ctx context.Context, e OrderCreatedPayload) error {
	existing, err := srv.repo.GetByOrderID(ctx, e.OrderID)
	switch {
	case err == nil:
		log.Printf("payment: order %s already processed (%s), re-publishing result", e.OrderID, existing.Status)
		return srv.publishResult(ctx, existing)
	case !errors.Is(err, ErrPaymentNotFound):
		return err
	}

	chargeErr := srv.gateway.Charge(ctx, e.UserID, e.Amount)
	if chargeErr != nil && !errors.Is(chargeErr, ErrDeclined) {
		return chargeErr // technical problem: retry the event
	}

	p := newPayment(e.OrderID, e.UserID, e.Amount, chargeErr)
	if err := srv.repo.Create(ctx, p); err != nil {
		if errors.Is(err, ErrAlreadyExists) { // a concurrent delivery won the race
			return srv.HandleOrderCreated(ctx, e)
		}
		return err
	}
	log.Printf("payment: order %s amount %d -> %s %s", e.OrderID, e.Amount, p.Status, p.Reason)
	return srv.publishResult(ctx, p)
}

// HandleOrderCancelled is the compensation on the payment side: when a paid
// order gets cancelled, the money is refunded.
func (srv *paymentSrv) HandleOrderCancelled(ctx context.Context, e OrderCancelledPayload) error {
	p, err := srv.repo.GetByOrderID(ctx, e.OrderID)
	if errors.Is(err, ErrPaymentNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	switch p.Status {
	case StatusFailed:
		return nil // nothing was charged
	case StatusRefunded:
		return srv.publishResult(ctx, p) // redelivery: make sure the result is out
	}

	if err := srv.gateway.Refund(ctx, p.ID, p.Amount); err != nil {
		return err
	}
	p.Status = StatusRefunded
	p.Reason = e.Reason
	p.UpdatedAt = time.Now().UTC()
	if err := srv.repo.Update(ctx, p); err != nil {
		return err
	}
	log.Printf("payment: order %s cancelled, payment %s refunded", e.OrderID, p.ID)
	return srv.publishResult(ctx, p)
}

func (srv *paymentSrv) GetByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	return srv.repo.GetByOrderID(ctx, orderID)
}

func (srv *paymentSrv) publishResult(ctx context.Context, p *Payment) error {
	eventType := map[Status]string{
		StatusCompleted: EventPaymentCompleted,
		StatusFailed:    EventPaymentFailed,
		StatusRefunded:  EventPaymentRefunded,
	}[p.Status]
	return srv.publisher.Publish(ctx, eventType, p.OrderID, ResultPayload{
		OrderID:   p.OrderID,
		UserID:    p.UserID,
		PaymentID: p.ID,
		Amount:    p.Amount,
		Reason:    p.Reason,
	})
}
