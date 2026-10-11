package payment

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrAlreadyExists   = errors.New("payment for this order already exists")
)

// Status is the state of a payment.
type Status string

const (
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
	StatusRefunded  Status = "REFUNDED"
)

// Payment is the result of charging the customer for one order. There is at
// most one payment per order, which makes the saga step idempotent.
type Payment struct {
	ID        string
	OrderID   string
	UserID    string
	Amount    int64 // cents
	Status    Status
	Reason    string // why the payment failed
	CreatedAt time.Time
	UpdatedAt time.Time
}

func newPayment(orderID, userID string, amount int64, chargeErr error) *Payment {
	now := time.Now().UTC()
	p := &Payment{
		ID:        uuid.NewString(),
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    StatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if chargeErr != nil {
		p.Status = StatusFailed
		p.Reason = chargeErr.Error()
	}
	return p
}
