// Package obs provides observability primitives for RoomFlow: Prometheus-compatible metrics
// (text exposition format, implemented on the standard library) and structured request logging.
package obs

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// DefaultBuckets are latency buckets in seconds.
var DefaultBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}

// Registry stores metric families and renders them in Prometheus text format.
type Registry struct {
	mu       sync.Mutex
	families map[string]*family
	order    []string
}

type family struct {
	name, help, kind string
	buckets          []float64
	series           map[string]*series
}

type series struct {
	labels string
	value  float64
	counts []uint64 // histogram buckets
	sum    float64
	count  uint64
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{families: map[string]*family{}} }

func (r *Registry) family(name, help, kind string, buckets []float64) *family {
	f, ok := r.families[name]
	if !ok {
		f = &family{name: name, help: help, kind: kind, buckets: buckets, series: map[string]*series{}}
		r.families[name] = f
		r.order = append(r.order, name)
	}
	return f
}

func (f *family) get(labels string) *series {
	s, ok := f.series[labels]
	if !ok {
		s = &series{labels: labels, counts: make([]uint64, len(f.buckets))}
		f.series[labels] = s
	}
	return s
}

// Labels formats label pairs ("k1", "v1", "k2", "v2") as {k1="v1",k2="v2"}.
func Labels(kv ...string) string {
	if len(kv) == 0 {
		return ""
	}
	parts := make([]string, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		v := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(kv[i+1])
		parts = append(parts, kv[i]+`="`+v+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// Add increments a counter.
func (r *Registry) Add(name, help, labels string, delta float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.family(name, help, "counter", nil).get(labels).value += delta
}

// Set sets a gauge value.
func (r *Registry) Set(name, help, labels string, v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.family(name, help, "gauge", nil).get(labels).value = v
}

// Observe records a histogram observation with DefaultBuckets.
func (r *Registry) Observe(name, help, labels string, v float64) {
	r.ObserveBuckets(name, help, labels, v, DefaultBuckets)
}

// ObserveBuckets records a histogram observation; buckets are fixed when the family is first used.
func (r *Registry) ObserveBuckets(name, help, labels string, v float64, buckets []float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f := r.family(name, help, "histogram", buckets)
	s := f.get(labels)
	for i, b := range f.buckets {
		if v <= b {
			s.counts[i]++
		}
	}
	s.sum += v
	s.count++
}

// Value returns the current value of a counter or gauge (0 if absent); used in tests.
func (r *Registry) Value(name, labels string) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if f, ok := r.families[name]; ok {
		if s, ok := f.series[labels]; ok {
			return s.value
		}
	}
	return 0
}

// Render renders all metrics in Prometheus text exposition format.
func (r *Registry) Render(w io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, name := range r.order {
		f := r.families[name]
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", f.name, f.help, f.name, f.kind)
		keys := make([]string, 0, len(f.series))
		for k := range f.series {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			writeSeries(w, f, f.series[k])
		}
	}
}

func writeSeries(w io.Writer, f *family, s *series) {
	if f.kind != "histogram" {
		_, _ = fmt.Fprintf(w, "%s%s %s\n", f.name, s.labels, num(s.value))
		return
	}
	for i, b := range f.buckets {
		_, _ = fmt.Fprintf(w, "%s_bucket%s %d\n", f.name, withLabel(s.labels, "le", num(b)), s.counts[i])
	}
	_, _ = fmt.Fprintf(w, "%s_bucket%s %d\n", f.name, withLabel(s.labels, "le", "+Inf"), s.count)
	_, _ = fmt.Fprintf(w, "%s_sum%s %s\n%s_count%s %d\n", f.name, s.labels, num(s.sum), f.name, s.labels, s.count)
}

func withLabel(labels, k, v string) string {
	pair := fmt.Sprintf("%s=%q", k, v)
	if labels == "" {
		return "{" + pair + "}"
	}
	return labels[:len(labels)-1] + "," + pair + "}"
}

func num(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}

// Handler serves the registry at /metrics.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		r.Render(w)
	})
}
