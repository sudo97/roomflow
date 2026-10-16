// Command roomflow runs the RoomFlow booking API.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudo97/roomflow/internal/booking"
	"github.com/sudo97/roomflow/internal/flags"
	"github.com/sudo97/roomflow/internal/httpapi"
	"github.com/sudo97/roomflow/internal/obs"
	"github.com/sudo97/roomflow/internal/store"
)

// Set at build time via -ldflags.
var (
	version = "dev"
	commit  = "unknown"
)

// buildInfo returns build metadata as key/value pairs; COLOR identifies the blue/green slot.
func buildInfo() []string {
	return []string{"version", version, "commit", commit, "color", os.Getenv("COLOR")}
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local /healthz endpoint and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	addr := envOr("ADDR", ":8080")
	switch {
	case *showVersion:
		fmt.Printf("roomflow %s (%s)\n", version, commit)
	case *healthcheck:
		os.Exit(probe("http://localhost" + addr + "/healthz"))
	default:
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		h, err := buildHandler(logger)
		if err != nil {
			log.Fatal(err)
		}
		servePprof(logger)
		srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
		logger.Info("starting", "addr", addr, "build", obs.Labels(buildInfo()...))
		if err := serve(srv, 10*time.Second); err != nil {
			log.Fatal(err)
		}
	}
}

// buildHandler assembles the application: API, feature flags, metrics, version and admin endpoints.
func buildHandler(logger *slog.Logger) (http.Handler, error) {
	reg := obs.NewRegistry()
	reg.Set("roomflow_build_info", "Build information.", obs.Labels(buildInfo()...), 1)
	fl, err := loadFlags(os.Getenv("FLAGS_FILE"))
	if err != nil {
		return nil, err
	}
	adminToken, err := adminTokenFromEnvOrVault()
	if err != nil {
		return nil, err
	}
	api := httpapi.New(booking.NewService(store.NewMemory())).WithFlags(fl).WithMetrics(reg).WithCache(newCache(reg))
	routes := api.Routes().(*http.ServeMux)
	mux := http.NewServeMux()
	mux.Handle("/", routes)
	mux.Handle("GET /metrics", reg.Handler())
	mux.HandleFunc("GET /version", serveVersion)
	mux.Handle("POST /admin/flags/reload", requireToken(adminToken, reloadFlags(fl)))
	return withMiddleware(reg, logger, routeName(routes, mux), mux), nil
}

// loadFlags reads feature flags from a JSON file and watches it for changes (e.g. a mounted ConfigMap).
func loadFlags(path string) (*flags.Store, error) {
	if path == "" {
		return flags.New(nil), nil
	}
	fl, err := flags.FromFile(path)
	if err != nil {
		return nil, fmt.Errorf("feature flags: %w", err)
	}
	go fl.Watch(5*time.Second, nil)
	return fl, nil
}

func serveVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	info, out := buildInfo(), map[string]string{}
	for i := 0; i+1 < len(info); i += 2 {
		out[info[i]] = info[i+1]
	}
	_ = json.NewEncoder(w).Encode(out)
}

func reloadFlags(fl *flags.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := fl.Reload(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// routeName returns the matched route pattern, used as a low-cardinality metric label.
func routeName(inner, outer *http.ServeMux) func(*http.Request) string {
	return func(r *http.Request) string {
		if _, pattern := inner.Handler(r); pattern != "" {
			return pattern
		}
		_, pattern := outer.Handler(r)
		return pattern
	}
}

// serve runs the server until SIGTERM/SIGINT, then stops accepting connections and lets in-flight
// requests finish (graceful shutdown) so that rolling updates do not drop requests.
func serve(srv *http.Server, grace time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	return srv.Shutdown(shutdown)
}

// probe returns a process exit code: 0 if the endpoint answers 200, 1 otherwise.
func probe(url string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
