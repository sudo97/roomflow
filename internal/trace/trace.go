// Package trace is a small distributed-tracing library compatible with W3C Trace Context and
// OpenTelemetry (spans are exported as OTLP/HTTP JSON, e.g. to Jaeger or an OTel Collector).
// It is stdlib-only on purpose: the service has no third-party dependencies.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Span kinds (OTLP numbering).
const (
	KindInternal = 1
	KindServer   = 2
	KindClient   = 3
)

// Span is a timed operation within a trace.
type Span struct {
	TraceID, SpanID, ParentID string
	Name                      string
	Kind                      int
	Start, End                time.Time
	Attrs                     map[string]string
	Error                     bool
	tracer                    *Tracer
}

type ctxKey struct{}

// FromContext returns the current span (nil if none).
func FromContext(ctx context.Context) *Span {
	s, _ := ctx.Value(ctxKey{}).(*Span)
	return s
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Start begins a child span of the span in ctx (or a new trace) and returns the derived context.
func (t *Tracer) Start(ctx context.Context, name string, kind int) (context.Context, *Span) {
	s := &Span{SpanID: randomHex(8), Name: name, Kind: kind, Start: time.Now(), Attrs: map[string]string{}, tracer: t}
	if parent := FromContext(ctx); parent != nil {
		s.TraceID, s.ParentID = parent.TraceID, parent.SpanID
	} else {
		s.TraceID = randomHex(16)
	}
	return context.WithValue(ctx, ctxKey{}, s), s
}

// Start begins a span using the tracer of the span in ctx; without one it returns a detached no-op span.
func Start(ctx context.Context, name string) (context.Context, *Span) {
	if parent := FromContext(ctx); parent != nil && parent.tracer != nil {
		return parent.tracer.Start(ctx, name, KindInternal)
	}
	return ctx, &Span{Attrs: map[string]string{}}
}

// Set records an attribute.
func (s *Span) Set(k, v string) { s.Attrs[k] = v }

// Finish ends the span and hands it to the exporter.
func (s *Span) Finish() {
	s.End = time.Now()
	if s.tracer != nil {
		s.tracer.export(s)
	}
}

// Traceparent formats the span as a W3C traceparent header value.
func (s *Span) Traceparent() string { return "00-" + s.TraceID + "-" + s.SpanID + "-01" }

// parseTraceparent extracts trace and parent span ids from a W3C traceparent header.
func parseTraceparent(h string) (traceID, spanID string, ok bool) {
	p := strings.Split(h, "-")
	if len(p) != 4 || len(p[1]) != 32 || len(p[2]) != 16 {
		return "", "", false
	}
	if _, err := hex.DecodeString(p[1] + p[2]); err != nil {
		return "", "", false
	}
	return p[1], p[2], true
}

// Inject propagates the current span to an outgoing request.
func Inject(ctx context.Context, h http.Header) {
	if s := FromContext(ctx); s != nil && s.TraceID != "" {
		h.Set("traceparent", s.Traceparent())
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

// Middleware starts a server span for every request, continuing the caller's trace if present.
func (t *Tracer) Middleware(route func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if traceID, spanID, ok := parseTraceparent(r.Header.Get("traceparent")); ok {
			ctx = context.WithValue(ctx, ctxKey{}, &Span{TraceID: traceID, SpanID: spanID})
		}
		ctx, span := t.Start(ctx, route(r), KindServer)
		span.Set("http.method", r.Method)
		span.Set("http.target", r.URL.Path)
		span.Set("tenant", r.Header.Get("X-Tenant-ID"))
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))
		span.Set("http.status_code", strconv.Itoa(rec.status))
		span.Error = rec.status >= 500
		span.Finish()
	})
}
