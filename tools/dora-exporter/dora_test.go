package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sudo97/roomflow/internal/obs"
)

func scrape(reg *obs.Registry) string {
	var b strings.Builder
	reg.Render(&b)
	return b.String()
}

func TestReplayComputesDORA(t *testing.T) {
	reg := obs.NewRegistry()
	d := newDORA(reg, "staging")
	t0 := time.Now().Add(-time.Hour)
	d.replay([]run{
		{CommitTime: t0, Finished: t0.Add(5 * time.Minute), Success: true},
		{CommitTime: t0.Add(10 * time.Minute), Finished: t0.Add(12 * time.Minute), Success: false},
		{CommitTime: t0.Add(20 * time.Minute), Finished: t0.Add(30 * time.Minute), Success: true},
	})
	out := scrape(reg)
	for _, want := range []string{
		`dora_deployments_total{environment="staging"} 2`,
		`dora_failed_deployments_total{environment="staging"} 1`,
		`dora_restore_seconds_sum{environment="staging"} 1080`,
		`dora_lead_time_seconds_bucket{environment="staging",le="300"} 1`,
		`dora_lead_time_seconds_count{environment="staging"} 2`,
		`dora_open_incidents{environment="staging"} 0`,
		`dora_history_runs{environment="staging"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
}

func TestEventsAndAlertmanagerWebhook(t *testing.T) {
	reg := obs.NewRegistry()
	d := newDORA(reg, "ci")
	post := func(h http.HandlerFunc, body string) int {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		return w.Code
	}
	now := time.Now().Unix()
	if post(d.handleEvent, `{"type":"deployment","commit_time":`+strconv.FormatInt(now-600, 10)+`}`) != 202 {
		t.Fatal("deployment event")
	}
	firing := `{"alerts":[{"status":"firing","labels":{"alertname":"HighErrorRate","severity":"critical"},"startsAt":"2026-10-16T10:00:00Z"},{"status":"firing","labels":{"alertname":"Slow","severity":"warning"}}]}`
	resolved := `{"alerts":[{"status":"resolved","labels":{"alertname":"HighErrorRate","severity":"critical"},"endsAt":"2026-10-16T10:04:00Z"}]}`
	if post(d.handleAlertmanager, firing) != 200 || !strings.Contains(scrape(reg), `dora_open_incidents{environment="ci"} 1`) {
		t.Fatal("firing alert must open an incident")
	}
	post(d.handleAlertmanager, firing) // repeated notification must not double count
	post(d.handleAlertmanager, resolved)
	out := scrape(reg)
	for _, want := range []string{`dora_failed_deployments_total{environment="ci"} 1`, `dora_restore_seconds_sum{environment="ci"} 240`, `dora_open_incidents{environment="ci"} 0`, `dora_lead_time_seconds_sum{environment="ci"} 600`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
	if post(d.handleEvent, `{"type":"incident","key":"manual","firing":true}`) != 202 || post(d.handleEvent, `{"type":"nope"}`) != 400 || post(d.handleEvent, `{`) != 400 || post(d.handleAlertmanager, `{`) != 400 {
		t.Fatal("event validation")
	}
	if d.windowStats(time.Now(), 24*time.Hour) != 1 {
		t.Fatal("deployment frequency per day")
	}
}

func TestGitHubHistory(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || !strings.Contains(r.URL.Path, "/repos/o/r/actions/workflows/ci.yml/runs") {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"workflow_runs":[
			{"conclusion":"success","updated_at":"2026-10-16T10:10:00Z","head_commit":{"timestamp":"2026-10-16T10:00:00Z"}},
			{"conclusion":"cancelled","updated_at":"2026-10-16T10:20:00Z","head_commit":{"timestamp":"2026-10-16T10:15:00Z"}},
			{"conclusion":"failure","updated_at":"2026-10-16T10:30:00Z","head_commit":{"timestamp":"2026-10-16T10:25:00Z"}}]}`))
	}))
	defer api.Close()
	reg := obs.NewRegistry()
	d := newDORA(reg, "staging")
	if err := d.backfill(&github{api: api.URL, repo: "o/r", token: "tok", workflow: "ci.yml"}); err != nil {
		t.Fatal(err)
	}
	if out := scrape(reg); !strings.Contains(out, `dora_history_runs{environment="staging"} 2`) || !strings.Contains(out, `dora_open_incidents{environment="staging"} 1`) {
		t.Fatalf("history:\n%s", out)
	}
	if err := d.backfill(&github{api: api.URL, repo: "o/r", token: "bad", workflow: "ci.yml"}); err == nil {
		t.Fatal("unauthorized must fail")
	}
	if envOr("DORA_NOPE", "x") != "x" {
		t.Fatal("envOr")
	}
}
