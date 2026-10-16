// Command frontend is the web front of RoomFlow: it serves the booking page and proxies /api/ to the
// backend, propagating the trace context. It is the second service of the microservice deployment (labs 10-12).
package main

import (
	"encoding/json"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/sudo97/roomflow/internal/obs"
	"github.com/sudo97/roomflow/internal/trace"
)

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="uk"><head><meta charset="utf-8"><title>RoomFlow</title>
<style>body{font-family:sans-serif;margin:2rem}.banner{padding:.6rem 1rem;color:#fff;background:{{.Color}};border-radius:6px}</style></head>
<body><div class="banner" id="version">RoomFlow frontend {{.Version}} ({{.Color}})</div>
<h1>Номери готелю</h1><pre id="rooms">loading...</pre>
<script>fetch('/api/rooms',{headers:{'X-Tenant-ID':'demo-hotel'}}).then(r=>r.json()).then(d=>{document.getElementById('rooms').textContent=JSON.stringify(d,null,2)})</script>
</body></html>`))

type info struct{ Version, Color string }

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// newHandler builds the frontend: page, version, health, metrics and a traced reverse proxy to backend.
func newHandler(backend *url.URL, in info, tracer *trace.Tracer, reg *obs.Registry, logger *slog.Logger) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(backend)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) { _ = page.Execute(w, in) })
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"service": "frontend", "version": in.Version, "color": in.Color})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.Handle("GET /metrics", reg.Handler())
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "proxy "+r.Method+" "+r.URL.Path, trace.KindClient)
		r = r.WithContext(ctx)
		trace.Inject(ctx, r.Header)
		proxy.ServeHTTP(w, r)
		span.Finish()
	}))
	route := func(r *http.Request) string {
		_, pattern := mux.Handler(r)
		return pattern
	}
	return obs.Instrument(reg, logger, route, tracer.Middleware(route, mux))
}

func main() {
	backend, err := url.Parse(envOr("BACKEND_URL", "http://localhost:8080"))
	if err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	in := info{Version: envOr("FRONTEND_VERSION", "v1"), Color: envOr("COLOR", "steelblue")}
	tracer := trace.New(envOr("OTEL_SERVICE_NAME", "frontend"), os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	go tracer.Run(2*time.Second, nil)
	reg := obs.NewRegistry()
	reg.Set("roomflow_frontend_build_info", "Frontend build information.", obs.Labels("version", in.Version, "color", in.Color), 1)
	srv := &http.Server{Addr: envOr("ADDR", ":8081"), Handler: newHandler(backend, in, tracer, reg, logger), ReadHeaderTimeout: 5 * time.Second}
	logger.Info("frontend starting", "addr", srv.Addr, "backend", backend.String(), "build", in.Version)
	log.Fatal(srv.ListenAndServe())
}
