//go:build integration

// Інтеграційні тести: справжній HTTP-сервер + сервіс + сховище, клієнт ходить через мережу.
package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/httpapi"
	"github.com/sudo97/roomflow/internal/store"
)

type client struct {
	t      *testing.T
	base   string
	tenant string
}

func (c client) do(method, path, body string) (int, map[string]any) {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.base+path, strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", c.tenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func newServer(t *testing.T) string {
	srv := httptest.NewServer(httpapi.New(booking.NewService(store.NewMemory())).Routes())
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestCreateAndCancelBookingOverHTTP(t *testing.T) {
	c := client{t, newServer(t), "hotel-a"}
	if code, _ := c.do("POST", "/api/rooms", `{"id":"r1","name":"Double","capacity":2,"price":1000}`); code != 201 {
		t.Fatalf("add room: %d", code)
	}
	code, b := c.do("POST", "/api/bookings", `{"room_id":"r1","guest_name":"Ivan","guest_email":"i@x.io","guests":2,"check_in":"2026-03-07","check_out":"2026-03-09"}`)
	if code != 201 || b["Total"].(float64) != 2400 {
		t.Fatalf("book: %d %v", code, b)
	}
	if code, _ := c.do("POST", "/api/bookings/"+b["ID"].(string)+"/cancel", ""); code != 204 {
		t.Fatalf("cancel: %d", code)
	}
	if code, _ := c.do("POST", "/api/bookings", `{"room_id":"r1","guest_name":"Olha","guest_email":"o@x.io","guests":1,"check_in":"2026-03-07","check_out":"2026-03-08"}`); code != 201 {
		t.Fatalf("rebook after cancel: %d", code)
	}
}

func TestTenantsCannotSeeEachOther(t *testing.T) {
	base := newServer(t)
	a, b := client{t, base, "hotel-a"}, client{t, base, "hotel-b"}
	a.do("POST", "/api/rooms", `{"id":"r1","name":"Double","capacity":2,"price":1000}`)
	code, _ := b.do("POST", "/api/bookings", `{"room_id":"r1","guest_name":"X","guest_email":"x@x.io","guests":1,"check_in":"2026-03-02","check_out":"2026-03-03"}`)
	if code != 404 {
		t.Fatalf("tenant b booked tenant a room: %d", code)
	}
}
