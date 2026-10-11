package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"catalog_service/internal/domain/product"
	"catalog_service/internal/transport/http/dto"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Handler holds the HTTP handlers for the catalog service.
type Handler struct {
	svc        product.Service
	instanceID string
}

// New builds a Handler around the given product service.
func New(svc product.Service, instanceID string) *Handler {
	return &Handler{svc: svc, instanceID: instanceID}
}

// Health reports that the instance is up. Consul uses it as a health check.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "instance": h.instanceID})
}

// Create handles POST /api/v1/products.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, description, price, stock, err := parseInput(req.Name, req.Description, req.Price, req.Stock)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.svc.Create(r.Context(), name, description, price, stock)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create product")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(p))
}

// List handles GET /api/catalog/products?page=&limit= (or ?limit=&offset=).
// With ?ids=a,b,c it returns exactly those products in one batch, which the
// GraphQL gateway uses to avoid N+1 requests.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if rawIDs := q.Get("ids"); rawIDs != "" {
		h.listByIDs(w, r, rawIDs)
		return
	}

	limit := parseInt(q.Get("limit"), defaultPageSize)
	if limit <= 0 || limit > maxPageSize {
		limit = defaultPageSize
	}
	offset := parseInt(q.Get("offset"), 0)
	if page := parseInt(q.Get("page"), 0); page > 0 {
		offset = (page - 1) * limit
	}
	if offset < 0 {
		offset = 0
	}

	products, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list products")
		return
	}
	total, err := h.svc.Count(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list products")
		return
	}
	writeJSON(w, http.StatusOK, dto.ProductListResponse{
		Items:  toResponses(products),
		Count:  len(products),
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Page:   offset/limit + 1,
	})
}

func (h *Handler) listByIDs(w http.ResponseWriter, r *http.Request, rawIDs string) {
	parts := strings.Split(rawIDs, ",")
	if len(parts) > maxPageSize {
		writeError(w, http.StatusBadRequest, "too many ids")
		return
	}
	ids := make([]product.ProductID, 0, len(parts))
	for _, raw := range parts {
		id, err := product.NewProductID(strings.TrimSpace(raw))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product id")
			return
		}
		ids = append(ids, id)
	}
	products, err := h.svc.GetMany(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list products")
		return
	}
	writeJSON(w, http.StatusOK, dto.ProductListResponse{
		Items: toResponses(products),
		Count: len(products),
		Total: int64(len(products)),
		Limit: len(ids),
		Page:  1,
	})
}

func toResponses(products []*product.Product) []dto.ProductResponse {
	items := make([]dto.ProductResponse, 0, len(products))
	for _, p := range products {
		items = append(items, toResponse(p))
	}
	return items
}

// Get handles GET /api/v1/products/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(p))
}

// Update handles PUT /api/v1/products/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, description, price, stock, err := parseInput(req.Name, req.Description, req.Price, req.Stock)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.svc.Update(r.Context(), id, name, description, price, stock)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(p))
}

// Delete handles DELETE /api/v1/products/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (product.ProductID, bool) {
	id, err := product.NewProductID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return product.ProductID{}, false
	}
	return id, true
}

func parseInput(rawName, rawDescription string, rawPrice int64, rawStock int) (product.Name, product.Description, product.Price, product.Stock, error) {
	name, err := product.NewName(rawName)
	if err != nil {
		return product.Name{}, product.Description{}, product.Price{}, product.Stock{}, err
	}
	description, err := product.NewDescription(rawDescription)
	if err != nil {
		return product.Name{}, product.Description{}, product.Price{}, product.Stock{}, err
	}
	price, err := product.NewPrice(rawPrice)
	if err != nil {
		return product.Name{}, product.Description{}, product.Price{}, product.Stock{}, err
	}
	stock, err := product.NewStock(rawStock)
	if err != nil {
		return product.Name{}, product.Description{}, product.Price{}, product.Stock{}, err
	}
	return name, description, price, stock, nil
}

func toResponse(p *product.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID:          p.ID().String(),
		Name:        p.Name().String(),
		Description: p.Description().String(),
		Price:       p.Price().Int64(),
		Stock:       p.Stock().Int(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, product.ErrProductNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
