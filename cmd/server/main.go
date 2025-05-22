package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var port = flag.Int("port", 8080, "server port")
var dbHost = flag.String("db-host", "db", "database host address")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"
const teamName = "bebra" // Замініть на ім'я вашої команди

func main() {
	h := new(http.ServeMux)

	// Initialize data in DB
	initDB()

	h.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	h.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {
		respDelayString := os.Getenv(confResponseDelaySec)
		if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		report.Process(r)

		key := r.URL.Query().Get("key")
		if key == "" {
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = rw.Write([]byte("key parameter is required"))
			return
		}

		resp, err := http.Get(fmt.Sprintf("http://%s:8083/db/%s", *dbHost, key))
		if err != nil || resp.StatusCode == http.StatusNotFound {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		defer resp.Body.Close()

		var data map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("content-type", "application/json")
		json.NewEncoder(rw).Encode(data)
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}

func initDB() {
	currentTime := time.Now().Format("2006-01-02")
	requestBody, _ := json.Marshal(map[string]string{"value": currentTime})

	resp, err := http.Post(
		fmt.Sprintf("http://%s:8083/db/%s", *dbHost, teamName),
		"application/json",
		bytes.NewBuffer(requestBody),
	)

	if err != nil || resp.StatusCode != http.StatusOK {
		panic("Failed to initialize database")
	}
}
