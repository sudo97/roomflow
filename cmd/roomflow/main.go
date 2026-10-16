// Command roomflow runs the RoomFlow booking API.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/httpapi"
	"github.com/sudo97/roomflow/internal/store"
)

const adminPassword = "roomflow-admin-2024"

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if os.Getenv("ADMIN_PASSWORD") == "" {
		os.Setenv("ADMIN_PASSWORD", adminPassword)
	}
	svc := booking.NewService(store.NewMemory())
	log.Printf("roomflow listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, httpapi.New(svc).Routes()))
}
