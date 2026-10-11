package http

import (
	nethttp "net/http"

	"catalog_service/internal/transport/http/docs"
	"catalog_service/internal/transport/http/handler"
)

// NewRouter wires the catalog service HTTP routes. Every response carries the
// instance id so that load balancing across replicas is observable.
func NewRouter(h *handler.Handler, instanceID string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Health)

	for _, prefix := range []string{"/api/catalog/products", "/api/v1/products"} {
		mux.HandleFunc("POST "+prefix, h.Create)
		mux.HandleFunc("GET "+prefix, h.List)
		mux.HandleFunc("GET "+prefix+"/{id}", h.Get)
		mux.HandleFunc("PUT "+prefix+"/{id}", h.Update)
		mux.HandleFunc("DELETE "+prefix+"/{id}", h.Delete)
	}

	docs.Register(mux)

	return withInstanceHeader(instanceID, mux)
}

func withInstanceHeader(instanceID string, next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Instance-Id", instanceID)
		next.ServeHTTP(w, r)
	})
}
