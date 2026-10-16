//go:build e2e

// E2E-тести проти розгорнутого контейнера (BASE_URL), як їх бачить зовнішній клієнт.
package e2e

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func base(t *testing.T) string {
	u := os.Getenv("BASE_URL")
	if u == "" {
		t.Skip("BASE_URL is not set")
	}
	return u
}

func post(t *testing.T, url, tenant, body string) int {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestGuestBooksARoomEndToEnd(t *testing.T) {
	u := base(t)
	tenant := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	steps := []struct {
		name, path, body string
		want             int
	}{
		{"owner adds room", "/api/rooms", `{"id":"r1","name":"Double","capacity":2,"price":1000}`, 201},
		{"guest books", "/api/bookings", `{"room_id":"r1","guest_name":"Ivan","guest_email":"i@x.io","guests":2,"check_in":"2030-03-02","check_out":"2030-03-04"}`, 201},
		{"second guest gets conflict", "/api/bookings", `{"room_id":"r1","guest_name":"Olha","guest_email":"o@x.io","guests":1,"check_in":"2030-03-03","check_out":"2030-03-05"}`, 409},
	}
	for _, s := range steps {
		if got := post(t, u+s.path, tenant, s.body); got != s.want {
			t.Fatalf("%s: status %d, want %d", s.name, got, s.want)
		}
	}
}
