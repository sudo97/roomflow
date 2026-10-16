package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sudo97/roomflow/internal/obs"
	"github.com/sudo97/roomflow/internal/trace"
)

func TestFrontendProxiesWithTraceContext(t *testing.T) {
	var gotParent, gotTenant, gotHost string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParent, gotTenant, gotHost = r.Header.Get("traceparent"), r.Header.Get("X-Tenant-ID"), r.Host
		_, _ = w.Write([]byte(`[]`))
	}))
	defer backend.Close()
	u, _ := url.Parse(backend.URL)
	h := newHandler(u, info{"v2", "green"}, trace.New("frontend", ""), obs.NewRegistry(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("X-Tenant-ID", "demo-hotel")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := get("/api/rooms"); w.Body.String() != "[]" || gotTenant != "demo-hotel" || !strings.HasPrefix(gotParent, "00-") {
		t.Fatalf("proxy: body %q tenant %q traceparent %q", w.Body, gotTenant, gotParent)
	}
	// Host має бути адресою backend: за ним sidecar service mesh обирає маршрут (ЛР12)
	if gotHost != u.Host {
		t.Fatalf("proxy Host = %q, want backend host %q", gotHost, u.Host)
	}
	if w := get("/"); !strings.Contains(w.Body.String(), "frontend v2 (green)") {
		t.Fatalf("page: %s", w.Body)
	}
	if w := get("/version"); !strings.Contains(w.Body.String(), `"color":"green"`) {
		t.Fatalf("version: %s", w.Body)
	}
	if w := get("/healthz"); w.Code != 200 {
		t.Fatal("healthz")
	}
	if w := get("/metrics"); !strings.Contains(w.Body.String(), `route="/api/"`) {
		t.Fatalf("metrics: %s", w.Body)
	}
	if envOr("ROOMFLOW_NOPE", "d") != "d" {
		t.Fatal("envOr")
	}
}
