package payment

// Kafka topics and event types of the order saga (see order_service).
const (
	TopicOrderEvents   = "order.events"
	TopicPaymentEvents = "payment.events"

	EventOrderCreated   = "OrderCreated"
	EventOrderCancelled = "OrderCancelled"

	EventPaymentCompleted = "PaymentCompleted"
	EventPaymentFailed    = "PaymentFailed"
	EventPaymentRefunded  = "PaymentRefunded"
)

// OrderCreatedPayload is consumed from order.events.
type OrderCreatedPayload struct {
	OrderID string `json:"orderId"`
	UserID  string `json:"userId"`
	Amount  int64  `json:"amount"`
}

// OrderCancelledPayload is consumed from order.events.
type OrderCancelledPayload struct {
	OrderID string `json:"orderId"`
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
}

// ResultPayload is published to payment.events for every payment outcome.
type ResultPayload struct {
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	PaymentID string `json:"paymentId"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason,omitempty"`
}
