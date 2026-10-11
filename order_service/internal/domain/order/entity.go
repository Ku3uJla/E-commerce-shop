package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const maxItemsPerOrder = 100

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrEmptyOrder        = errors.New("order must contain at least one item")
	ErrTooManyItems      = errors.New("order contains too many items")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrInvalidID         = errors.New("invalid id")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidTransition = errors.New("status transition is not allowed")
	ErrProductNotFound   = errors.New("product not found in catalog")
	ErrConcurrentUpdate  = errors.New("order was modified concurrently")
)

// Status is the lifecycle state of an order.
type Status string

const (
	StatusPending   Status = "PENDING"   // created, waiting for payment
	StatusPaid      Status = "PAID"      // payment completed
	StatusShipped   Status = "SHIPPED"   // handed over to delivery
	StatusDelivered Status = "DELIVERED" // received by the customer
	StatusCancelled Status = "CANCELLED" // payment failed or cancelled by request
)

// ParseStatus validates a raw status value.
func ParseStatus(raw string) (Status, error) {
	switch s := Status(raw); s {
	case StatusPending, StatusPaid, StatusShipped, StatusDelivered, StatusCancelled:
		return s, nil
	}
	return "", ErrInvalidStatus
}

// manualTransitions are the changes allowed through the API. PENDING → PAID
// is deliberately absent: only the payment saga may mark an order as paid.
var manualTransitions = map[Status][]Status{
	StatusPending: {StatusCancelled},
	StatusPaid:    {StatusShipped, StatusCancelled},
	StatusShipped: {StatusDelivered},
}

// Item is a single order line with the unit price captured at order time.
type Item struct {
	productID string
	quantity  int
	unitPrice int64 // cents
}

// NewItem validates and builds an order line (price is filled in later).
func NewItem(productID string, quantity int) (Item, error) {
	if _, err := uuid.Parse(productID); err != nil {
		return Item{}, ErrInvalidID
	}
	if quantity <= 0 {
		return Item{}, ErrInvalidQuantity
	}
	return Item{productID: productID, quantity: quantity}, nil
}

// ReconstituteItem rebuilds an order line from persisted state.
func ReconstituteItem(productID string, quantity int, unitPrice int64) Item {
	return Item{productID: productID, quantity: quantity, unitPrice: unitPrice}
}

func (i Item) ProductID() string { return i.productID }
func (i Item) Quantity() int     { return i.quantity }
func (i Item) UnitPrice() int64  { return i.unitPrice }

// Order is the aggregate root of the ordering domain. State changes record
// domain events which the repository stores in the outbox atomically.
type Order struct {
	id           string
	userID       string
	status       Status
	total        int64 // cents
	cancelReason string
	paymentID    string
	createdAt    time.Time
	updatedAt    time.Time
	version      int
	items        []Item

	events []Event
}

// New creates a PENDING order. Lines for the same product are merged and
// priced with the given catalog prices (cents).
func New(userID string, items []Item, prices map[string]int64) (*Order, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidID
	}
	if len(items) == 0 {
		return nil, ErrEmptyOrder
	}
	if len(items) > maxItemsPerOrder {
		return nil, ErrTooManyItems
	}

	merged := mergeItems(items)
	var total int64
	for i := range merged {
		price, ok := prices[merged[i].productID]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrProductNotFound, merged[i].productID)
		}
		merged[i].unitPrice = price
		total += price * int64(merged[i].quantity)
	}

	now := time.Now().UTC()
	o := &Order{
		id:        uuid.NewString(),
		userID:    userID,
		status:    StatusPending,
		total:     total,
		createdAt: now,
		updatedAt: now,
		items:     merged,
	}
	o.record(EventOrderCreated, o.createdPayload())
	return o, nil
}

// Reconstitute rebuilds an order from persisted state.
func Reconstitute(id, userID string, status Status, total int64, cancelReason, paymentID string,
	createdAt, updatedAt time.Time, version int, items []Item) *Order {
	return &Order{
		id: id, userID: userID, status: status, total: total,
		cancelReason: cancelReason, paymentID: paymentID,
		createdAt: createdAt, updatedAt: updatedAt, version: version, items: items,
	}
}

// MarkPaid is the happy path of the saga. It returns false when the order is
// no longer waiting for payment (duplicate or late event), which is a no-op.
func (o *Order) MarkPaid(paymentID string) bool {
	if o.status != StatusPending {
		return false
	}
	o.status = StatusPaid
	o.paymentID = paymentID
	o.touch()
	o.record(EventOrderPaid, OrderPaidPayload{
		OrderID: o.id, UserID: o.userID, PaymentID: paymentID, Amount: o.total,
	})
	return true
}

// CancelByPayment is the compensating action of the saga: the payment failed,
// so the order is cancelled. No-op unless the order is still PENDING.
func (o *Order) CancelByPayment(reason string) bool {
	if o.status != StatusPending {
		return false
	}
	o.cancel("payment failed: " + reason)
	return true
}

// ChangeStatus applies a manual status change requested through the API.
func (o *Order) ChangeStatus(target Status) error {
	if target == o.status {
		return nil
	}
	allowed := false
	for _, s := range manualTransitions[o.status] {
		if s == target {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, o.status, target)
	}

	if target == StatusCancelled {
		o.cancel("cancelled by request")
		return nil
	}
	o.status = target
	o.touch()
	o.record(EventOrderStatusChanged, OrderStatusChangedPayload{
		OrderID: o.id, UserID: o.userID, Status: string(target),
	})
	return nil
}

func (o *Order) cancel(reason string) {
	o.status = StatusCancelled
	o.cancelReason = reason
	o.touch()
	// When the order was already paid, Payment reacts to this event with a refund.
	o.record(EventOrderCancelled, OrderCancelledPayload{
		OrderID: o.id, UserID: o.userID, Reason: reason, PaymentID: o.paymentID,
	})
}

func (o *Order) touch() { o.updatedAt = time.Now().UTC() }

func (o *Order) record(eventType string, payload any) {
	o.events = append(o.events, Event{Type: eventType, AggregateID: o.id, Payload: payload})
}

// PullEvents returns and clears the recorded domain events.
func (o *Order) PullEvents() []Event {
	events := o.events
	o.events = nil
	return events
}

func (o *Order) createdPayload() OrderCreatedPayload {
	items := make([]OrderItemPayload, 0, len(o.items))
	for _, it := range o.items {
		items = append(items, OrderItemPayload{ProductID: it.productID, Quantity: it.quantity, UnitPrice: it.unitPrice})
	}
	return OrderCreatedPayload{OrderID: o.id, UserID: o.userID, Amount: o.total, Items: items}
}

func (o *Order) ID() string           { return o.id }
func (o *Order) UserID() string       { return o.userID }
func (o *Order) Status() Status       { return o.status }
func (o *Order) Total() int64         { return o.total }
func (o *Order) CancelReason() string { return o.cancelReason }
func (o *Order) PaymentID() string    { return o.paymentID }
func (o *Order) CreatedAt() time.Time { return o.createdAt }
func (o *Order) UpdatedAt() time.Time { return o.updatedAt }
func (o *Order) Version() int         { return o.version }
func (o *Order) Items() []Item        { return o.items }

func mergeItems(items []Item) []Item {
	index := make(map[string]int, len(items))
	merged := make([]Item, 0, len(items))
	for _, it := range items {
		if pos, ok := index[it.productID]; ok {
			merged[pos].quantity += it.quantity
			continue
		}
		index[it.productID] = len(merged)
		merged = append(merged, it)
	}
	return merged
}
