package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tmpl := "name: ${{ values.name }}\n{%- if values.db == \"pg\" %}\ndb: yes\n{%- else %}\ndb: no\n{%- endif %}\nend"
	got, err := render(tmpl, map[string]string{"name": "svc", "db": "pg"})
	if err != nil || got != "name: svc\ndb: yes\nend" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, _ = render(tmpl, map[string]string{"name": "svc", "db": "none"})
	if got != "name: svc\ndb: no\nend" {
		t.Fatalf("else branch: %q", got)
	}
	nested := "{% if values.a != \"x\" %}\n{% if values.b == \"y\" %}\nboth\n{% endif %}\nouter\n{% endif %}"
	if got, _ := render(nested, map[string]string{"a": "z", "b": "n"}); got != "outer" {
		t.Fatalf("nested: %q", got)
	}
	for _, bad := range []string{"${{ values.nope }}", "{% endif %}", "{% else %}", "{% if values.a == \"b\" %}"} {
		if _, err := render(bad, map[string]string{"a": "b"}); err == nil {
			t.Fatalf("%q must fail", bad)
		}
	}
}

func TestNewServiceFromSkeleton(t *testing.T) {
	out := filepath.Join(t.TempDir(), "svc")
	var stdout, stderr bytes.Buffer
	code := run([]string{"new", "--name", "guest-reviews", "--lang", "go", "--db", "postgres", "--out", out,
		"--template", "../../platform/templates/service/skeleton", "-o", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	var res message
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil || res.Status != "created" {
		t.Fatalf("json output: %s", stdout.String())
	}
	deploy, _ := os.ReadFile(filepath.Join(out, "k8s", "deployment.yaml"))
	if !strings.Contains(string(deploy), "kind: PostgreSQLInstance") || strings.Contains(string(deploy), "REDIS_ADDR") || strings.Contains(string(deploy), "${{") {
		t.Fatalf("deployment:\n%s", deploy)
	}
	if _, err := os.Stat(filepath.Join(out, "main.go")); err != nil {
		t.Fatal("go skeleton missing")
	}
	if code := run([]string{"catalog", "validate", filepath.Join(out, "catalog-info.yaml")}, &stdout, &stderr); code != 0 {
		t.Fatalf("generated catalog must be valid: %s", stderr.String())
	}
	for _, bad := range [][]string{{"new", "--name", "X"}, {"new", "--name", "abc", "--lang", "rust"}, {"new", "--name", "abc", "--db", "mongo"}, {"nope"}, {}} {
		if run(bad, &stdout, &stderr) == 0 {
			t.Fatalf("%v must fail", bad)
		}
	}
}

func TestProvisionCatalogLoginStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if run([]string{"provision", "--name", "reviews-db", "--size", "medium", "--namespace", "team-a"}, &stdout, &stderr) != 0 ||
		!strings.Contains(stdout.String(), "storageGB: 5") || !strings.Contains(stdout.String(), "namespace: team-a") {
		t.Fatalf("provision: %s %s", stdout.String(), stderr.String())
	}
	for _, bad := range [][]string{{"provision", "--kind", "mysql", "--name", "abc"}, {"provision", "--name", "abc", "--size", "huge"}, {"provision", "--name", "A"}} {
		if run(bad, &stdout, &stderr) == 0 {
			t.Fatalf("%v must fail", bad)
		}
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(bad, []byte("kind: Component\nmetadata:\n  name: x\n---\nkind: API\nmetadata:\n  name: y\n"), 0o600)
	stdout.Reset()
	if run([]string{"catalog", "validate", bad, "-o", "json"}, &stdout, &stderr) == 0 {
		t.Fatal("invalid catalog must fail")
	}
	if run([]string{"catalog", "validate", "../../catalog-info.yaml"}, &stdout, &stderr) != 0 {
		t.Fatalf("repository catalog must be valid: %s", stderr.String())
	}
	if run([]string{"catalog"}, &stdout, &stderr) == 0 {
		t.Fatal("catalog without subcommand")
	}

	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("GITHUB_TOKEN", "")
	if run([]string{"login", "--token", "secret-token"}, &stdout, &stderr) != 0 || token() != "secret-token" {
		t.Fatal("login must persist the token")
	}
	if run([]string{"login"}, &stdout, &stderr) == 0 {
		t.Fatal("login without token")
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"workflow_runs":[{"name":"ci","status":"completed","conclusion":"success","head_sha":"abcdef123","updated_at":"2026-10-16T10:00:00Z"},{"name":"ci","status":"completed","conclusion":"failure","head_sha":"000","updated_at":"2026-10-16T09:00:00Z"}]}`))
	}))
	defer api.Close()
	githubAPI = api.URL
	stdout.Reset()
	if run([]string{"status", "-o", "json"}, &stdout, &stderr) != 0 || strings.Count(stdout.String(), `"workflow"`) != 1 || !strings.Contains(stdout.String(), "abcdef1") {
		t.Fatalf("status: %s %s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	if run([]string{"status"}, &stdout, &stderr) != 0 || !strings.Contains(stdout.String(), "WORKFLOW") {
		t.Fatalf("status text: %s", stdout.String())
	}
	t.Setenv("GITHUB_TOKEN", "wrong")
	if run([]string{"status"}, &stdout, &stderr) == 0 {
		t.Fatal("unauthorized status must fail")
	}
}
