package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	baseAddress = "http://balancer:8090"
	dbAddress   = "http://db:8083"
	teamName    = "bebra" // Замініть на ім'я вашої команди
)

var client = http.Client{
	Timeout: 10 * time.Second,
}

func TestClientConsistency(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// Створюємо окремий клієнт для тесту
	client := &http.Client{
		Timeout: 10 * time.Second,
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
			t.Errorf("Expected same server (%s) for same client, got %s\n", firstServer, server)
		}
		t.Logf("Request %d: served by %s", i+1, server)
	}
}

func TestDatabaseIntegration(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// 1. Спочатку перевіримо, чи база даних доступна
	resp, err := client.Get(fmt.Sprintf("%s/db/%s", dbAddress, teamName))
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 from DB, got %d", resp.StatusCode)
	}

	// 2. Тестуємо, чи сервер правильно взаємодіє з базою даних
	testKey := "test_key_" + time.Now().Format("20060102150405")
	testValue := "test_value_" + time.Now().Format("20060102150405")

	// Додаємо тестові дані в базу
	putDataToDB(t, testKey, testValue)

	// Перевіряємо, чи сервер може отримати ці дані
	checkServerResponse(t, testKey, testValue)

	// 3. Перевіряємо обробку відсутніх даних
	nonExistentKey := "non_existent_key_" + time.Now().Format("20060102150405")
	checkNonExistentKey(t, nonExistentKey)
}

func putDataToDB(t *testing.T, key, value string) {
	requestBody, err := json.Marshal(map[string]string{"value": value})
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	resp, err := client.Post(
		fmt.Sprintf("%s/db/%s", dbAddress, key),
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		t.Fatalf("Failed to put data to DB: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 when putting data to DB, got %d", resp.StatusCode)
	}
}

func checkServerResponse(t *testing.T, key, expectedValue string) {
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, key))
	if err != nil {
		t.Fatalf("Request to server failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 from server, got %d", resp.StatusCode)
		return
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if data["key"] != key {
		t.Errorf("Expected key %s, got %s", key, data["key"])
	}

	if data["value"] != expectedValue {
		t.Errorf("Expected value %s, got %s", expectedValue, data["value"])
	}
}

func checkNonExistentKey(t *testing.T, key string) {
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, key))
	if err != nil {
		t.Fatalf("Request to server failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent key, got %d", resp.StatusCode)
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
