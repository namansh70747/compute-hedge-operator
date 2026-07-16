// Command console serves the Compute Hedge live control-room UI and its JSON API.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1alpha1 "github.com/namansh70747/compute-hedge-operator/api/v1alpha1"
	"github.com/namansh70747/compute-hedge-operator/internal/console"
)

//go:embed all:web/dist
var webFS embed.FS

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = computev1alpha1.AddToScheme(scheme)
}

func main() {
	addr := envDefault("LISTEN_ADDR", ":8090")
	priceURL := envDefault("OCPI_URL", "http://mockocpi:8080")
	cluster := envDefault("CLUSTER_NAME", "kind-compute-hedge")
	interval := envDuration("POLL_INTERVAL", 2*time.Second)

	cfg := ctrl.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("build client: %v", err)
	}

	builder := console.NewBuilder(c, console.NewHTTPPrices(priceURL), cluster)

	var (
		mu     sync.RWMutex
		latest console.State
	)
	poll := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		st, err := builder.Build(ctx)
		if err != nil {
			log.Printf("build state: %v", err)
			return
		}
		mu.Lock()
		latest = st
		mu.Unlock()
	}
	poll()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			poll()
		}
	}()

	dist, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("sub web/dist: %v", err)
	}
	fileServer := http.FileServer(http.FS(dist))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, _ *http.Request) {
		mu.RLock()
		st := latest
		mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/", spaHandler(dist, fileServer))

	log.Printf("console listening on %s (prices=%s, cluster=%s)", addr, priceURL, cluster)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

// spaHandler serves static assets, falling back to index.html for client routes.
func spaHandler(dist fs.FS, fileServer http.Handler) http.HandlerFunc {
	index, _ := fs.ReadFile(dist, "index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			serveIndex(w, index)
			return
		}
		if _, err := fs.Stat(dist, p); err != nil {
			serveIndex(w, index)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

func serveIndex(w http.ResponseWriter, index []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
