// Command roomflow runs the RoomFlow booking API.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/httpapi"
	"github.com/sudo97/roomflow/internal/store"
)

// Set at build time via -ldflags.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local /healthz endpoint and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	addr := envOr("ADDR", ":8080")
	switch {
	case *showVersion:
		fmt.Printf("roomflow %s (%s)\n", version, commit)
	case *healthcheck:
		os.Exit(probe("http://localhost" + addr + "/healthz"))
	default:
		serve(addr)
	}
}

func serve(addr string) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(booking.NewService(store.NewMemory())).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("roomflow %s (%s) listening on %s", version, commit, addr)
	log.Fatal(srv.ListenAndServe())
}

// probe returns a process exit code: 0 if the endpoint answers 200, 1 otherwise.
func probe(url string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
