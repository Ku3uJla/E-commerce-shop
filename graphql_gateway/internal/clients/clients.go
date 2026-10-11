// Package clients holds tiny REST clients for the upstream services.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CatalogBatchSize is the maximum number of ids Catalog accepts per request.
const CatalogBatchSize = 100

// UpstreamError is returned when an upstream service fails or rejects a call.
type UpstreamError struct {
	Status  int // 0 when the service is unreachable
	Message string
}

func (e *UpstreamError) Error() string { return e.Message }

// OrderItem mirrors an order line returned by the Order service.
type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// Order mirrors an order returned by the Order service.
type Order struct {
	ID           string      `json:"id"`
	UserID       string      `json:"userId"`
	CreatedAt    time.Time   `json:"createdAt"`
	Status       string      `json:"status"`
	Total        int64       `json:"total"` // cents
	CancelReason string      `json:"cancelReason"`
	Items        []OrderItem `json:"items"`
}

// Product mirrors a product returned by the Catalog service.
type Product struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"` // cents
}

type listResponse[T any] struct {
	Items []T `json:"items"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func getJSON(ctx context.Context, client *http.Client, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &UpstreamError{Message: "upstream unavailable: " + req.URL.Host}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body errorResponse
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.Error == "" {
			body.Error = fmt.Sprintf("upstream responded %d", resp.StatusCode)
		}
		return &UpstreamError{Status: resp.StatusCode, Message: body.Error}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// OrderClient talks to the Order service REST API.
type OrderClient struct {
	baseURL string
	http    *http.Client
}

func NewOrderClient(baseURL string) *OrderClient {
	return &OrderClient{baseURL: strings.TrimRight(baseURL, "/"), http: newHTTPClient()}
}

// ListByUser calls GET /api/orders?userId=.
func (c *OrderClient) ListByUser(ctx context.Context, userID string) ([]Order, error) {
	var body listResponse[Order]
	err := getJSON(ctx, c.http, c.baseURL+"/api/orders?userId="+url.QueryEscape(userID), &body)
	return body.Items, err
}

// CatalogClient talks to the Catalog service REST API.
type CatalogClient struct {
	baseURL string
	http    *http.Client
}

func NewCatalogClient(baseURL string) *CatalogClient {
	return &CatalogClient{baseURL: strings.TrimRight(baseURL, "/"), http: newHTTPClient()}
}

// GetMany calls GET /api/catalog/products?ids=a,b,c, one request per
// CatalogBatchSize ids. Unknown ids are simply absent from the result.
func (c *CatalogClient) GetMany(ctx context.Context, ids []string) ([]Product, error) {
	products := make([]Product, 0, len(ids))
	for start := 0; start < len(ids); start += CatalogBatchSize {
		end := min(start+CatalogBatchSize, len(ids))
		var body listResponse[Product]
		u := c.baseURL + "/api/catalog/products?ids=" + url.QueryEscape(strings.Join(ids[start:end], ","))
		if err := getJSON(ctx, c.http, u, &body); err != nil {
			return nil, err
		}
		products = append(products, body.Items...)
	}
	return products, nil
}
