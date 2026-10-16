package flags

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRules(t *testing.T) {
	s := New(map[string]Rule{
		"global": {Enabled: true},
		"listed": {Tenants: []string{"hotel-a"}},
		"half":   {Percent: 50},
		"nobody": {},
	})
	if !s.Enabled("global", "x") || !s.Enabled("listed", "hotel-a") || s.Enabled("listed", "hotel-b") || s.Enabled("nobody", "x") || s.Enabled("missing", "x") {
		t.Fatal("unexpected evaluation")
	}
	on := 0
	for i := 0; i < 1000; i++ {
		if s.Enabled("half", fmt.Sprintf("t%d", i)) {
			on++
		}
	}
	if on < 400 || on > 600 {
		t.Fatalf("percentage rollout is skewed: %d/1000", on)
	}
	if s.Enabled("half", "t1") != s.Enabled("half", "t1") {
		t.Fatal("rollout must be stable per tenant")
	}
	if all := s.All("hotel-a"); !all["global"] || !all["listed"] || all["nobody"] {
		t.Fatalf("All = %v", all)
	}
	if New(nil).Enabled("x", "y") {
		t.Fatal("empty store must have no flags")
	}
}

func TestReloadFromFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "flags.json")
	write := func(s string, mod time.Time) {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
		_ = os.Chtimes(p, mod, mod)
	}
	write(`{"f":{"enabled":false}}`, time.Now().Add(-time.Hour))
	s, err := FromFile(p)
	if err != nil || s.Enabled("f", "a") {
		t.Fatalf("err=%v", err)
	}
	write(`{"f":{"enabled":true}}`, time.Now())
	stop := make(chan struct{})
	go s.Watch(10*time.Millisecond, stop)
	deadline := time.Now().Add(2 * time.Second)
	for !s.Enabled("f", "a") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	close(stop)
	if !s.Enabled("f", "a") {
		t.Fatal("flag change was not picked up without restart")
	}
	write(`not json`, time.Now().Add(time.Minute))
	if err := s.Reload(); err == nil || !s.Enabled("f", "a") {
		t.Fatal("broken file must keep the last good configuration")
	}
	if _, err := FromFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing file must be an error")
	}
	if New(nil).Reload() != nil {
		t.Fatal("reload without file must be a no-op")
	}
}
