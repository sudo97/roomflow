// Package chaos injects faults on purpose. It is used to verify that progressive delivery
// (canary analysis) and resilience mechanisms detect a broken release; it is off by default.
package chaos

import (
	"math/rand/v2"
	"net/http"
	"strings"
)

// Faults wraps next so that a fraction rate (0..1) of /api requests fails with 500.
func Faults(rate float64, next http.Handler) http.Handler {
	if rate <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && rand.Float64() < rate { //nolint:gosec // not security sensitive
			http.Error(w, "injected fault", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
