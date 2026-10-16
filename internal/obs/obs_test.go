package obs

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExposition(t *testing.T) {
	r := NewRegistry()
	r.Add("x_total", "help", Labels("a", `q"uote`), 2)
	r.Set("g", "gauge", "", 1.5)
	r.Observe("h_seconds", "hist", Labels("route", "r"), 0.003)
	r.Observe("h_seconds", "hist", Labels("route", "r"), 2)
	var b bytes.Buffer
	r.Render(&b)
	out := b.String()
	for _, want := range []string{
		"# TYPE x_total counter", `x_total{a="q\"uote"} 2`, "g 1.5",
		`h_seconds_bucket{route="r",le="0.005"} 1`, `h_seconds_bucket{route="r",le="+Inf"} 2`,
		`h_seconds_count{route="r"} 2`, `h_seconds_sum{route="r"} 2.003`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
	if r.Value("x_total", Labels("a", `q"uote`)) != 2 || r.Value("nope", "") != 0 {
		t.Fatal("Value mismatch")
	}
}

func TestInstrument(t *testing.T) {
	reg := NewRegistry()
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	h := Instrument(reg, log, func(r *http.Request) string { return r.Method + " /x" },
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("fail") != "" {
				w.WriteHeader(500)
			}
		}))
	for _, u := range []string{"/x", "/x?fail=1"} {
		req := httptest.NewRequest("GET", u, nil)
		req.Header.Set("X-Tenant-ID", "hotel-a")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Header().Get("X-Request-ID") == "" {
			t.Fatal("request id header missing")
		}
	}
	if reg.Value("roomflow_http_requests_total", Labels("route", "GET /x", "code", "500")) != 1 {
		t.Fatal("5xx not counted")
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) || !strings.Contains(logs.String(), `"tenant":"hotel-a"`) {
		t.Fatalf("unexpected logs: %s", logs.String())
	}
	rec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(rec.Body.String(), "roomflow_http_request_duration_seconds_bucket") {
		t.Fatal("histogram not exposed")
	}
}
