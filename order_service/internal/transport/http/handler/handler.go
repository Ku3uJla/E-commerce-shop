package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"order_service/internal/domain/order"
	"order_service/internal/transport/http/dto"
)

// Handler holds the HTTP handlers for the order service.
type Handler struct {
	svc        order.Service
	instanceID string
}

// New builds a Handler around the given order service.
func New(svc order.Service, instanceID string) *Handler {
	return &Handler{svc: svc, instanceID: instanceID}
}

// Health reports that the instance is up. Consul uses it as a health check.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "instance": h.instanceID})
}

// Create handles POST /api/orders.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items := make([]order.Item, 0, len(req.Items))
	for _, it := range req.Items {
		item, err := order.NewItem(it.ProductID, it.Quantity)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		items = append(items, item)
	}

	o, err := h.svc.Place(r.Context(), req.UserID, items)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(o))
}

// List handles GET /api/orders?userId=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "userId query parameter is required")
		return
	}
	orders, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]dto.OrderResponse, 0, len(orders))
	for _, o := range orders {
		items = append(items, toResponse(o))
	}
	writeJSON(w, http.StatusOK, dto.OrderListResponse{Items: items, Count: len(items)})
}

// Get handles GET /api/orders/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(o))
}

func toResponse(o *order.Order) dto.OrderResponse {
	items := make([]dto.ItemResponse, 0, len(o.Items()))
	for _, it := range o.Items() {
		items = append(items, dto.ItemResponse{ProductID: it.ProductID(), Quantity: it.Quantity()})
	}
	return dto.OrderResponse{
		ID:        o.ID(),
		UserID:    o.UserID(),
		CreatedAt: o.CreatedAt(),
		Items:     items,
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, order.ErrInvalidID),
		errors.Is(err, order.ErrEmptyOrder),
		errors.Is(err, order.ErrTooManyItems),
		errors.Is(err, order.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
