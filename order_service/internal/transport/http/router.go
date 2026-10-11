package http

import (
	nethttp "net/http"

	"order_service/internal/transport/http/docs"
	"order_service/internal/transport/http/handler"
)

// NewRouter wires the order service HTTP routes.
func NewRouter(h *handler.Handler, instanceID string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/orders", h.Create)
	mux.HandleFunc("GET /api/orders", h.List)
	mux.HandleFunc("GET /api/orders/{id}", h.Get)

	docs.Register(mux)

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Instance-Id", instanceID)
		mux.ServeHTTP(w, r)
	})
}
