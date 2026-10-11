// Package messaging is a thin layer over Kafka (segmentio/kafka-go): a JSON
// event envelope, a publisher and an at-least-once consumer loop.
package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Envelope is the wire format of every event.
type Envelope struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	OccurredAt time.Time       `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
}

// NewEnvelope wraps a payload into a new envelope.
func NewEnvelope(source, eventType string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		ID:         uuid.NewString(),
		Type:       eventType,
		Source:     source,
		OccurredAt: time.Now().UTC(),
		Payload:    raw,
	}, nil
}

// Message is an envelope addressed to a topic. Key decides the partition, so
// all events with the same key (order id) are consumed in order.
type Message struct {
	Topic    string
	Key      string
	Envelope Envelope
}

// Publisher writes events to Kafka.
type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string) *Publisher {
	return &Publisher{writer: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
	}}
}

// Publish writes messages synchronously and returns once Kafka acknowledged them.
func (p *Publisher) Publish(ctx context.Context, msgs ...Message) error {
	out := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		value, err := json.Marshal(m.Envelope)
		if err != nil {
			return err
		}
		out = append(out, kafka.Message{
			Topic: m.Topic,
			Key:   []byte(m.Key),
			Value: value,
			Headers: []kafka.Header{
				{Key: "event-type", Value: []byte(m.Envelope.Type)},
			},
		})
	}
	return p.writer.WriteMessages(ctx, out...)
}

func (p *Publisher) Close() error { return p.writer.Close() }

// Handler processes one event. Returning an error makes the consumer retry.
type Handler func(ctx context.Context, env Envelope) error

// Consume reads topic as part of consumer group groupID until ctx is done.
// The offset is committed only after the handler succeeded (at-least-once),
// so handlers must be idempotent. A message that keeps failing is logged and
// skipped after maxAttempts so that it does not block the partition.
func Consume(ctx context.Context, brokers []string, groupID, topic string, handle Handler) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		Topic:       topic,
		StartOffset: kafka.FirstOffset,
		MaxWait:     500 * time.Millisecond,
	})
	defer reader.Close()

	const maxAttempts = 5
	log.Printf("consumer %s subscribed to %s", groupID, topic)
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("consumer %s: fetch from %s failed: %v", groupID, topic, err)
			sleep(ctx, time.Second)
			continue
		}

		var env Envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Printf("consumer %s: malformed message at %s/%d/%d skipped: %v", groupID, topic, msg.Partition, msg.Offset, err)
		} else {
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				err = handle(ctx, env)
				if err == nil || ctx.Err() != nil {
					break
				}
				log.Printf("consumer %s: %s %s attempt %d/%d failed: %v", groupID, env.Type, env.ID, attempt, maxAttempts, err)
				sleep(ctx, time.Duration(attempt)*500*time.Millisecond)
			}
			if ctx.Err() != nil {
				return // not committed: the message is redelivered after restart
			}
			if err != nil {
				log.Printf("consumer %s: giving up on %s %s", groupID, env.Type, env.ID)
			}
		}

		if err := reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
			log.Printf("consumer %s: commit failed: %v", groupID, err)
		}
	}
}

// EnsureTopics creates the topics if they do not exist yet. It retries while
// the broker is starting up.
func EnsureTopics(ctx context.Context, brokers []string, partitions int, topics ...string) error {
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		if lastErr = createTopics(brokers[0], partitions, topics); lastErr == nil {
			return nil
		}
		log.Printf("kafka not ready (%v), retrying...", lastErr)
		if !sleep(ctx, 2*time.Second) {
			return ctx.Err()
		}
	}
	return fmt.Errorf("ensure topics: %w", lastErr)
}

func createTopics(broker string, partitions int, topics []string) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	ctrl, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer ctrl.Close()

	configs := make([]kafka.TopicConfig, 0, len(topics))
	for _, t := range topics {
		configs = append(configs, kafka.TopicConfig{Topic: t, NumPartitions: partitions, ReplicationFactor: 1})
	}
	err = ctrl.CreateTopics(configs...)
	if errors.Is(err, kafka.TopicAlreadyExists) {
		return nil
	}
	return err
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
