// Package kafka subscribes the order service to payment.events.
package kafka

import (
	"context"
	"encoding/json"

	"order_service/internal/domain/order"
	"order_service/internal/messaging"
)

// PaymentEventsHandler routes payment results to the order saga steps.
func PaymentEventsHandler(svc order.Service) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		var p order.PaymentResultPayload
		switch env.Type {
		case order.EventPaymentCompleted:
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				return nil // malformed payload: retrying will not help
			}
			return svc.HandlePaymentCompleted(ctx, p.OrderID, p.PaymentID)
		case order.EventPaymentFailed:
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				return nil
			}
			return svc.HandlePaymentFailed(ctx, p.OrderID, p.Reason)
		default:
			return nil // other payment events are not relevant for orders
		}
	}
}
