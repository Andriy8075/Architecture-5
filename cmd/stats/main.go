package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/roman-mazur/architecture-practice-4-template/design-db-practice/datastore"
	"log"
	"net/http"
	"os"
	"time"
)

var https = flag.Bool("https", false, "whether backends support HTTPs")

var serversPool = []string{
	"localhost:8080",
	"localhost:8081",
	"localhost:8082",
}

type report map[string][]string

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func main() {
	flag.Parse()

	// Блок тестування datastore
	log.Println("== Testing datastore with segment support ==")
	testDataStore()
	log.Println("== End of datastore test ==")

	// HTTP-запит до серверів
	client := new(http.Client)
	client.Timeout = 10 * time.Second

	res := make([]report, len(serversPool))
	for i, s := range serversPool {
		resp, err := client.Get(fmt.Sprintf("%s://%s/report", scheme(), s))
		if err == nil {
			var data report
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			} else {
				for k, v := range data {
					l := len(v)
					if l > 5 {
						l = 5
					}
					data[k] = v[len(v)-l:]
				}
				res[i] = data
			}
		} else {
			log.Printf("error %s %s", s, err)
		}

		log.Println("=========================")
		log.Println("SERVER", i, serversPool[i])
		log.Println("=========================")
		data, _ := json.MarshalIndent(res[i], "", "  ")
		log.Println(string(data))
	}
}

func testDataStore() {
	dir := "testdata"
	os.MkdirAll(dir, 0755)

	db, err := datastore.Open(dir)
	if err != nil {
		log.Fatal("Open error:", err)
	}
	defer db.Close()

	err = db.Put("name", "Alice")
	check(err)
	err = db.Put("age", "30")
	check(err)
	err = db.Put("name", "Bob") // overwrite
	check(err)

	val, err := db.Get("name")
	check(err)
	log.Println("Got name:", val)

	val, err = db.Get("age")
	check(err)
	log.Println("Got age:", val)

	size, err := db.Size()
	check(err)
	log.Println("DB size (bytes):", size)

	db.Close()

	// Reopen to test segment recovery
	db2, err := datastore.Open(dir)
	check(err)
	defer db2.Close()

	val, err = db2.Get("name")
	check(err)
	log.Println("Recovered name:", val)

	val, err = db2.Get("age")
	check(err)
	log.Println("Recovered age:", val)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
