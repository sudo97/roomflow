package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucketPerKey(t *testing.T) {
	l := New(2, 3)
	now := time.Unix(0, 0)
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("burst request %d rejected", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait != 500*time.Millisecond {
		t.Fatalf("4th request: ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other tenants must not be affected")
	}
	now = now.Add(time.Second) // +2 tokens
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatal("bucket must refill")
		}
	}
	now = now.Add(time.Hour)
	l.Allow("a")
	if b := l.buckets["a"]; b.tokens != 2 {
		t.Fatalf("tokens must be capped by burst, got %v", b.tokens)
	}
}

func TestMiddleware(t *testing.T) {
	rejected := 0
	h := New(1, 1).Middleware(func(r *http.Request) string { return r.Header.Get("X-Tenant-ID") }, func() { rejected++ },
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	codes := []int{}
	for _, tenant := range []string{"a", "a", "", ""} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Tenant-ID", tenant)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		codes = append(codes, w.Code)
		if w.Code == 429 && w.Header().Get("Retry-After") != "1" {
			t.Fatalf("Retry-After = %q", w.Header().Get("Retry-After"))
		}
	}
	if codes[0] != 200 || codes[1] != 429 || codes[2] != 200 || codes[3] != 200 || rejected != 1 {
		t.Fatalf("codes %v rejected %d", codes, rejected)
	}
}
