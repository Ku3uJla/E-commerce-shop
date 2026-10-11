package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"catalog_service/internal/domain/product"
)

const defaultListLimit = 50

type productRepository struct {
	db *gorm.DB
}

// NewProductRepo builds a product repository backed by GORM.
func NewProductRepo(db *gorm.DB) *productRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, p *product.Product) error {
	return r.db.WithContext(ctx).Create(toModel(p)).Error
}

func (r *productRepository) GetByID(ctx context.Context, id product.ProductID) (*product.Product, error) {
	var m ProductModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id.String()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, product.ErrProductNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *productRepository) GetByIDs(ctx context.Context, ids []product.ProductID) ([]*product.Product, error) {
	raw := make([]string, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, id.String())
	}
	var models []ProductModel
	if err := r.db.WithContext(ctx).Where("id IN ?", raw).Find(&models).Error; err != nil {
		return nil, err
	}
	products := make([]*product.Product, 0, len(models))
	for i := range models {
		if p := toDomain(&models[i]); p != nil {
			products = append(products, p)
		}
	}
	return products, nil
}

func (r *productRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&ProductModel{}).Count(&total).Error
	return total, err
}

func (r *productRepository) List(ctx context.Context, limit, offset int) ([]*product.Product, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	var models []ProductModel
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	products := make([]*product.Product, 0, len(models))
	for i := range models {
		if p := toDomain(&models[i]); p != nil {
			products = append(products, p)
		}
	}
	return products, nil
}

func (r *productRepository) Update(ctx context.Context, p *product.Product) error {
	res := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ?", p.ID().String()).
		Updates(map[string]any{
			"name":        p.Name().String(),
			"description": p.Description().String(),
			"price":       p.Price().Int64(),
			"stock":       p.Stock().Int(),
			"updated_at":  p.UpdatedAt(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return product.ErrProductNotFound
	}
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id product.ProductID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id.String()).Delete(&ProductModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return product.ErrProductNotFound
	}
	return nil
}
