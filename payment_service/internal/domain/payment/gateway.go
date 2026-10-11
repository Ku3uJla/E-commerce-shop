package payment

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
)

// ErrDeclined is wrapped by every business decline of the gateway.
var ErrDeclined = errors.New("payment declined")

// Gateway charges and refunds money with an external payment provider.
// A returned error wrapping ErrDeclined is a final business decline; any
// other error is technical and the saga step is retried.
type Gateway interface {
	Charge(ctx context.Context, userID string, amount int64) error
	Refund(ctx context.Context, paymentID string, amount int64) error
}

// SimulatedGateway stands in for a real provider: it declines payments above
// maxAmount and a random share failureRate of the others.
type SimulatedGateway struct {
	maxAmount   int64
	failureRate float64
}

func NewSimulatedGateway(maxAmount int64, failureRate float64) *SimulatedGateway {
	return &SimulatedGateway{maxAmount: maxAmount, failureRate: failureRate}
}

func (g *SimulatedGateway) Charge(_ context.Context, _ string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("%w: invalid amount", ErrDeclined)
	}
	if amount > g.maxAmount {
		return fmt.Errorf("%w: insufficient funds (amount %.2f exceeds limit %.2f)",
			ErrDeclined, float64(amount)/100, float64(g.maxAmount)/100)
	}
	if g.failureRate > 0 && rand.Float64() < g.failureRate {
		return fmt.Errorf("%w: rejected by issuing bank", ErrDeclined)
	}
	return nil
}

func (g *SimulatedGateway) Refund(context.Context, string, int64) error { return nil }
