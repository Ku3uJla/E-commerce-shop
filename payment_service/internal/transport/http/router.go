package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"time"

	"payment_service/internal/domain/payment"
	"payment_service/internal/transport/http/docs"
)

type paymentResponse struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"orderId"`
	UserID    string    `json:"userId"`
	Amount    int64     `json:"amount"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewRouter wires the payment service HTTP routes. Payments are created only
// by the saga (Kafka); HTTP is read-only.
func NewRouter(svc payment.Service, instanceID string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok", "instance": instanceID})
	})

	mux.HandleFunc("GET /api/payments", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		orderID := r.URL.Query().Get("orderId")
		if orderID == "" {
			writeJSON(w, nethttp.StatusBadRequest, map[string]string{"error": "orderId query parameter is required"})
			return
		}
		p, err := svc.GetByOrderID(r.Context(), orderID)
		if errors.Is(err, payment.ErrPaymentNotFound) {
			writeJSON(w, nethttp.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, nethttp.StatusOK, paymentResponse{
			ID: p.ID, OrderID: p.OrderID, UserID: p.UserID, Amount: p.Amount,
			Status: string(p.Status), Reason: p.Reason, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		})
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
