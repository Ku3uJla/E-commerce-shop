// Package discovery registers a service instance with Consul so that the API
// gateway (and other consumers) can find healthy instances through Consul DNS.
package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Registration describes the service instance to advertise.
type Registration struct {
	ID              string
	Name            string
	Address         string
	Port            int
	Tags            []string
	HealthCheckPath string
}

// Client is a tiny Consul HTTP API client (no external dependencies).
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds a Consul client for the given host:port (e.g. "consul:8500").
func NewClient(addr string) *Client {
	return &Client{
		baseURL: "http://" + addr,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type consulService struct {
	ID      string         `json:"ID"`
	Name    string         `json:"Name"`
	Address string         `json:"Address"`
	Port    int            `json:"Port"`
	Tags    []string       `json:"Tags,omitempty"`
	Check   map[string]any `json:"Check,omitempty"`
}

// Register advertises the instance, including an HTTP health check.
func (c *Client) Register(reg Registration) error {
	payload := consulService{
		ID:      reg.ID,
		Name:    reg.Name,
		Address: reg.Address,
		Port:    reg.Port,
		Tags:    reg.Tags,
	}
	if reg.HealthCheckPath != "" {
		payload.Check = map[string]any{
			"HTTP":                           fmt.Sprintf("http://%s:%d%s", reg.Address, reg.Port, reg.HealthCheckPath),
			"Interval":                       "5s",
			"Timeout":                        "2s",
			"DeregisterCriticalServiceAfter": "30s",
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/v1/agent/service/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("consul register failed: %s", resp.Status)
	}
	return nil
}

// Deregister removes the instance from Consul.
func (c *Client) Deregister(serviceID string) error {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/v1/agent/service/deregister/"+serviceID, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("consul deregister failed: %s", resp.Status)
	}
	return nil
}

// LocalIP returns the container's outbound IP address, which is reachable by
// the other containers on the same Docker network.
func LocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	return addr.IP.String()
}
