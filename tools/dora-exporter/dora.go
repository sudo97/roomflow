package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sudo97/roomflow/internal/obs"
)

// leadTimeBuckets and restoreBuckets are histogram buckets in seconds (1 min .. 1 week).
// Metric names and descriptions.
const (
	mDeployments   = "dora_deployments_total"
	hDeployments   = "Deployments to the environment."
	mFailed        = "dora_failed_deployments_total"
	hFailed        = "Deployments that caused an incident (change failures)."
	mOpenIncidents = "dora_open_incidents"
	hOpenIncidents = "Incidents that are not restored yet."
)

var leadTimeBuckets = []float64{60, 300, 900, 1800, 3600, 4 * 3600, 24 * 3600, 7 * 24 * 3600}

type dora struct {
	mu        sync.Mutex
	reg       *obs.Registry
	env       string
	openSince map[string]time.Time // alertname -> incident start
	deploys   []time.Time          // deployment times for the rolling window gauge
}

func newDORA(reg *obs.Registry, env string) *dora {
	d := &dora{reg: reg, env: env, openSince: map[string]time.Time{}}
	labels := obs.Labels("environment", env)
	reg.Add(mDeployments, hDeployments, labels, 0)
	reg.Add(mFailed, hFailed, labels, 0)
	reg.Set(mOpenIncidents, hOpenIncidents, labels, 0)
	return d
}

func (d *dora) labels() string { return obs.Labels("environment", d.env) }

// observe records a value in a histogram with DORA buckets.
func (d *dora) observe(name, help string, seconds float64) {
	d.reg.ObserveBuckets(name, help, d.labels(), seconds, leadTimeBuckets)
}

// deployment records a deployment at deployedAt of a change committed at committedAt.
func (d *dora) deployment(committedAt, deployedAt time.Time, failed bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reg.Add(mDeployments, hDeployments, d.labels(), 1)
	if failed {
		d.reg.Add(mFailed, hFailed, d.labels(), 1)
	}
	if !committedAt.IsZero() && deployedAt.After(committedAt) {
		d.observe("dora_lead_time_seconds", "Lead time for changes: commit to deployment.", deployedAt.Sub(committedAt).Seconds())
	}
	d.reg.Set("dora_last_deployment_timestamp_seconds", "Time of the last deployment.", d.labels(), float64(deployedAt.Unix()))
	d.deploys = append(d.deploys, deployedAt)
}

// incident opens or resolves an incident identified by key.
func (d *dora) incident(key string, firing bool, at time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	start, open := d.openSince[key]
	switch {
	case firing && !open:
		d.openSince[key] = at
		// The change that is live now caused the failure.
		d.reg.Add(mFailed, hFailed, d.labels(), 1)
	case !firing && open:
		delete(d.openSince, key)
		d.observe("dora_restore_seconds", "Time to restore service after an incident.", at.Sub(start).Seconds())
	}
	d.reg.Set(mOpenIncidents, hOpenIncidents, d.labels(), float64(len(d.openSince)))
}

type event struct {
	Type       string `json:"type"`        // deployment | incident
	CommitTime int64  `json:"commit_time"` // unix seconds
	Time       int64  `json:"time"`        // unix seconds, default now
	Failed     bool   `json:"failed"`
	Key        string `json:"key"`
	Firing     bool   `json:"firing"`
}

func (d *dora) handleEvent(w http.ResponseWriter, r *http.Request) {
	var e event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	at := time.Now()
	if e.Time > 0 {
		at = time.Unix(e.Time, 0)
	}
	switch e.Type {
	case "deployment":
		var commit time.Time
		if e.CommitTime > 0 {
			commit = time.Unix(e.CommitTime, 0)
		}
		d.deployment(commit, at, e.Failed)
	case "incident":
		d.incident(e.Key, e.Firing, at)
	default:
		http.Error(w, "unknown event type", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type alertmanagerPayload struct {
	Alerts []struct {
		Status   string            `json:"status"`
		Labels   map[string]string `json:"labels"`
		StartsAt time.Time         `json:"startsAt"`
		EndsAt   time.Time         `json:"endsAt"`
	} `json:"alerts"`
}

// handleAlertmanager turns critical alerts into incidents (Alertmanager webhook receiver).
func (d *dora) handleAlertmanager(w http.ResponseWriter, r *http.Request) {
	var p alertmanagerPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, a := range p.Alerts {
		if a.Labels["severity"] != "critical" {
			continue
		}
		if a.Status == "firing" {
			d.incident(a.Labels["alertname"], true, a.StartsAt)
		} else {
			d.incident(a.Labels["alertname"], false, a.EndsAt)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// windowStats returns deployment frequency per day over the last window.
func (d *dora) windowStats(now time.Time, window time.Duration) float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, t := range d.deploys {
		if now.Sub(t) <= window {
			n++
		}
	}
	return float64(n) / window.Hours() * 24
}

// run is a completed CI/CD run (one deployment attempt).
type run struct {
	CommitTime, Finished time.Time
	Success              bool
}

// replay feeds historical runs in chronological order: successful runs are deployments; a failed run
// opens an incident (the main branch is broken) that is restored by the next successful run.
func (d *dora) replay(runs []run) {
	sort.Slice(runs, func(i, j int) bool { return runs[i].Finished.Before(runs[j].Finished) })
	for _, r := range runs {
		if r.Success {
			d.deployment(r.CommitTime, r.Finished, false)
			d.incident("pipeline", false, r.Finished)
		} else {
			d.incident("pipeline", true, r.Finished)
		}
	}
	d.reg.Set("dora_history_runs", "CI/CD runs loaded from history.", d.labels(), float64(len(runs)))
	d.reg.Set("dora_deployment_frequency_per_day", "Deployments per day over the last 7 days (from history).", d.labels(), d.windowStats(time.Now(), 7*24*time.Hour))
}
