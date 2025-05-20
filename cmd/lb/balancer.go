package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 10, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
	healthyServers []string
	serversMutex   sync.RWMutex
)

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func getHealthyServers() []string {
	serversMutex.RLock()
	defer serversMutex.RUnlock()
	return append([]string{}, healthyServers...)
}

func updateServerHealth(server string, isHealthy bool) {
	serversMutex.Lock()
	defer serversMutex.Unlock()

	if isHealthy {
		found := false
		for _, s := range healthyServers {
			if s == server {
				found = true
				break
			}
		}
		if !found {
			healthyServers = append(healthyServers, server)
			log.Printf("Server %s added to healthy pool", server)
		}
	} else {
		for i, s := range healthyServers {
			if s == server {
				healthyServers = append(healthyServers[:i], healthyServers[i+1:]...)
				log.Printf("Server %s removed from healthy pool", server)
				break
			}
		}
	}
}

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	// Додамо логування для діагностики
	log.Printf("Forwarding request from %s to %s", r.RemoteAddr, dst)

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err != nil {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}

	for k, values := range resp.Header {
		for _, value := range values {
			rw.Header().Add(k, value)
		}
	}
	if *traceEnabled {
		rw.Header().Set("lb-from", dst)
	}
	log.Println("fwd", resp.StatusCode, resp.Request.URL)
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()

	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		log.Printf("Failed to write response: %s", err)
		return err
	}
	return nil
}

func selectServerByClientHash(remoteAddr string) (string, error) {
	servers := getHealthyServers()
	if len(servers) == 0 {
		return "", fmt.Errorf("no healthy servers available")
	}

	// Детальне логування для діагностики
	log.Printf("Selecting server for client: %s (healthy servers: %v)", remoteAddr, servers)

	hash := hashString(remoteAddr)
	serverIndex := int(hash) % len(servers)
	selected := servers[serverIndex]

	log.Printf("Selected server: %s (index: %d)", selected, serverIndex)
	return selected, nil
}

func main() {
	flag.Parse()

	for _, server := range serversPool {
		if health(server) {
			updateServerHealth(server, true)
		}
	}

	for _, server := range serversPool {
		server := server
		go func() {
			for range time.Tick(10 * time.Second) {
				isHealthy := health(server)
				log.Println(server, "healthy:", isHealthy)
				updateServerHealth(server, isHealthy)
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientAddr := r.RemoteAddr

		server, err := selectServerByClientHash(clientAddr)
		if err != nil {
			log.Printf("Failed to select server: %s", err)
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if err := forward(server, rw, r); err != nil {
			log.Printf("Failed to forward request: %s", err)
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
