package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// github reads the history of a GitHub Actions workflow.
type github struct {
	api, repo, token, workflow string
}

type ghRuns struct {
	WorkflowRuns []struct {
		Conclusion string    `json:"conclusion"`
		UpdatedAt  time.Time `json:"updated_at"`
		HeadCommit struct {
			Timestamp time.Time `json:"timestamp"`
		} `json:"head_commit"`
	} `json:"workflow_runs"`
}

func (g *github) runs() ([]run, error) {
	body, err := g.fetch()
	if err != nil {
		return nil, err
	}
	out := make([]run, 0, len(body.WorkflowRuns))
	for _, r := range body.WorkflowRuns {
		if r.Conclusion == "cancelled" || r.Conclusion == "skipped" {
			continue
		}
		out = append(out, run{CommitTime: r.HeadCommit.Timestamp, Finished: r.UpdatedAt, Success: r.Conclusion == "success"})
	}
	return out, nil
}

func (g *github) fetch() (ghRuns, error) {
	var body ghRuns
	url := fmt.Sprintf("%s/repos/%s/actions/workflows/%s/runs?branch=main&event=push&status=completed&per_page=100", g.api, g.repo, g.workflow)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return body, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return body, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("github: status %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	return body, err
}

func (d *dora) backfill(g *github) error {
	runs, err := g.runs()
	if err != nil {
		return err
	}
	d.replay(runs)
	return nil
}
