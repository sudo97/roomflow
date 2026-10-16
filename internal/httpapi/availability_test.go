package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/obs"
	"github.com/sudo97/roomflow/internal/store"
)

// mapCache is an in-memory cache.Cache for tests.
type mapCache struct {
	data     map[string][]byte
	versions map[string]int64
}

func (c *mapCache) Get(_ context.Context, k string) ([]byte, bool) { v, ok := c.data[k]; return v, ok }
func (c *mapCache) Set(_ context.Context, k string, v []byte, _ time.Duration) {
	c.data[k] = v
}
func (c *mapCache) Version(_ context.Context, s string) int64 { return c.versions[s] }
func (c *mapCache) Bump(_ context.Context, s string)          { c.versions[s]++ }

func TestAvailabilityCacheAside(t *testing.T) {
	reg := obs.NewRegistry()
	c := &mapCache{data: map[string][]byte{}, versions: map[string]int64{}}
	h := New(booking.NewService(store.NewMemory())).WithCache(c).WithMetrics(reg).Routes()
	do(t, h, "POST", "/api/rooms", "a", `{"id":"r1","name":"Double","capacity":2,"price":1000}`)
	url := "/api/rooms/r1/availability?from=2026-03-02&to=2026-03-04"
	first := do(t, h, "GET", url, "a", "")
	second := do(t, h, "GET", url, "a", "")
	if first.Code != 200 || second.Body.String() != first.Body.String() || strings.Contains(first.Body.String(), "true") {
		t.Fatalf("first %d %s second %s", first.Code, first.Body, second.Body)
	}
	if reg.Value("roomflow_cache_hits_total", "") != 1 || reg.Value("roomflow_cache_misses_total", "") != 1 {
		t.Fatal("hit/miss not counted")
	}
	do(t, h, "POST", "/api/bookings", "a", `{"room_id":"r1","guest_name":"Ivan","guest_email":"i@x.io","guests":1,"check_in":"2026-03-02","check_out":"2026-03-03"}`)
	if after := do(t, h, "GET", url, "a", ""); !strings.Contains(after.Body.String(), `"booked":true`) {
		t.Fatalf("booking must invalidate the cached calendar: %s", after.Body)
	}
	do(t, h, "POST", "/api/bookings/bk-1/cancel", "a", "")
	if after := do(t, h, "GET", url, "a", ""); strings.Contains(after.Body.String(), `"booked":true`) {
		t.Fatalf("cancellation must invalidate the cached calendar: %s", after.Body)
	}
	if bad := do(t, h, "GET", "/api/rooms/r1/availability?from=x", "a", ""); bad.Code != 400 {
		t.Fatalf("bad dates: %d", bad.Code)
	}
}
