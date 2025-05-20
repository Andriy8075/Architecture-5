package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Testing updates to healthy servers
func TestUpdateServerHealth(t *testing.T) {
	healthyServers = []string{}

	updateServerHealth("server1:8080", true)
	if len(healthyServers) != 1 {
		t.Errorf("Expected 1 healthy server, got %d", len(healthyServers))
	}

	updateServerHealth("server2:8080", true)
	if len(healthyServers) != 2 {
		t.Errorf("Expected 2 healthy servers, got %d", len(healthyServers))
	}

	updateServerHealth("server1:8080", false)
	if len(healthyServers) != 1 {
		t.Errorf("Expected 1 healthy server after removal, got %d", len(healthyServers))
	}

	if healthyServers[0] != "server2:8080" {
		t.Errorf("Expected remaining server to be server2:8080, got %s", healthyServers[0])
	}
}

// Testing server selection by client IP address
func TestSelectServerByClientHash(t *testing.T) {
	healthyServers = []string{"server1:8080", "server2:8080", "server3:8080"}

	addr1 := "192.168.0.101:12345"
	addr2 := "192.168.0.103:12345"

	s1, err := selectServerByClientHash(addr1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	s2, err := selectServerByClientHash(addr2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if s1 == s2 {
		t.Errorf("Expected different servers for different IPs, got same: %s", s1)
	}

	s1Again, _ := selectServerByClientHash(addr1)
	if s1 != s1Again {
		t.Errorf("Expected consistent hash result, got %s and %s", s1, s1Again)
	}
}

// Testing whether getHealthyServers returns a copy
func TestGetHealthyServers(t *testing.T) {
	healthyServers = []string{"server1:8080"}
	list := getHealthyServers()

	list[0] = "fake"

	if healthyServers[0] == "fake" {
		t.Errorf("Expected original slice to remain unchanged")
	}
}

// Testing proxying the request via forward (via a fake backend)
func TestForward(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer backend.Close()

	parts := strings.Split(backend.URL, "//")
	dst := parts[1]

	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	err := forward(dst, w, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if string(body) != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", string(body))
	}
}
