package dto

// CreateProductRequest is the payload accepted by the create endpoint.
// Price is expressed in minor units (e.g. cents).
type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
}

// UpdateProductRequest is the payload accepted by the update endpoint.
type UpdateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
}
