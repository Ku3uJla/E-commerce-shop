package order

// Kafka topics and event types of the order saga. The JSON payloads are the
// public contract shared with the Payment and Notification services.
const (
	TopicOrderEvents   = "order.events"
	TopicPaymentEvents = "payment.events"

	EventOrderCreated       = "OrderCreated"
	EventOrderPaid          = "OrderPaid"
	EventOrderCancelled     = "OrderCancelled"
	EventOrderStatusChanged = "OrderStatusChanged"

	EventPaymentCompleted = "PaymentCompleted"
	EventPaymentFailed    = "PaymentFailed"
)

// Event is a domain event waiting to be stored in the outbox.
type Event struct {
	Type        string
	AggregateID string // Kafka message key: keeps the events of one order in order
	Payload     any
}

type OrderItemPayload struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

type OrderCreatedPayload struct {
	OrderID string             `json:"orderId"`
	UserID  string             `json:"userId"`
	Amount  int64              `json:"amount"`
	Items   []OrderItemPayload `json:"items"`
}

type OrderPaidPayload struct {
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	PaymentID string `json:"paymentId"`
	Amount    int64  `json:"amount"`
}

type OrderCancelledPayload struct {
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	Reason    string `json:"reason"`
	PaymentID string `json:"paymentId,omitempty"` // set when a refund is required
}

type OrderStatusChangedPayload struct {
	OrderID string `json:"orderId"`
	UserID  string `json:"userId"`
	Status  string `json:"status"`
}

// PaymentResultPayload is consumed from payment.events.
type PaymentResultPayload struct {
	OrderID   string `json:"orderId"`
	PaymentID string `json:"paymentId"`
	Reason    string `json:"reason"`
}
