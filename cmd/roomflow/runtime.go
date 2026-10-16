package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // served only on PPROF_ADDR, a separate internal port (G108)
	"os"
	"strconv"
	"time"

	"github.com/sudo97/roomflow/internal/cache"
	"github.com/sudo97/roomflow/internal/chaos"
	"github.com/sudo97/roomflow/internal/obs"
	"github.com/sudo97/roomflow/internal/ratelimit"
	"github.com/sudo97/roomflow/internal/trace"
)

func envFloat(key string) float64 {
	v, _ := strconv.ParseFloat(os.Getenv(key), 64)
	return v
}

// newCache returns a Redis-backed cache when CACHE_REDIS_ADDR is set, otherwise a disabled cache.
func newCache(reg *obs.Registry) cache.Cache {
	addr := os.Getenv("CACHE_REDIS_ADDR")
	if addr == "" {
		return cache.Nop{}
	}
	c := cache.NewRedis(addr, 64)
	c.OnError = func(error) {
		reg.Add("roomflow_cache_errors_total", "Failed cache commands (served as misses).", "", 1)
	}
	return c
}

// withMiddleware wraps the router (outermost first): metrics and logs, tracing, per-tenant rate limiting
// (RATE_LIMIT_RPS, RATE_LIMIT_BURST) and optional fault injection (FAULT_RATE, chaos experiments only).
func withMiddleware(reg *obs.Registry, logger *slog.Logger, route func(*http.Request) string, h http.Handler) http.Handler {
	h = chaos.Faults(envFloat("FAULT_RATE"), h)
	if rps := envFloat("RATE_LIMIT_RPS"); rps > 0 {
		burst := int(envFloat("RATE_LIMIT_BURST"))
		if burst <= 0 {
			burst = int(rps)
		}
		tenant := func(r *http.Request) string { return r.Header.Get("X-Tenant-ID") }
		rejected := func() {
			reg.Add("roomflow_rate_limited_total", "Requests rejected by the per-tenant rate limiter.", "", 1)
		}
		h = ratelimit.New(rps, burst).Middleware(tenant, rejected, h)
	}
	tracer := trace.New(envOr("OTEL_SERVICE_NAME", "roomflow"), os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	go tracer.Run(2*time.Second, nil)
	h = tracer.Middleware(route, h)
	return obs.Instrument(reg, logger, route, h)
}

// servePprof exposes Go runtime profiles on a separate internal port when PPROF_ADDR is set.
func servePprof(logger *slog.Logger) {
	addr := os.Getenv("PPROF_ADDR")
	if addr == "" {
		return
	}
	srv := &http.Server{Addr: addr, Handler: http.DefaultServeMux, ReadHeaderTimeout: 5 * time.Second}
	go func() { logger.Error("pprof server stopped", "err", srv.ListenAndServe()) }()
}
