package user

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRepo struct {
	byEmail map[string]*User
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byEmail: map[string]*User{}}
}

func (r *fakeRepo) Create(_ context.Context, u *User) error {
	r.byEmail[u.Email().String()] = u
	return nil
}

func (r *fakeRepo) GetByEmail(_ context.Context, email Email) (*User, error) {
	if u, ok := r.byEmail[email.String()]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func (r *fakeRepo) GetByUID(_ context.Context, uid UserID) (*User, error) {
	for _, u := range r.byEmail {
		if u.ID().String() == uid.String() {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

type fakeHasher struct{}

func (fakeHasher) HashPassword(password string) (string, error) { return "hashed:" + password, nil }
func (fakeHasher) ComparePassword(password, hashed string) bool {
	return hashed == "hashed:"+password
}

type fakeJWT struct{}

func (fakeJWT) CreateToken(id string) (string, error) { return "token:" + id, nil }

func newTestService() (*userSrv, *fakeRepo) {
	repo := newFakeRepo()
	return NewUserSrv(repo, fakeHasher{}, fakeJWT{}), repo
}

func mustName(t *testing.T, value string) Name {
	t.Helper()
	name, err := NewName(value)
	if err != nil {
		t.Fatalf("NewName(%q) error: %v", value, err)
	}
	return name
}

func mustEmail(t *testing.T, value string) Email {
	t.Helper()
	email, err := NewEmail(value)
	if err != nil {
		t.Fatalf("NewEmail(%q) error: %v", value, err)
	}
	return email
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestService()
	email := mustEmail(t, "John@Example.Com")

	if err := srv.Register(ctx, mustName(t, "John"), email, "password123"); err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	id, token, err := srv.Login(ctx, email, "password123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if id == "" {
		t.Fatal("Login() returned empty id")
	}
	if token != "token:"+id {
		t.Fatalf("Login() token = %q, want %q", token, "token:"+id)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestService()
	email := mustEmail(t, "dup@example.com")

	if err := srv.Register(ctx, mustName(t, "First"), email, "password123"); err != nil {
		t.Fatalf("first Register() error: %v", err)
	}
	err := srv.Register(ctx, mustName(t, "Second"), email, "password123")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("second Register() error = %v, want %v", err, ErrEmailAlreadyExists)
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestService()

	err := srv.Register(ctx, mustName(t, "John"), mustEmail(t, "john@example.com"), "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("Register() error = %v, want %v", err, ErrWeakPassword)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestService()
	email := mustEmail(t, "jane@example.com")

	if err := srv.Register(ctx, mustName(t, "Jane"), email, "password123"); err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	_, _, err := srv.Login(ctx, email, "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestService()

	_, _, err := srv.Login(ctx, mustEmail(t, "missing@example.com"), "password123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestRegistrationStoresHashedPassword(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestService()
	email := mustEmail(t, "hash@example.com")

	if err := srv.Register(ctx, mustName(t, "Hash"), email, "password123"); err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	stored := repo.byEmail[email.String()]
	if stored.Password().String() != "hashed:password123" {
		t.Fatalf("stored password = %q, want the hasher output", stored.Password().String())
	}
	if strings.HasPrefix(stored.Password().String(), "password123") {
		t.Fatal("stored password must not be the plaintext password")
	}
}
