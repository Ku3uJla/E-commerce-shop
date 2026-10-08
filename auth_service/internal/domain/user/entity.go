// internal/domain/user/user.go
package user

import (
	"time"
)

type User struct {
	id        UserID
	name      Name
	email     Email
	password  HashedPassword
	createdAt time.Time
	updatedAt *time.Time

	events []DomainEvent
}

func Register(name Name, email Email, password HashedPassword) *User {
	now := time.Now().UTC()
	u := &User{
		id:        GenerateUserID(),
		name:      name,
		email:     email,
		password:  password,
		createdAt: now,
	}
	u.recordEvent(UserRegistered{
		UserID:     u.id.String(),
		Email:      u.email.String(),
		OccurredAt: now,
	})
	return u
}

func Reconstitute(
	id UserID,
	name Name,
	email Email,
	password HashedPassword,
	createdAt time.Time,
	updatedAt *time.Time,
) *User {
	return &User{
		id:        id,
		name:      name,
		email:     email,
		password:  password,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (u *User) ID() UserID               { return u.id }
func (u *User) Name() Name               { return u.name }
func (u *User) Email() Email             { return u.email }
func (u *User) Password() HashedPassword { return u.password }
func (u *User) CreatedAt() time.Time     { return u.createdAt }
func (u *User) UpdatedAt() *time.Time    { return u.updatedAt }

func (u *User) Events() []DomainEvent { return u.events }
func (u *User) ClearEvents()          { u.events = nil }

func (u *User) recordEvent(e DomainEvent) {
	u.events = append(u.events, e)
}
