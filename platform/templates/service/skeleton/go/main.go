// Command ${{ values.name }}: ${{ values.description }}
package main

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var requests atomic.Int64

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("# TYPE service_requests_total counter\nservice_requests_total " + strconv.FormatInt(requests.Load(), 10) + "\n"))
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); mux.ServeHTTP(w, r) })
}

func main() {
	srv := &http.Server{Addr: ":8080", Handler: handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
