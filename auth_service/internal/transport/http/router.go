package http

import (
	nethttp "net/http"

	"auth_service/internal/transport/http/docs"
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

	auth := middleware.Auth(parser)
	for _, prefix := range []string{"/api/auth", "/api/v1/auth"} {
		mux.HandleFunc("POST "+prefix+"/register", h.Register)
		mux.HandleFunc("POST "+prefix+"/login", h.Login)
		mux.Handle("GET "+prefix+"/me", auth(nethttp.HandlerFunc(h.Me)))
	}

	docs.Register(mux)

	return mux
}
