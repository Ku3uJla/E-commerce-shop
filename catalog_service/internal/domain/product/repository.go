package product

import "context"

// Repository persists products.
type Repository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id ProductID) (*Product, error)
	GetByIDs(ctx context.Context, ids []ProductID) ([]*Product, error)
	List(ctx context.Context, limit, offset int) ([]*Product, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id ProductID) error
}
