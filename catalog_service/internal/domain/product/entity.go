package product

import "time"

// Product is the aggregate root of the catalog domain.
type Product struct {
	id          ProductID
	name        Name
	description Description
	price       Price
	stock       Stock
	createdAt   time.Time
	updatedAt   *time.Time
}

// New creates a brand new product.
func New(name Name, description Description, price Price, stock Stock) *Product {
	return &Product{
		id:          GenerateProductID(),
		name:        name,
		description: description,
		price:       price,
		stock:       stock,
		createdAt:   time.Now().UTC(),
	}
}

// Reconstitute rebuilds a product from persisted state.
func Reconstitute(
	id ProductID,
	name Name,
	description Description,
	price Price,
	stock Stock,
	createdAt time.Time,
	updatedAt *time.Time,
) *Product {
	return &Product{
		id:          id,
		name:        name,
		description: description,
		price:       price,
		stock:       stock,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Update mutates the mutable attributes of the product.
func (p *Product) Update(name Name, description Description, price Price, stock Stock) {
	now := time.Now().UTC()
	p.name = name
	p.description = description
	p.price = price
	p.stock = stock
	p.updatedAt = &now
}

func (p *Product) ID() ProductID            { return p.id }
func (p *Product) Name() Name               { return p.name }
func (p *Product) Description() Description { return p.description }
func (p *Product) Price() Price             { return p.price }
func (p *Product) Stock() Stock             { return p.stock }
func (p *Product) CreatedAt() time.Time     { return p.createdAt }
func (p *Product) UpdatedAt() *time.Time    { return p.updatedAt }
