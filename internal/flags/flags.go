// Package flags implements runtime feature flags for RoomFlow: flags are read from a JSON file
// and re-read periodically, so behaviour can be changed without restarting the service.
// A flag can be enabled globally, for a list of tenants, or for a percentage of tenants.
package flags

import (
	"encoding/json"
	"hash/fnv"
	"os"
	"sync"
	"time"
)

// Rule describes when a flag is on.
type Rule struct {
	Enabled bool     `json:"enabled"`
	Tenants []string `json:"tenants,omitempty"` // always on for these tenants
	Percent int      `json:"percent,omitempty"` // share of tenants (0-100), stable per tenant
}

// Store holds the current flag rules.
type Store struct {
	mu    sync.RWMutex
	rules map[string]Rule
	path  string
	mtime time.Time
}

// New creates a store with the given initial rules (may be nil).
func New(rules map[string]Rule) *Store {
	if rules == nil {
		rules = map[string]Rule{}
	}
	return &Store{rules: rules}
}

// FromFile loads rules from path; Reload re-reads it when it changes.
func FromFile(path string) (*Store, error) {
	s := &Store{rules: map[string]Rule{}, path: path}
	return s, s.Reload()
}

// Reload re-reads the file if its modification time changed.
func (s *Store) Reload() error {
	if s.path == "" {
		return nil
	}
	st, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if !st.ModTime().After(s.mtime) {
		return nil
	}
	raw, err := os.ReadFile(s.path) //nolint:gosec // path comes from operator configuration (FLAGS_FILE)
	if err != nil {
		return err
	}
	rules := map[string]Rule{}
	if err := json.Unmarshal(raw, &rules); err != nil {
		return err
	}
	s.mu.Lock()
	s.rules, s.mtime = rules, st.ModTime()
	s.mu.Unlock()
	return nil
}

// Watch reloads the file every interval until stop is closed.
func (s *Store) Watch(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			_ = s.Reload() // keep the last good configuration on errors
		}
	}
}

// Enabled reports whether flag name is on for tenant.
func (s *Store) Enabled(name, tenant string) bool {
	s.mu.RLock()
	r, ok := s.rules[name]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if r.Enabled {
		return true
	}
	for _, t := range r.Tenants {
		if t == tenant {
			return true
		}
	}
	return r.Percent > 0 && bucket(name, tenant) < r.Percent
}

// All evaluates every flag for tenant.
func (s *Store) All(tenant string) map[string]bool {
	s.mu.RLock()
	names := make([]string, 0, len(s.rules))
	for n := range s.rules {
		names = append(names, n)
	}
	s.mu.RUnlock()
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = s.Enabled(n, tenant)
	}
	return out
}

// bucket maps (flag, tenant) to a stable number in [0, 100).
func bucket(name, tenant string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name + "/" + tenant))
	return int(h.Sum32() % 100)
}
