// Package kafka connects the payment domain to Kafka.
package kafka

import (
	"context"
	"encoding/json"

	"payment_service/internal/domain/payment"
	"payment_service/internal/messaging"
)

// OrderEventsHandler routes order.events to the payment saga steps.
func OrderEventsHandler(svc payment.Service) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		switch env.Type {
		case payment.EventOrderCreated:
			var e payment.OrderCreatedPayload
			if err := json.Unmarshal(env.Payload, &e); err != nil {
				return nil // malformed payload: retrying will not help
			}
			return svc.HandleOrderCreated(ctx, e)
		case payment.EventOrderCancelled:
			var e payment.OrderCancelledPayload
			if err := json.Unmarshal(env.Payload, &e); err != nil {
				return nil
			}
			return svc.HandleOrderCancelled(ctx, e)
		default:
			return nil
		}
	}
}

// Publisher adapts messaging.Publisher to payment.Publisher.
type Publisher struct {
	pub *messaging.Publisher
}

func NewPublisher(pub *messaging.Publisher) *Publisher {
	return &Publisher{pub: pub}
}

func (p *Publisher) Publish(ctx context.Context, eventType, key string, payload any) error {
	env, err := messaging.NewEnvelope("payment-service", eventType, payload)
	if err != nil {
		return err
	}
	return p.pub.Publish(ctx, messaging.Message{Topic: payment.TopicPaymentEvents, Key: key, Envelope: env})
}
