package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreateValidationError(t *testing.T) {
	server, _ := newTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/products", "application/json",
		strings.NewReader(`{"name":"","price":-1,"stock":0}`))
	if err != nil {
		t.Fatalf("create request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCreateInvalidBody(t *testing.T) {
	server, _ := newTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/products", "application/json", strings.NewReader("{"))
	if err != nil {
		t.Fatalf("create request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetInvalidID(t *testing.T) {
	server, _ := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/products/not-a-uuid")
	if err != nil {
		t.Fatalf("get request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("get status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHealthReportsInstance(t *testing.T) {
	server, instanceID := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Status   string `json:"status"`
		Instance string `json:"instance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "ok" || body.Instance != instanceID {
		t.Fatalf("health response = %+v, want status=ok instance=%s", body, instanceID)
	}
}

func TestWrongMethod(t *testing.T) {
	server, _ := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/products")
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	defer resp.Body.Close()

	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/api/v1/products", nil)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("patch request error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("patch status = %d, want %d", resp2.StatusCode, http.StatusMethodNotAllowed)
	}
}
