package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"payment_service/internal/domain/payment"
)

// PaymentModel is the GORM persistence model for the payments table.
// The unique index on order_id guarantees one payment per order.
type PaymentModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Amount    int64     `gorm:"not null"`
	Status    string    `gorm:"size:20;not null"`
	Reason    string    `gorm:"size:255"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (PaymentModel) TableName() string { return "payments" }

// Connect opens a PostgreSQL connection with the given DSN.
func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// EnsureDatabase creates the database if it does not exist yet.
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

// AutoMigrate creates or updates the payments schema.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PaymentModel{})
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) *paymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	err := r.db.WithContext(ctx).Create(toModel(p)).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return payment.ErrAlreadyExists
	}
	return err
}

func (r *paymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	return r.db.WithContext(ctx).Model(&PaymentModel{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{"status": string(p.Status), "reason": p.Reason, "updated_at": p.UpdatedAt}).Error
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, payment.ErrPaymentNotFound
	}
	var m PaymentModel
	err := r.db.WithContext(ctx).First(&m, "order_id = ?", orderID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payment.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &payment.Payment{
		ID:        m.ID.String(),
		OrderID:   m.OrderID.String(),
		UserID:    m.UserID.String(),
		Amount:    m.Amount,
		Status:    payment.Status(m.Status),
		Reason:    m.Reason,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func toModel(p *payment.Payment) *PaymentModel {
	return &PaymentModel{
		ID:        uuid.MustParse(p.ID),
		OrderID:   uuid.MustParse(p.OrderID),
		UserID:    uuid.MustParse(p.UserID),
		Amount:    p.Amount,
		Status:    string(p.Status),
		Reason:    p.Reason,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
