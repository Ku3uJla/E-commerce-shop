package http

import (
	nethttp "net/http"

	"catalog_service/internal/transport/http/handler"
)

// NewRouter wires the catalog service HTTP routes. Every response carries the
// instance id so that load balancing across replicas is observable.
func NewRouter(h *handler.Handler, instanceID string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Health)

	mux.HandleFunc("POST /api/v1/products", h.Create)
	mux.HandleFunc("GET /api/v1/products", h.List)
	mux.HandleFunc("GET /api/v1/products/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/products/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/products/{id}", h.Delete)

	return withInstanceHeader(instanceID, mux)
}

func withInstanceHeader(instanceID string, next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Instance-Id", instanceID)
		next.ServeHTTP(w, r)
	})
}
