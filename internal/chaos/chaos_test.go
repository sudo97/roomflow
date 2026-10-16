package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFaults(t *testing.T) {
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	count := func(h http.Handler, path string) (fails int) {
		for i := 0; i < 1000; i++ {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code == 500 {
				fails++
			}
		}
		return fails
	}
	if count(Faults(0, ok), "/api/rooms") != 0 || count(Faults(1, ok), "/healthz") != 0 {
		t.Fatal("faults must be off by default and never hit probes")
	}
	if n := count(Faults(0.3, ok), "/api/rooms"); n < 200 || n > 400 {
		t.Fatalf("expected ~30%% failures, got %d/1000", n)
	}
}
