package http

import (
	nethttp "net/http"

	"auth_service/internal/transport/http/handler"
	"auth_service/internal/transport/http/middleware"
)

// NewRouter wires the auth service HTTP routes.
func NewRouter(h *handler.Handler, parser middleware.TokenParser) *nethttp.ServeMux {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)

	auth := middleware.Auth(parser)
	mux.Handle("GET /api/v1/auth/me", auth(nethttp.HandlerFunc(h.Me)))

	return mux
}
