// Package ratelimit protects the API from overload with a token bucket per tenant:
// one noisy hotel cannot exhaust capacity shared by all tenants of the SaaS.
package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Limiter holds one token bucket per key.
type Limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New creates a limiter that allows rate requests per second with the given burst per key.
func New(rate float64, burst int) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), buckets: map[string]*bucket{}, now: time.Now}
}

// Allow takes a token for key; when the bucket is empty it returns false and the time to wait.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

// Middleware rejects requests over the limit with 429 Too Many Requests and a Retry-After header.
// key extracts the limited identity (tenant) from the request; empty keys are not limited.
func (l *Limiter) Middleware(key func(*http.Request) string, onReject func(), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := key(r)
		if k != "" {
			if ok, wait := l.Allow(k); !ok {
				onReject()
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
