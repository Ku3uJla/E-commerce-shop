package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth_service/internal/domain/user"
	"auth_service/internal/features"
	"auth_service/internal/transport/http/handler"
)

// memRepo is an in-memory user.UserRepository used for end-to-end tests.
type memRepo struct {
	users map[string]*user.User
}

func newMemRepo() *memRepo {
	return &memRepo{users: map[string]*user.User{}}
}

func (r *memRepo) Create(_ context.Context, u *user.User) error {
	if _, ok := r.users[u.Email().String()]; ok {
		return errors.New("duplicate email")
	}
	r.users[u.Email().String()] = u
	return nil
}

func (r *memRepo) GetByEmail(_ context.Context, email user.Email) (*user.User, error) {
	if u, ok := r.users[email.String()]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}

func (r *memRepo) GetByUID(_ context.Context, uid user.UserID) (*user.User, error) {
	for _, u := range r.users {
		if u.ID().String() == uid.String() {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func TestAuthFlowEndToEnd(t *testing.T) {
	tokens := features.NewJWTService("test-secret", time.Hour)
	userService := user.NewUserSrv(newMemRepo(), features.NewBcryptHasher(), tokens)
	server := httptest.NewServer(NewRouter(handler.New(userService), tokens))
	defer server.Close()

	// 1. Register.
	registerBody := `{"name":"John","email":"john@example.com","password":"password123"}`
	resp, err := http.Post(server.URL+"/api/v1/auth/register", "application/json", strings.NewReader(registerBody))
	if err != nil {
		t.Fatalf("register request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	// 2. Login and grab the token.
	loginBody := `{"email":"john@example.com","password":"password123"}`
	resp, err = http.Post(server.URL+"/api/v1/auth/login", "application/json", strings.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var auth struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if auth.Token == "" {
		t.Fatal("login response did not include a token")
	}

	// 3. Call the protected endpoint with the token.
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("build me request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var profile struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if profile.Email != "john@example.com" {
		t.Fatalf("me email = %q, want %q", profile.Email, "john@example.com")
	}
	if profile.ID != auth.ID {
		t.Fatalf("me id = %q, want %q", profile.ID, auth.ID)
	}
}
