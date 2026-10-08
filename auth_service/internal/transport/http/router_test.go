package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth_service/internal/domain/user"
	"auth_service/internal/features"
	"auth_service/internal/transport/http/handler"
)

type routerFakeService struct{}

func (routerFakeService) Register(context.Context, user.Name, user.Email, string) error {
	return nil
}

func (routerFakeService) Login(context.Context, user.Email, string) (string, string, error) {
	return "", "", user.ErrInvalidCredentials
}

func (routerFakeService) Profile(context.Context, user.UserID) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func TestRouterRoutes(t *testing.T) {
	tokens := features.NewJWTService("test-secret", time.Hour)
	router := NewRouter(handler.New(routerFakeService{}), tokens)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		header string
		want   int
	}{
		{"health", http.MethodGet, "/healthz", "", "", http.StatusOK},
		{"register", http.MethodPost, "/api/v1/auth/register", `{"name":"John","email":"john@example.com","password":"password123"}`, "", http.StatusCreated},
		{"login bad body", http.MethodPost, "/api/v1/auth/login", "{", "", http.StatusBadRequest},
		{"me without token", http.MethodGet, "/api/v1/auth/me", "", "", http.StatusUnauthorized},
		{"wrong method", http.MethodGet, "/api/v1/auth/login", "", "", http.StatusMethodNotAllowed},
		{"not found", http.MethodGet, "/does-not-exist", "", "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *strings.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
