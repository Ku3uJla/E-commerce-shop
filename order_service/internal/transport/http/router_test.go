package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"order_service/internal/domain/order"
	"order_service/internal/transport/http/dto"
	"order_service/internal/transport/http/handler"
)

type memRepo struct {
	mu    sync.Mutex
	items map[string]*order.Order
}

func (r *memRepo) Create(_ context.Context, o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[o.ID()] = o
	return nil
}

func (r *memRepo) GetByID(_ context.Context, id string) (*order.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.items[id]; ok {
		return o, nil
	}
	return nil, order.ErrOrderNotFound
}

func (r *memRepo) ListByUser(_ context.Context, userID string) ([]*order.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*order.Order{}
	for _, o := range r.items {
		if o.UserID() == userID {
			out = append(out, o)
		}
	}
	return out, nil
}

func newTestServer() *httptest.Server {
	svc := order.NewService(&memRepo{items: map[string]*order.Order{}})
	return httptest.NewServer(NewRouter(handler.New(svc, "test"), "test"))
}

func TestCreateGetAndList(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	userID, productID := uuid.NewString(), uuid.NewString()
	body := `{"userId":"` + userID + `","items":[{"productId":"` + productID + `","quantity":2}]}`
	resp, err := nethttp.Post(srv.URL+"/api/orders", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("POST /api/orders status = %d, want 201", resp.StatusCode)
	}
	var created dto.OrderResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	resp, err = nethttp.Get(srv.URL + "/api/orders/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("GET /api/orders/{id} status = %d, want 200", resp.StatusCode)
	}

	resp, err = nethttp.Get(srv.URL + "/api/orders?userId=" + userID)
	if err != nil {
		t.Fatal(err)
	}
	var list dto.OrderListResponse
	_ = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if list.Count != 1 || list.Items[0].Items[0].ProductID != productID {
		t.Fatalf("GET /api/orders?userId= returned %+v", list)
	}
}

func TestBadRequests(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"missing userId", "GET", "/api/orders", "", 400},
		{"bad order id", "GET", "/api/orders/nope", "", 400},
		{"unknown order", "GET", "/api/orders/" + uuid.NewString(), "", 404},
		{"empty items", "POST", "/api/orders", `{"userId":"` + uuid.NewString() + `","items":[]}`, 400},
		{"zero quantity", "POST", "/api/orders", `{"userId":"` + uuid.NewString() + `","items":[{"productId":"` + uuid.NewString() + `","quantity":0}]}`, 400},
	}
	for _, tc := range cases {
		req, _ := nethttp.NewRequest(tc.method, srv.URL+tc.path, strings.NewReader(tc.body))
		resp, err := nethttp.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, resp.StatusCode, tc.want)
		}
	}
}
