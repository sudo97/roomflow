package obs

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const labelRoute = "route"

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func requestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Instrument wraps a handler with RED metrics (rate, errors, duration) and a JSON access log.
// route returns a low-cardinality route name for the request (e.g. "POST /api/bookings").
func Instrument(reg *Registry, log *slog.Logger, route func(*http.Request) string, next http.Handler) http.Handler {
	var inFlight atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reg.Set("roomflow_http_in_flight_requests", "Requests being served.", "", float64(inFlight.Add(1)))
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = requestID()
		}
		sw.Header().Set("X-Request-ID", id)
		next.ServeHTTP(sw, r)
		inFlight.Add(-1)

		rt, code := route(r), strconv.Itoa(sw.status)
		elapsed := time.Since(start)
		reg.Add("roomflow_http_requests_total", "HTTP requests by route and status code.", Labels(labelRoute, rt, "code", code), 1)
		reg.Observe("roomflow_http_request_duration_seconds", "HTTP request latency.", Labels(labelRoute, rt), elapsed.Seconds())
		level := slog.LevelInfo
		if sw.status >= 500 {
			level = slog.LevelError
		}
		log.LogAttrs(r.Context(), level, "http_request",
			slog.String("request_id", id), slog.String("tenant", r.Header.Get("X-Tenant-ID")),
			slog.String(labelRoute, rt), slog.Int("status", sw.status),
			slog.Float64("duration_ms", float64(elapsed.Microseconds())/1000))
	})
}
