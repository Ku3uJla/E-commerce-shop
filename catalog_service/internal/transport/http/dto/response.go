package dto

import "time"

// ProductResponse is the public representation of a product.
type ProductResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Price       int64      `json:"price"`
	Stock       int        `json:"stock"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// ProductListResponse wraps a page of products.
type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
	Count int               `json:"count"`
}

// ErrorResponse is the unified error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
