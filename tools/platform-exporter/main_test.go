package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudo97/roomflow/internal/obs"
)

func TestPlatformMetrics(t *testing.T) {
	p := &platform{reg: obs.NewRegistry()}
	body := `[{"type":"service_created","template":"roomflow-service","language":"go"},
		{"type":"provisioning","resource":"postgres","seconds":42},
		{"type":"request","channel":"self-service"},{"type":"request","channel":"ticket"},
		{"type":"survey","score":4},{"type":"survey","score":5}]`
	w := httptest.NewRecorder()
	p.handle(w, httptest.NewRequest("POST", "/events", strings.NewReader(body)))
	if w.Code != 202 {
		t.Fatalf("status %d", w.Code)
	}
	var b strings.Builder
	p.reg.Render(&b)
	for _, want := range []string{
		`platform_services_created_total{template="roomflow-service",language="go"} 1`,
		`platform_provisioning_duration_seconds_bucket{resource="postgres",le="60"} 1`,
		`platform_requests_total{channel="ticket"} 1`,
		`platform_developer_satisfaction_score 4.5`,
	} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("missing %q in\n%s", want, b.String())
		}
	}
	for _, bad := range []string{`[{"type":"nope"}]`, `{`} {
		w := httptest.NewRecorder()
		p.handle(w, httptest.NewRequest("POST", "/events", strings.NewReader(bad)))
		if w.Code != 400 {
			t.Fatalf("%s: status %d", bad, w.Code)
		}
	}
	f := filepath.Join(t.TempDir(), "h.json")
	_ = os.WriteFile(f, []byte(`[{"type":"survey","score":1}]`), 0o600)
	if err := p.replay(f); err != nil || p.answers != 3 {
		t.Fatalf("replay: %v", err)
	}
	if p.replay(filepath.Join(t.TempDir(), "missing")) == nil {
		t.Fatal("missing history must fail")
	}
	_ = os.WriteFile(f, []byte(`{`), 0o600)
	if p.replay(f) == nil {
		t.Fatal("broken history must fail")
	}
}
