package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProductCRUDFlow(t *testing.T) {
	server, instanceID := newTestServer()
	defer server.Close()

	// Create.
	resp, err := http.Post(server.URL+"/api/v1/products", "application/json",
		strings.NewReader(`{"name":"Widget","description":"A widget","price":1999,"stock":10}`))
	if err != nil {
		t.Fatalf("create request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if got := resp.Header.Get("X-Instance-Id"); got != instanceID {
		t.Fatalf("X-Instance-Id = %q, want %q", got, instanceID)
	}
	var created struct {
		ID    string `json:"id"`
		Price int64  `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("create response did not include an id")
	}
	if created.Price != 1999 {
		t.Fatalf("create price = %d, want 1999", created.Price)
	}

	// List.
	resp, err = http.Get(server.URL + "/api/v1/products")
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var list struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if list.Count != 1 {
		t.Fatalf("list count = %d, want 1", list.Count)
	}

	// Update.
	req, err := http.NewRequest(http.MethodPut, server.URL+"/api/v1/products/"+created.ID,
		strings.NewReader(`{"name":"Widget v2","description":"updated","price":2500,"stock":5}`))
	if err != nil {
		t.Fatalf("build update request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var updated struct {
		Name  string `json:"name"`
		Price int64  `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Name != "Widget v2" || updated.Price != 2500 {
		t.Fatalf("update response = %+v, want name=Widget v2 price=2500", updated)
	}

	// Get.
	resp, err = http.Get(server.URL + "/api/v1/products/" + created.ID)
	if err != nil {
		t.Fatalf("get request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Delete.
	req, err = http.NewRequest(http.MethodDelete, server.URL+"/api/v1/products/"+created.ID, nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Get after delete -> 404.
	resp, err = http.Get(server.URL + "/api/v1/products/" + created.ID)
	if err != nil {
		t.Fatalf("get-after-delete request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
