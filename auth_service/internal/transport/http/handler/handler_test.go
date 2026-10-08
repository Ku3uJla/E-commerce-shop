package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"auth_service/internal/domain/user"
	"auth_service/internal/transport/http/middleware"
)

type fakeService struct {
	user *user.User
}

func (s *fakeService) Register(_ context.Context, name user.Name, email user.Email, password string) error {
	if email.String() == "taken@example.com" {
		return user.ErrEmailAlreadyExists
	}
	if len(password) < 8 {
		return user.ErrWeakPassword
	}
	hashed, _ := user.NewHashedPassword("hashed:" + password)
	s.user = user.Register(name, email, hashed)
	return nil
}

func (s *fakeService) Login(_ context.Context, email user.Email, password string) (string, string, error) {
	if email.String() != "john@example.com" || password != "password123" {
		return "", "", user.ErrInvalidCredentials
	}
	return "user-id", "token", nil
}

func (s *fakeService) Profile(_ context.Context, uid user.UserID) (*user.User, error) {
	if s.user == nil {
		return nil, user.ErrUserNotFound
	}
	return s.user, nil
}

type fakeParser struct{ id string }

func (p fakeParser) ParseToken(string) (string, error) { return p.id, nil }

func TestRegisterHandlerCreated(t *testing.T) {
	h := New(&fakeService{})
	body := `{"name":"John","email":"john@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestRegisterHandlerConflict(t *testing.T) {
	h := New(&fakeService{})
	body := `{"name":"John","email":"taken@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestLoginHandlerOK(t *testing.T) {
	h := New(&fakeService{})
	body := `{"email":"john@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token != "token" {
		t.Fatalf("token = %q, want %q", resp.Token, "token")
	}
}

func TestLoginHandlerUnauthorized(t *testing.T) {
	h := New(&fakeService{})
	body := `{"email":"john@example.com","password":"nope"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginHandlerBadBody(t *testing.T) {
	h := New(&fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMeHandlerRequiresToken(t *testing.T) {
	h := New(&fakeService{})
	protected := middleware.Auth(fakeParser{id: "user-id"})(http.HandlerFunc(h.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status without token = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMeHandlerWithToken(t *testing.T) {
	email, _ := user.NewEmail("john@example.com")
	profile := user.Register(mustName(t), email, mustHashed(t))
	h := New(&fakeService{user: profile})
	protected := middleware.Auth(fakeParser{id: profile.ID().String()})(http.HandlerFunc(h.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status with token = %d, want %d", rec.Code, http.StatusOK)
	}
}

func mustName(t *testing.T) user.Name {
	t.Helper()
	name, err := user.NewName("John")
	if err != nil {
		t.Fatalf("NewName error: %v", err)
	}
	return name
}

func mustHashed(t *testing.T) user.HashedPassword {
	t.Helper()
	hashed, err := user.NewHashedPassword("hashed:password123")
	if err != nil {
		t.Fatalf("NewHashedPassword error: %v", err)
	}
	return hashed
}
