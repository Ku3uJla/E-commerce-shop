// Package catalog fetches product prices from the Catalog service.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUnavailable is returned when the Catalog service cannot be reached.
var ErrUnavailable = errors.New("catalog service unavailable")

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 5 * time.Second}}
}

// Prices returns price (cents) by product id, using one batch request.
// Unknown products are absent from the result.
func (c *Client) Prices(ctx context.Context, productIDs []string) (map[string]int64, error) {
	u := c.baseURL + "/api/catalog/products?ids=" + url.QueryEscape(strings.Join(productIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}

	var body struct {
		Items []struct {
			ID    string `json:"id"`
			Price int64  `json:"price"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	prices := make(map[string]int64, len(body.Items))
	for _, p := range body.Items {
		prices[p.ID] = p.Price
	}
	return prices, nil
}
