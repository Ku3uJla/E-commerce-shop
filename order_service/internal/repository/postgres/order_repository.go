package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"order_service/internal/domain/order"
	"order_service/internal/messaging"
)

// OrderModel is the GORM persistence model for the orders table.
type OrderModel struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID        `gorm:"type:uuid;not null;index"`
	Status       string           `gorm:"size:20;not null;default:PENDING;index"`
	Total        int64            `gorm:"not null;default:0"`
	CancelReason string           `gorm:"size:255"`
	PaymentID    string           `gorm:"size:64"`
	CreatedAt    time.Time        `gorm:"not null"`
	UpdatedAt    time.Time        `gorm:"not null"`
	Version      int              `gorm:"not null;default:0"`
	Items        []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

func (OrderModel) TableName() string { return "orders" }

// OrderItemModel is the GORM persistence model for the order_items table.
type OrderItemModel struct {
	ID        uint      `gorm:"primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductID uuid.UUID `gorm:"type:uuid;not null"`
	Quantity  int       `gorm:"not null"`
	UnitPrice int64     `gorm:"not null;default:0"`
}

func (OrderItemModel) TableName() string { return "order_items" }

// OutboxModel is an event waiting to be published to Kafka. It is written in
// the same transaction as the order change (transactional outbox pattern).
type OutboxModel struct {
	Seq         int64     `gorm:"primaryKey;autoIncrement"`
	EventID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Topic       string    `gorm:"size:100;not null"`
	Key         string    `gorm:"size:100;not null"`
	Envelope    []byte    `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
	PublishedAt *time.Time
}

func (OutboxModel) TableName() string { return "outbox" }

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
	return db.AutoMigrate(&OrderModel{}, &OrderItemModel{}, &OutboxModel{})
}

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepo builds an order repository backed by GORM.
func NewOrderRepo(db *gorm.DB) *orderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, o *order.Order, events []order.Event) error {
	m := OrderModel{
		ID:        uuid.MustParse(o.ID()),
		UserID:    uuid.MustParse(o.UserID()),
		Status:    string(o.Status()),
		Total:     o.Total(),
		CreatedAt: o.CreatedAt(),
		UpdatedAt: o.UpdatedAt(),
	}
	for _, it := range o.Items() {
		m.Items = append(m.Items, OrderItemModel{
			ProductID: uuid.MustParse(it.ProductID()),
			Quantity:  it.Quantity(),
			UnitPrice: it.UnitPrice(),
		})
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return insertOutbox(tx, events)
	})
}

// Update saves the mutable fields with optimistic locking and the events.
func (r *orderRepository) Update(ctx context.Context, o *order.Order, events []order.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&OrderModel{}).
			Where("id = ? AND version = ?", o.ID(), o.Version()).
			Updates(map[string]any{
				"status":        string(o.Status()),
				"cancel_reason": o.CancelReason(),
				"payment_id":    o.PaymentID(),
				"updated_at":    o.UpdatedAt(),
				"version":       gorm.Expr("version + 1"),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return order.ErrConcurrentUpdate
		}
		return insertOutbox(tx, events)
	})
}

func insertOutbox(tx *gorm.DB, events []order.Event) error {
	for _, e := range events {
		env, err := messaging.NewEnvelope("order-service", e.Type, e.Payload)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(env)
		if err != nil {
			return err
		}
		row := OutboxModel{
			EventID:   uuid.MustParse(env.ID),
			Topic:     order.TopicOrderEvents,
			Key:       e.AggregateID,
			Envelope:  raw,
			CreatedAt: env.OccurredAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
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
		items = append(items, order.ReconstituteItem(it.ProductID.String(), it.Quantity, it.UnitPrice))
	}
	return order.Reconstitute(
		m.ID.String(), m.UserID.String(), order.Status(m.Status), m.Total,
		m.CancelReason, m.PaymentID, m.CreatedAt, m.UpdatedAt, m.Version, items,
	)
}
