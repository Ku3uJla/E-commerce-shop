package user

import (
	"context"
	"errors"
)

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrWeakPassword       = errors.New("password must be at least 8 characters long")
)

const minPasswordLength = 8

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByUID(ctx context.Context, uid UserID) (*User, error)
}

type Hash interface {
	HashPassword(password string) (string, error)
	ComparePassword(password string, hashedPassword string) bool
}

type JWT interface {
	CreateToken(id string) (string, error)
}

// Service exposes the user use-cases to the transport layer.
type Service interface {
	Register(ctx context.Context, name Name, email Email, password string) error
	Login(ctx context.Context, email Email, password string) (string, string, error)
	Profile(ctx context.Context, uid UserID) (*User, error)
}

type userSrv struct {
	repo UserRepository
	hash Hash
	jwt  JWT
}

func NewUserSrv(userRepo UserRepository, hash Hash, jwt JWT) *userSrv {
	return &userSrv{repo: userRepo, hash: hash, jwt: jwt}
}

// Register creates a new user account.
func (srv *userSrv) Register(ctx context.Context, name Name, email Email, password string) error {
	if len(password) < minPasswordLength {
		return ErrWeakPassword
	}

	if _, err := srv.repo.GetByEmail(ctx, email); err == nil {
		return ErrEmailAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return err
	}

	hashed, err := srv.hash.HashPassword(password)
	if err != nil {
		return err
	}
	hashedPassword, err := NewHashedPassword(hashed)
	if err != nil {
		return err
	}

	return srv.repo.Create(ctx, Register(name, email, hashedPassword))
}

// Login verifies credentials and returns the user id together with a JWT.
func (srv *userSrv) Login(ctx context.Context, email Email, password string) (string, string, error) {
	u, err := srv.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}
	if !srv.hash.ComparePassword(password, u.Password().String()) {
		return "", "", ErrInvalidCredentials
	}
	token, err := srv.jwt.CreateToken(u.ID().String())
	if err != nil {
		return "", "", err
	}
	return u.ID().String(), token, nil
}

// Profile returns the user with the given id.
func (srv *userSrv) Profile(ctx context.Context, uid UserID) (*User, error) {
	return srv.repo.GetByUID(ctx, uid)
}
