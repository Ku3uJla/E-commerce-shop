package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"order_service/internal/domain/order"
)

// OrderModel is the GORM persistence model for the orders table.
type OrderModel struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID        `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time        `gorm:"not null"`
	Items     []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

func (OrderModel) TableName() string { return "orders" }

// OrderItemModel is the GORM persistence model for the order_items table.
type OrderItemModel struct {
	ID        uint      `gorm:"primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductID uuid.UUID `gorm:"type:uuid;not null"`
	Quantity  int       `gorm:"not null"`
}

func (OrderItemModel) TableName() string { return "order_items" }

// Connect opens a PostgreSQL connection with the given DSN.
func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// EnsureDatabase creates the database if it does not exist yet. initdb scripts
// only run on an empty data volume, so a new service cannot rely on them.
func EnsureDatabase(adminDSN, name string) error {
	db, err := Connect(adminDSN)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var exists bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", name).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}
	return db.Exec("CREATE DATABASE " + pgx.Identifier{name}.Sanitize()).Error
}

// AutoMigrate creates or updates the orders schema.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&OrderModel{}, &OrderItemModel{})
}

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepo builds an order repository backed by GORM.
func NewOrderRepo(db *gorm.DB) *orderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, o *order.Order) error {
	m := OrderModel{
		ID:        uuid.MustParse(o.ID()),
		UserID:    uuid.MustParse(o.UserID()),
		CreatedAt: o.CreatedAt(),
	}
	for _, it := range o.Items() {
		m.Items = append(m.Items, OrderItemModel{
			ProductID: uuid.MustParse(it.ProductID()),
			Quantity:  it.Quantity(),
		})
	}
	// GORM saves the order and its items in a single transaction.
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*order.Order, error) {
	var m OrderModel
	err := r.db.WithContext(ctx).Preload("Items", orderItems).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID string) ([]*order.Order, error) {
	var models []OrderModel
	err := r.db.WithContext(ctx).
		Preload("Items", orderItems).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	orders := make([]*order.Order, 0, len(models))
	for i := range models {
		orders = append(orders, toDomain(&models[i]))
	}
	return orders, nil
}

func orderItems(db *gorm.DB) *gorm.DB {
	return db.Order("order_items.id")
}

func toDomain(m *OrderModel) *order.Order {
	items := make([]order.Item, 0, len(m.Items))
	for _, it := range m.Items {
		if item, err := order.NewItem(it.ProductID.String(), it.Quantity); err == nil {
			items = append(items, item)
		}
	}
	return order.Reconstitute(m.ID.String(), m.UserID.String(), m.CreatedAt, items)
}
