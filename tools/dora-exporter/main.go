// Command dora-exporter exposes DORA metrics (deployment frequency, lead time for changes,
// change failure rate, time to restore service) in Prometheus format.
//
// Data sources:
//   - GitHub Actions history of the CI/CD workflow (backfilled once at start);
//   - deployment events posted by the pipeline: POST /events {"type":"deployment","commit_time":...};
//   - Alertmanager webhooks: a firing critical alert opens an incident (the last deployment is
//     counted as failed), its resolution closes the incident and records the time to restore.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sudo97/roomflow/internal/obs"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	reg := obs.NewRegistry()
	d := newDORA(reg, envOr("DORA_ENVIRONMENT", "staging"))
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		delay, _ := time.ParseDuration(envOr("HISTORY_DELAY", "20s"))
		gh := &github{api: envOr("GITHUB_API_URL", "https://api.github.com"), repo: repo, token: os.Getenv("GITHUB_TOKEN"), workflow: envOr("DORA_WORKFLOW", "ci.yml")}
		go func() {
			// Counters start at zero so that Prometheus observes the backfill as an increase.
			time.Sleep(delay)
			if err := d.backfill(gh); err != nil {
				log.Printf("history backfill failed: %v", err)
			}
		}()
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", reg.Handler())
	mux.HandleFunc("POST /events", d.handleEvent)
	mux.HandleFunc("POST /alertmanager", d.handleAlertmanager)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	srv := &http.Server{Addr: envOr("ADDR", ":9102"), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("dora-exporter listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
