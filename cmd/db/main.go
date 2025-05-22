package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/roman-mazur/architecture-practice-4-template/design-db-practice/datastore"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data"
	}

	// Ensure database directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Printf("Failed to create database directory: %v\n", err)
		os.Exit(1)
	}

	db, err := datastore.Open(dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	http.HandleFunc("/db/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[len("/db/"):]
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			value, err := db.Get(key)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"key":   key,
				"value": value,
			})

		case http.MethodPost:
			var request struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := db.Put(key, request.Value); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Database server started on :8083")
	http.ListenAndServe(":8083", nil)
}
