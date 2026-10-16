// Command platform-exporter exposes Platform Engineering metrics (lab 11): service creation rate,
// infrastructure provisioning duration, self-service adoption and developer satisfaction.
// Events are posted by the platform tooling (rfctl, CI jobs, survey form) to POST /events;
// an optional JSON history file is replayed at start.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sudo97/roomflow/internal/obs"
)

var durationBuckets = []float64{10, 30, 60, 120, 300, 600, 1800}

// event is one platform interaction.
type event struct {
	Type     string  `json:"type"`     // service_created | provisioning | request | survey
	Template string  `json:"template"` // service_created
	Language string  `json:"language"` // service_created
	Resource string  `json:"resource"` // provisioning
	Seconds  float64 `json:"seconds"`  // provisioning
	Channel  string  `json:"channel"`  // request: self-service | ticket
	Score    float64 `json:"score"`    // survey: 1..5
}

type platform struct {
	mu      sync.Mutex
	reg     *obs.Registry
	scores  float64
	answers int
}

func (p *platform) record(e event) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch e.Type {
	case "service_created":
		p.reg.Add("platform_services_created_total", "Services created from golden-path templates.", obs.Labels("template", e.Template, "language", e.Language), 1)
	case "provisioning":
		p.reg.ObserveBuckets("platform_provisioning_duration_seconds", "Time from request to ready infrastructure.", obs.Labels("resource", e.Resource), e.Seconds, durationBuckets)
	case "request":
		p.reg.Add("platform_requests_total", "Infrastructure and service requests by channel.", obs.Labels("channel", e.Channel), 1)
	case "survey":
		p.scores += e.Score
		p.answers++
		p.reg.Set("platform_developer_satisfaction_score", "Average developer satisfaction (1-5).", "", p.scores/float64(p.answers))
		p.reg.Set("platform_survey_responses", "Developer survey responses.", "", float64(p.answers))
	default:
		return false
	}
	return true
}

func (p *platform) handle(w http.ResponseWriter, r *http.Request) {
	var events []event
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, e := range events {
		if !p.record(e) {
			http.Error(w, "unknown event type "+e.Type, http.StatusBadRequest)
			return
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (p *platform) replay(path string) error {
	raw, err := os.ReadFile(path) //nolint:gosec // history file configured by the operator
	if err != nil {
		return err
	}
	var events []event
	if err := json.Unmarshal(raw, &events); err != nil {
		return err
	}
	for _, e := range events {
		p.record(e)
	}
	return nil
}

func main() {
	p := &platform{reg: obs.NewRegistry()}
	if path := os.Getenv("HISTORY_FILE"); path != "" {
		go func() {
			time.Sleep(15 * time.Second) // let Prometheus scrape the zero state first
			if err := p.replay(path); err != nil {
				log.Printf("history: %v", err)
			}
		}()
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", p.reg.Handler())
	mux.HandleFunc("POST /events", p.handle)
	srv := &http.Server{Addr: ":9103", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
