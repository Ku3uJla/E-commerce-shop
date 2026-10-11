package dto

import "time"

// ItemRequest is a single order line in a create request.
type ItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderRequest is the payload accepted by the create endpoint.
type CreateOrderRequest struct {
	UserID string        `json:"userId"`
	Items  []ItemRequest `json:"items"`
}

// ChangeStatusRequest is the payload accepted by the status endpoint.
type ChangeStatusRequest struct {
	Status string `json:"status"`
}

// ItemResponse is the public representation of an order line.
type ItemResponse struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

// OrderResponse is the public representation of an order.
type OrderResponse struct {
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	Status       string         `json:"status"`
	Total        int64          `json:"total"`
	CancelReason string         `json:"cancelReason,omitempty"`
	PaymentID    string         `json:"paymentId,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Items        []ItemResponse `json:"items"`
}

// OrderListResponse wraps a list of orders.
type OrderListResponse struct {
	Items []OrderResponse `json:"items"`
	Count int             `json:"count"`
}

// ErrorResponse is the unified error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}
