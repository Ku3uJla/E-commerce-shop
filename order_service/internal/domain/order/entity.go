package order

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const maxItemsPerOrder = 100

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrEmptyOrder      = errors.New("order must contain at least one item")
	ErrTooManyItems    = errors.New("order contains too many items")
	ErrInvalidQuantity = errors.New("quantity must be positive")
	ErrInvalidID       = errors.New("invalid id")
)

// Item is a single order line: which product and how many of it.
type Item struct {
	productID string
	quantity  int
}

// NewItem validates and builds an order line.
func NewItem(productID string, quantity int) (Item, error) {
	if _, err := uuid.Parse(productID); err != nil {
		return Item{}, ErrInvalidID
	}
	if quantity <= 0 {
		return Item{}, ErrInvalidQuantity
	}
	return Item{productID: productID, quantity: quantity}, nil
}

func (i Item) ProductID() string { return i.productID }
func (i Item) Quantity() int     { return i.quantity }

// Order is the aggregate root of the ordering domain.
type Order struct {
	id        string
	userID    string
	createdAt time.Time
	items     []Item
}

// New creates a brand new order. Lines referring to the same product are merged.
func New(userID string, items []Item) (*Order, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidID
	}
	if len(items) == 0 {
		return nil, ErrEmptyOrder
	}
	if len(items) > maxItemsPerOrder {
		return nil, ErrTooManyItems
	}
	return &Order{
		id:        uuid.NewString(),
		userID:    userID,
		createdAt: time.Now().UTC(),
		items:     mergeItems(items),
	}, nil
}

// Reconstitute rebuilds an order from persisted state.
func Reconstitute(id, userID string, createdAt time.Time, items []Item) *Order {
	return &Order{id: id, userID: userID, createdAt: createdAt, items: items}
}

func (o *Order) ID() string           { return o.id }
func (o *Order) UserID() string       { return o.userID }
func (o *Order) CreatedAt() time.Time { return o.createdAt }
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
