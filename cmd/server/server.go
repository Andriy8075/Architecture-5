package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/design-db-practice/datastore"
	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var port = flag.Int("port", 8080, "server port")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	h := new(http.ServeMux)

	// Initialize database
	db, err := datastore.Open("/opt/practice-4/db")
	if err != nil {
		panic("Failed to open database: " + err.Error())
	}
	defer db.Close()

	// Insert initial data with current time
	currentTime := time.Now().Format(time.RFC3339)
	if err := db.Put("bebra", currentTime); err != nil {
		panic("Failed to put initial data: " + err.Error())
	}

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

		value, err := db.Get(key)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write([]byte("data not found"))
			return
		}

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"key":   key,
			"value": value,
		})
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}
