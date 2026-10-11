package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"graphql_gateway/internal/clients"
	"graphql_gateway/internal/graph"
)

const (
	p1   = "11111111-1111-1111-1111-111111111111"
	p2   = "22222222-2222-2222-2222-222222222222"
	user = "99999999-9999-9999-9999-999999999999"
)

// fakeUpstream plays both the Order and the Catalog service.
func fakeUpstream(catalogCalls *atomic.Int32, lastIDs *atomic.Value) *httptest.Server {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET /api/orders", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		_, _ = w.Write([]byte(`{"items":[
			{"id":"o1","userId":"` + user + `","createdAt":"2026-01-01T00:00:00Z","items":[{"productId":"` + p1 + `","quantity":1},{"productId":"` + p2 + `","quantity":2}]},
			{"id":"o2","userId":"` + user + `","createdAt":"2026-01-02T00:00:00Z","items":[{"productId":"` + p1 + `","quantity":5}]}
		]}`))
	})
	mux.HandleFunc("GET /api/catalog/products", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		catalogCalls.Add(1)
		ids := strings.Split(r.URL.Query().Get("ids"), ",")
		lastIDs.Store(ids)
		all := map[string]clients.Product{
			p1: {ID: p1, Name: "Keyboard", Price: 4999},
			p2: {ID: p2, Name: "Mouse", Price: 1500},
		}
		items := []clients.Product{}
		for _, id := range ids {
			if p, ok := all[id]; ok {
				items = append(items, p)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	return httptest.NewServer(mux)
}

type gqlResponse struct {
	Data struct {
		OrdersByUserID []struct {
			ID        string `json:"id"`
			CreatedAt string `json:"createdAt"`
			Items     []struct {
				Quantity int `json:"quantity"`
				Product  struct {
					ID    string  `json:"id"`
					Name  string  `json:"name"`
					Price float64 `json:"price"`
				} `json:"product"`
			} `json:"items"`
		} `json:"ordersByUserId"`
		Type *struct {
			Description string `json:"description"`
		} `json:"__type"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func newGateway(t *testing.T, dev bool) (*httptest.Server, *atomic.Int32, *atomic.Value) {
	t.Helper()
	calls, lastIDs := &atomic.Int32{}, &atomic.Value{}
	upstream := fakeUpstream(calls, lastIDs)
	t.Cleanup(upstream.Close)

	catalog := clients.NewCatalogClient(upstream.URL)
	schema := graph.NewSchema(clients.NewOrderClient(upstream.URL), dev)
	gw := httptest.NewServer(NewRouter(schema, catalog, dev))
	t.Cleanup(gw.Close)
	return gw, calls, lastIDs
}

func query(t *testing.T, url, q string) gqlResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": q})
	resp, err := nethttp.Post(url+"/graphql", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out gqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestOrdersByUserIdBatchesCatalogCalls(t *testing.T) {
	gw, calls, lastIDs := newGateway(t, true)

	res := query(t, gw.URL, `{ ordersByUserId(userId: "`+user+`") { id createdAt items { quantity product { id name price } } } }`)
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
	orders := res.Data.OrdersByUserID
	if len(orders) != 2 {
		t.Fatalf("got %d orders, want 2", len(orders))
	}
	first := orders[0].Items[0].Product
	if first.Name != "Keyboard" || first.Price != 49.99 {
		t.Fatalf("first product = %+v, want Keyboard 49.99", first)
	}
	if orders[1].Items[0].Product.Name != "Keyboard" {
		t.Fatalf("second order product = %+v", orders[1].Items[0].Product)
	}

	if n := calls.Load(); n != 1 {
		t.Fatalf("catalog called %d times, want 1 (N+1 not prevented)", n)
	}
	// p1 is used in both orders but must be requested only once.
	if ids := lastIDs.Load().([]string); len(ids) != 2 {
		t.Fatalf("catalog asked for %v, want 2 unique ids", ids)
	}
}

func TestIntrospectionShowsDescriptionsInDev(t *testing.T) {
	gw, _, _ := newGateway(t, true)
	res := query(t, gw.URL, `{ __type(name: "Order") { description } }`)
	if res.Data.Type == nil || !strings.Contains(res.Data.Type.Description, "customer order") {
		t.Fatalf("introspection did not return description: %+v", res)
	}
}

func TestIntrospectionDisabledInProduction(t *testing.T) {
	gw, _, _ := newGateway(t, false)
	res := query(t, gw.URL, `{ __type(name: "Order") { description } }`)
	if res.Data.Type != nil {
		t.Fatalf("introspection must be disabled in production, got %+v", res.Data.Type)
	}
}

func TestSandboxServedOnlyInDev(t *testing.T) {
	for _, dev := range []bool{true, false} {
		gw, _, _ := newGateway(t, dev)
		req, _ := nethttp.NewRequest(nethttp.MethodGet, gw.URL+"/graphql", nil)
		req.Header.Set("Accept", "text/html")
		resp, err := nethttp.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		want := nethttp.StatusOK
		if !dev {
			want = nethttp.StatusMethodNotAllowed
		}
		if resp.StatusCode != want {
			t.Errorf("dev=%t: GET /graphql status = %d, want %d", dev, resp.StatusCode, want)
		}
	}
}
