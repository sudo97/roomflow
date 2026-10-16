package trace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPropagationAndExport(t *testing.T) {
	var mu sync.Mutex
	var got map[string]any
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		_ = json.Unmarshal(b, &got)
		mu.Unlock()
	}))
	defer collector.Close()
	tr := New("roomflow", collector.URL)

	var outgoing string
	h := tr.Middleware(func(*http.Request) string { return "GET /x" }, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := Start(r.Context(), "work")
		span.Set("k", "v")
		hdr := http.Header{}
		Inject(ctx, hdr)
		outgoing = hdr.Get("traceparent")
		span.Finish()
		w.WriteHeader(503)
	}))
	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if !strings.HasPrefix(outgoing, "00-0af7651916cd43dd8448eb211c80319c-") || strings.Contains(outgoing, "b7ad6b7169203331") {
		t.Fatalf("outgoing traceparent must continue the trace with a new parent: %s", outgoing)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { tr.Run(time.Hour, stop); close(done) }()
	close(stop)
	<-done
	if tr.Exported != 2 {
		t.Fatalf("exported %d spans", tr.Exported)
	}
	mu.Lock()
	defer mu.Unlock()
	spans := got["resourceSpans"].([]any)[0].(map[string]any)["scopeSpans"].([]any)[0].(map[string]any)["spans"].([]any)
	server := spans[1].(map[string]any)
	if server["parentSpanId"] != "b7ad6b7169203331" || server["kind"].(float64) != KindServer || server["status"].(map[string]any)["code"].(float64) != 2 {
		t.Fatalf("server span: %v", server)
	}
}

func TestNoopAndParsing(t *testing.T) {
	_, s := Start(context.Background(), "detached")
	s.Finish() // must not panic without a tracer
	Inject(context.Background(), http.Header{})
	for _, bad := range []string{"", "00-xyz-abc-01", "00-zzf7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"} {
		if _, _, ok := parseTraceparent(bad); ok {
			t.Fatalf("%q must be rejected", bad)
		}
	}
	tr := New("svc", "")
	_, sp := tr.Start(context.Background(), "x", KindInternal)
	sp.Finish()
	if tr.Flush() != nil {
		t.Fatal("disabled exporter")
	}
	broken := New("svc", "http://127.0.0.1:1")
	_, sp = broken.Start(context.Background(), "x", KindClient)
	sp.Finish()
	if broken.Flush() == nil {
		t.Fatal("unreachable collector must return an error")
	}
}
