package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("ROOMFLOW_X", "")
	if envOr("ROOMFLOW_X", "def") != "def" {
		t.Fatal("default expected")
	}
	t.Setenv("ROOMFLOW_X", "v")
	if envOr("ROOMFLOW_X", "def") != "v" {
		t.Fatal("env value expected")
	}
}

func TestProbe(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }))
	defer bad.Close()
	if probe(ok.URL) != 0 || probe(bad.URL) != 1 || probe("http://127.0.0.1:1") != 1 {
		t.Fatal("probe returned unexpected exit codes")
	}
}

func call(t *testing.T, h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("X-Tenant-ID", "hotel-a")
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHandlerWiring(t *testing.T) {
	flagsFile := filepath.Join(t.TempDir(), "flags.json")
	_ = os.WriteFile(flagsFile, []byte(`{"price_breakdown":{"enabled":true}}`), 0o600)
	t.Setenv("FLAGS_FILE", flagsFile)
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("ADMIN_TOKEN", "s3cret")
	t.Setenv("COLOR", "blue")
	h, err := buildHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if w := call(t, h, "GET", "/api/flags", ""); !strings.Contains(w.Body.String(), `"price_breakdown":true`) {
		t.Fatalf("flags: %s", w.Body)
	}
	if w := call(t, h, "GET", "/version", ""); !strings.Contains(w.Body.String(), `"color":"blue"`) {
		t.Fatalf("version: %s", w.Body)
	}
	if call(t, h, "POST", "/admin/flags/reload", "").Code != 401 || call(t, h, "POST", "/admin/flags/reload", "Bearer s3cret").Code != 204 {
		t.Fatal("admin endpoint auth")
	}
	if w := call(t, h, "GET", "/metrics", ""); !strings.Contains(w.Body.String(), `roomflow_http_requests_total{route="GET /api/flags",code="200"} 1`) {
		t.Fatalf("metrics: %s", w.Body)
	}
	t.Setenv("FLAGS_FILE", filepath.Join(t.TempDir(), "missing.json"))
	if _, err := buildHandler(slog.New(slog.NewJSONHandler(io.Discard, nil))); err == nil {
		t.Fatal("missing flags file must fail startup")
	}
}

func TestVault(t *testing.T) {
	vault := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Vault-Token") {
		case "good":
			_, _ = w.Write([]byte(`{"data":{"data":{"admin_token":"from-vault"}}}`))
		case "empty":
			_, _ = w.Write([]byte(`{"data":{"data":{}}}`))
		case "garbage":
			_, _ = w.Write([]byte(`{`))
		default:
			w.WriteHeader(403)
		}
	}))
	defer vault.Close()
	t.Setenv("VAULT_ADDR", vault.URL)
	cases := map[string]bool{"good": true, "empty": false, "garbage": false, "bad": false}
	for token, ok := range cases {
		t.Setenv("VAULT_TOKEN", token)
		got, err := adminTokenFromEnvOrVault()
		if (err == nil) != ok || (ok && got != "from-vault") {
			t.Fatalf("token %s: got %q err %v", token, got, err)
		}
	}
	t.Setenv("VAULT_ADDR", "http://127.0.0.1:1")
	if _, err := adminTokenFromEnvOrVault(); err == nil {
		t.Fatal("unreachable vault must fail")
	}
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("ADMIN_TOKEN", "local")
	if got, _ := adminTokenFromEnvOrVault(); got != "local" {
		t.Fatal("env fallback")
	}
}

func TestRuntimeOptions(t *testing.T) {
	t.Setenv("FLAGS_FILE", "")
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "")
	t.Setenv("FAULT_RATE", "1")
	t.Setenv("CACHE_REDIS_ADDR", "127.0.0.1:1")
	t.Setenv("PPROF_ADDR", "127.0.0.1:0")
	servePprof(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	h, err := buildHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if got := call(t, h, "GET", "/api/rooms", "").Code; got != 500 {
		t.Fatalf("FAULT_RATE=1 must fail API calls, got %d", got)
	}
	if got := call(t, h, "GET", "/api/rooms", "").Code; got != 429 {
		t.Fatalf("second request over 1 rps must be limited, got %d", got)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil)) // scrapes carry no tenant and are not limited
	m := w.Body.String()
	if !strings.Contains(m, "roomflow_rate_limited_total") {
		t.Fatalf("rate limit metric missing: %s", m)
	}
}
