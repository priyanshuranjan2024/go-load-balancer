
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
    "net"
    "errors"
)

func main() {
	targets := []string{
		"http://localhost:9001",
		"http://localhost:9002",
	}

	proxies := make([]*httputil.ReverseProxy, len(targets))
	healthy := make([]bool, len(targets))

	for i, target := range targets {
		backendURL, err := url.Parse(target)
		if err != nil {
			log.Fatal(err)
		}

		proxies[i] = httputil.NewSingleHostReverseProxy(backendURL)

        proxies[i].Transport = &http.Transport{
    ResponseHeaderTimeout: 3 * time.Second,
}


proxies[i].ErrorHandler = func(
    w http.ResponseWriter,
    r *http.Request,
    err error,
) {
    log.Printf("Backend request failed: %v", err)

    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        http.Error(
            w,
            "Gateway Timeout",
            http.StatusGatewayTimeout,
        )
        return
    }

    http.Error(
        w,
        "Bad Gateway",
        http.StatusBadGateway,
    )
}



	}

	var mu sync.Mutex
	next := 0

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	checkBackends := func() {
		for i, target := range targets {
			resp, err := client.Get(target + "/health")

			isHealthy := err == nil &&
				resp.StatusCode >= 200 &&
				resp.StatusCode < 300

			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}

			if err != nil {
				log.Printf("Health check failed for %s: %v", target, err)
			}

			mu.Lock()
			wasHealthy := healthy[i]
			healthy[i] = isHealthy
			mu.Unlock()

			if wasHealthy != isHealthy {
				log.Printf("Backend %s healthy: %t", target, isHealthy)
			}
		}
	}

	// Check backends immediately, then every 5 seconds.
	checkBackends()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			checkBackends()
		}
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()

		selected := -1

		for i := 0; i < len(proxies); i++ {
			index := (next + i) % len(proxies)

			if healthy[index] {
				selected = index
				next = (index + 1) % len(proxies)
				break
			}
		}

		mu.Unlock()

		if selected == -1 {
			http.Error(
				w,
				"No healthy backends available",
				http.StatusServiceUnavailable,
			)
			return
		}

		log.Printf("%s %s -> %s", r.Method, r.URL.Path, targets[selected])

		proxies[selected].ServeHTTP(w, r)
	})

	fmt.Println("Load balancer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
