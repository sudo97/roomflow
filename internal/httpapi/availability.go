package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/cache"
	"github.com/sudo97/roomflow/internal/trace"
)

// availabilityTTL bounds staleness if an invalidation is lost (e.g. the cache was unreachable).
const availabilityTTL = 60 * time.Second

// WithCache enables cache-aside for read endpoints.
func (a *API) WithCache(c cache.Cache) *API { a.cache = c; return a }

// availability serves a room calendar using cache-aside: read the cache, on a miss compute and store.
// The tenant's version is part of the key, so any booking change makes old entries unreachable.
func (a *API) availability(w http.ResponseWriter, r *http.Request, tenant booking.TenantID) {
	ctx := r.Context()
	room, from, to := r.PathValue("id"), r.URL.Query().Get("from"), r.URL.Query().Get("to")
	ctx, span := trace.Start(ctx, "cache.lookup")
	key := fmt.Sprintf("avail:%s:v%d:%s:%s:%s", tenant, a.cache.Version(ctx, string(tenant)), room, from, to)
	cached, hit := a.cache.Get(ctx, key)
	span.Set("cache.hit", fmt.Sprint(hit))
	span.Finish()
	if hit {
		a.count("roomflow_cache_hits_total", "Availability responses served from cache.")
		writeRawJSON(w, cached)
		return
	}
	a.count("roomflow_cache_misses_total", "Availability responses computed (cache miss).")
	_, span = trace.Start(ctx, "booking.availability")
	nights, err := a.svc.Availability(tenant, room, from, to)
	span.Finish()
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	body, _ := json.Marshal(nights) // []Night always marshals
	body = append(body, '\n')
	a.cache.Set(ctx, key, body, availabilityTTL)
	writeRawJSON(w, body)
}
