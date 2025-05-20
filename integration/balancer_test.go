package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 10 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// Чекаємо, поки всі сервери стануть доступними
	time.Sleep(15 * time.Second)

	servers := make(map[string]bool)
	requests := 10

	// Використовуємо різні порти для унікальних RemoteAddr
	for i := 0; i < requests; i++ {
		client := http.Client{Timeout: 3 * time.Second}
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/some-data", baseAddress), nil)
		if err != nil {
			t.Fatal(err)
		}

		// Імітуємо різних клієнтів через різні порти
		req.RemoteAddr = fmt.Sprintf("192.168.1.%d:%d", i%3+1, 50000+i)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		defer resp.Body.Close()

		server := resp.Header.Get("lb-from")
		if server == "" {
			t.Errorf("Request %d: missing 'lb-from' header", i+1)
			continue
		}

		servers[server] = true
		t.Logf("Request %d from %s served by: %s", i+1, req.RemoteAddr, server)
	}

	if len(servers) < 2 {
		t.Errorf("Expected requests to be distributed between servers, but got only %d server(s): %v", len(servers), servers)
	} else {
		t.Logf("Success: requests distributed between %d servers: %v", len(servers), servers)
	}
}

func TestClientConsistency(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// Створюємо окремий клієнт для тесту
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Фіксуємо локальний IP для всіх з'єднань
				dialer := &net.Dialer{
					LocalAddr: &net.TCPAddr{IP: net.ParseIP("192.168.1.100")},
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	var firstServer string
	const requestsCount = 5

	for i := 0; i < requestsCount; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		server := resp.Header.Get("lb-from")
		if server == "" {
			t.Error("'lb-from' header is missing")
			continue
		}

		if firstServer == "" {
			firstServer = server
		} else if server != firstServer {
			t.Errorf("Expected same server (%s) for same client, got %s", firstServer, server)
		}
		t.Logf("Request %d: served by %s", i+1, server)
	}
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			b.Error(err)
			continue
		}
		resp.Body.Close()
	}
}
