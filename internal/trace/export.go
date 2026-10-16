package trace

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Tracer collects finished spans and exports them in batches to an OTLP/HTTP endpoint.
type Tracer struct {
	service  string
	endpoint string // e.g. http://jaeger:4318; empty disables export
	client   *http.Client
	mu       sync.Mutex
	batch    []*Span
	// Exported counts spans sent successfully (for tests and diagnostics).
	Exported int
}

// New creates a tracer for service. If endpoint is empty spans are created but not exported.
func New(service, endpoint string) *Tracer {
	return &Tracer{service: service, endpoint: endpoint, client: &http.Client{Timeout: 3 * time.Second}}
}

func (t *Tracer) export(s *Span) {
	if t.endpoint == "" {
		return
	}
	t.mu.Lock()
	t.batch = append(t.batch, s)
	t.mu.Unlock()
}

// Run flushes spans every interval until stop is closed.
func (t *Tracer) Run(interval time.Duration, stop <-chan struct{}) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			_ = t.Flush()
			return
		case <-tick.C:
			_ = t.Flush()
		}
	}
}

// Flush sends the pending batch.
func (t *Tracer) Flush() error {
	t.mu.Lock()
	spans := t.batch
	t.batch = nil
	t.mu.Unlock()
	if len(spans) == 0 {
		return nil
	}
	body, err := json.Marshal(t.payload(spans))
	if err != nil {
		return err
	}
	resp, err := t.client.Post(t.endpoint+"/v1/traces", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 300 {
		t.mu.Lock()
		t.Exported += len(spans)
		t.mu.Unlock()
	}
	return nil
}

type kv struct {
	Key   string            `json:"key"`
	Value map[string]string `json:"value"`
}

func attrs(m map[string]string) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{Key: k, Value: map[string]string{"stringValue": v}})
	}
	return out
}

// payload builds an OTLP ExportTraceServiceRequest in JSON encoding.
func (t *Tracer) payload(spans []*Span) map[string]any {
	items := make([]map[string]any, 0, len(spans))
	for _, s := range spans {
		status := 1
		if s.Error {
			status = 2
		}
		items = append(items, map[string]any{
			"traceId": s.TraceID, "spanId": s.SpanID, "parentSpanId": s.ParentID,
			"name": s.Name, "kind": s.Kind,
			"startTimeUnixNano": strconv.FormatInt(s.Start.UnixNano(), 10),
			"endTimeUnixNano":   strconv.FormatInt(s.End.UnixNano(), 10),
			"attributes":        attrs(s.Attrs),
			"status":            map[string]int{"code": status},
		})
	}
	return map[string]any{"resourceSpans": []any{map[string]any{
		"resource":   map[string]any{"attributes": attrs(map[string]string{"service.name": t.service})},
		"scopeSpans": []any{map[string]any{"scope": map[string]string{"name": "roomflow/trace"}, "spans": items}},
	}}}
}
