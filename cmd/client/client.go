package main

import (
	"flag"
	"fmt"
	"github.com/roman-mazur/architecture-practice-4-template/design-db-practice/datastore"
	"log"
	"net/http"
	"os"
	"time"
)

var target = flag.String("target", "http://localhost:8090", "request target")

func main() {
	flag.Parse()
	client := new(http.Client)
	client.Timeout = 10 * time.Second

	dir := "realdb"
	os.MkdirAll(dir, 0755)

	db, err := datastore.Open(dir)
	if err != nil {
		log.Fatal("Open error:", err)
	}
	defer db.Close()

	err = db.Put("bebra", "Alice")

	for range time.Tick(1 * time.Second) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", *target))
		if err == nil {
			log.Printf("response %d", resp.StatusCode)
		} else {
			log.Printf("error %s", err)
		}
	}
}
