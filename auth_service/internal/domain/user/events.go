// internal/domain/user/event.go
package user

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAtTime() time.Time
}

type UserRegistered struct {
	UserID     string
	Email      string
	OccurredAt time.Time
}

func (e UserRegistered) EventName() string         { return "user.registered" }
func (e UserRegistered) OccurredAtTime() time.Time { return e.OccurredAt }
