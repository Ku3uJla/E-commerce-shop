package http

import (
	"encoding/json"
	nethttp "net/http"

	"notification_service/internal/notification"
	"notification_service/internal/transport/http/docs"
)

// NewRouter wires the notification service HTTP routes.
func NewRouter(svc *notification.Service, instanceID string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok", "instance": instanceID})
	})

	mux.HandleFunc("GET /api/notifications", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		userID := r.URL.Query().Get("userId")
		if userID == "" {
			writeJSON(w, nethttp.StatusBadRequest, map[string]string{"error": "userId query parameter is required"})
			return
		}
		items := svc.ListByUser(userID)
		writeJSON(w, nethttp.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	docs.Register(mux)

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Instance-Id", instanceID)
		mux.ServeHTTP(w, r)
	})
}

func writeJSON(w nethttp.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
