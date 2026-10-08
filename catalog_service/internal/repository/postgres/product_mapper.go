package repository

import (
	"github.com/google/uuid"

	"catalog_service/internal/domain/product"
)

// toModel converts a domain product into its persistence representation.
func toModel(p *product.Product) *ProductModel {
	return &ProductModel{
		ID:          uuid.MustParse(p.ID().String()),
		Name:        p.Name().String(),
		Description: p.Description().String(),
		Price:       p.Price().Int64(),
		Stock:       p.Stock().Int(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

// toDomain rehydrates a domain product from its persistence representation.
func toDomain(m *ProductModel) *product.Product {
	id, err := product.NewProductID(m.ID.String())
	if err != nil {
		id = product.GenerateProductID()
	}
	name, err := product.NewName(m.Name)
	if err != nil {
		return nil
	}
	description, err := product.NewDescription(m.Description)
	if err != nil {
		return nil
	}
	price, err := product.NewPrice(m.Price)
	if err != nil {
		return nil
	}
	stock, err := product.NewStock(m.Stock)
	if err != nil {
		return nil
	}
	return product.Reconstitute(id, name, description, price, stock, m.CreatedAt, m.UpdatedAt)
}
