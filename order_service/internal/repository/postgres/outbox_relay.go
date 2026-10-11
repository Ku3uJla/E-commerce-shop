package repository

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order_service/internal/messaging"
)

// OutboxRelay publishes outbox rows to Kafka in insertion order. Rows are
// locked with SKIP LOCKED, so several service replicas can run it safely.
// A crash between publishing and marking a row produces a duplicate, which
// consumers tolerate because they are idempotent.
type OutboxRelay struct {
	db        *gorm.DB
	publisher *messaging.Publisher
	interval  time.Duration
	batchSize int
}

func NewOutboxRelay(db *gorm.DB, publisher *messaging.Publisher) *OutboxRelay {
	return &OutboxRelay{db: db, publisher: publisher, interval: 300 * time.Millisecond, batchSize: 100}
}

// Run polls the outbox until ctx is done.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		for {
			n, err := r.publishBatch(ctx)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("outbox relay: %v", err)
				}
				break
			}
			if n < r.batchSize {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelay) publishBatch(ctx context.Context) (int, error) {
	published := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []OutboxModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("published_at IS NULL").
			Order("seq").
			Limit(r.batchSize).
			Find(&rows).Error
		if err != nil || len(rows) == 0 {
			return err
		}

		msgs := make([]messaging.Message, 0, len(rows))
		seqs := make([]int64, 0, len(rows))
		for _, row := range rows {
			var env messaging.Envelope
			if err := json.Unmarshal(row.Envelope, &env); err != nil {
				return err
			}
			msgs = append(msgs, messaging.Message{Topic: row.Topic, Key: row.Key, Envelope: env})
			seqs = append(seqs, row.Seq)
		}
		if err := r.publisher.Publish(ctx, msgs...); err != nil {
			return err
		}
		published = len(rows)
		return tx.Model(&OutboxModel{}).Where("seq IN ?", seqs).Update("published_at", time.Now().UTC()).Error
	})
	return published, err
}
