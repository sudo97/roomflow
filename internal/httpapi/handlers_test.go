package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/store"
)

func do(t *testing.T, h http.Handler, method, path, tenant, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if tenant != "" {
		r.Header.Set(tenantHeader, tenant)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestBookingFlow(t *testing.T) {
	h := New(booking.NewService(store.NewMemory())).Routes()
	steps := []struct {
		name, method, path, tenant, body string
		want                             int
	}{
		{"health", "GET", "/healthz", "", "", 200},
		{"no tenant", "GET", "/api/rooms", "", "", 401},
		{"bad json", "POST", "/api/rooms", "a", "{", 400},
		{"invalid room", "POST", "/api/rooms", "a", `{"id":"r1"}`, 400},
		{"add room", "POST", "/api/rooms", "a", `{"id":"r1","name":"Double","capacity":2,"price":1000}`, 201},
		{"list rooms", "GET", "/api/rooms", "a", "", 200},
		{"book", "POST", "/api/bookings", "a", `{"room_id":"r1","guest_name":"Ivan","guest_email":"i@x.io","guests":2,"check_in":"2026-03-02","check_out":"2026-03-04"}`, 201},
		{"double booking", "POST", "/api/bookings", "a", `{"room_id":"r1","guest_name":"Olha","guest_email":"o@x.io","guests":1,"check_in":"2026-03-03","check_out":"2026-03-05"}`, 409},
		{"other tenant room", "POST", "/api/bookings", "b", `{"room_id":"r1","guest_name":"Ivan","guest_email":"i@x.io","guests":1,"check_in":"2026-03-02","check_out":"2026-03-04"}`, 404},
		{"bad booking json", "POST", "/api/bookings", "a", "nope", 400},
		{"invalid booking", "POST", "/api/bookings", "a", `{"room_id":"r1"}`, 400},
		{"cancel", "POST", "/api/bookings/bk-1/cancel", "a", "", 204},
		{"cancel missing", "POST", "/api/bookings/bk-9/cancel", "a", "", 404},
	}
	for _, s := range steps {
		if w := do(t, h, s.method, s.path, s.tenant, s.body); w.Code != s.want {
			t.Fatalf("%s: status %d, want %d, body %s", s.name, w.Code, s.want, w.Body)
		}
	}
}

func TestStatusForUnknownError(t *testing.T) {
	if statusFor(http.ErrAbortHandler) != http.StatusInternalServerError {
		t.Fatal("unknown errors must map to 500")
	}
}
