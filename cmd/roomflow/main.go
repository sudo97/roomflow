// Command roomflow runs the RoomFlow booking API.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/httpapi"
	"github.com/sudo97/roomflow/internal/store"
)

func main() {
	srv := &http.Server{
		Addr:              envOr("ADDR", ":8080"),
		Handler:           httpapi.New(booking.NewService(store.NewMemory())).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("roomflow listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
