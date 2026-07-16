// Command gpuexporter simulates per-position GPU utilization for local demos.
//
// On a real cluster this role is filled by NVIDIA's dcgm-exporter. Here it produces
// a controllable utilization signal per ComputePosition: the operator reads it over a
// small JSON API, and it also publishes dcgm-style Prometheus metrics for the dashboard.
// A demo forces a position idle with POST /control/{name}?util=5.
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type positionState struct {
	baseline float64
	util     float64
	forced   *float64
}

type store struct {
	mu    sync.Mutex
	rng   *rand.Rand
	items map[string]*positionState
}

func newStore() *store {
	return &store{
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
		items: make(map[string]*positionState),
	}
}

func (s *store) getOrCreate(name string) *positionState {
	st, ok := s.items[name]
	if !ok {
		base := 55 + s.rng.Float64()*20 // 55-75%
		st = &positionState{baseline: base, util: base}
		s.items[name] = st
	}
	return st
}

func (s *store) utilization(name string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.getOrCreate(name)
	if st.forced != nil {
		return *st.forced
	}
	return st.util
}

func (s *store) control(name string, util float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.getOrCreate(name)
	if util < 0 {
		st.forced = nil
		return
	}
	v := clamp(util, 0, 100)
	st.forced = &v
}

func (s *store) step() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range s.items {
		if st.forced != nil {
			continue
		}
		st.util += st.rngWalk(s.rng)
		st.util = clamp(st.util, 0, 100)
	}
}

func (st *positionState) rngWalk(rng *rand.Rand) float64 {
	// Mild mean reversion toward baseline plus noise.
	return 0.1*(st.baseline-st.util) + rng.NormFloat64()*3
}

func (s *store) snapshot() map[string]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]float64, len(s.items))
	for name, st := range s.items {
		if st.forced != nil {
			out[name] = *st.forced
			continue
		}
		out[name] = round1(st.util)
	}
	return out
}

func main() {
	addr := envDefault("LISTEN_ADDR", ":8081")
	tick := envDuration("TICK_INTERVAL", 3*time.Second)

	st := newStore()
	// Seed the sample positions so the dashboard has series from the start.
	for _, name := range []string{"training-cluster", "batch-render"} {
		st.getOrCreate(name)
	}

	utilGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "DCGM_FI_DEV_GPU_UTIL",
		Help: "Simulated GPU utilization percent (dcgm-style) per position.",
	}, []string{"position"})
	prometheus.MustRegister(utilGauge)

	go func() {
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for range ticker.C {
			st.step()
			for name, util := range st.snapshot() {
				utilGauge.WithLabelValues(name).Set(util)
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /positions/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		writeJSON(w, http.StatusOK, map[string]any{
			"position":       name,
			"utilizationPct": round1(st.utilization(name)),
			"asOf":           time.Now().UTC().Format(time.RFC3339),
		})
	})

	// POST /control/{name}?util=5 forces utilization; util<0 resumes the random walk.
	mux.HandleFunc("POST /control/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		util := -1.0
		if q := r.URL.Query().Get("util"); q != "" {
			if v, err := strconv.ParseFloat(q, 64); err == nil {
				util = v
			}
		}
		st.control(name, util)
		writeJSON(w, http.StatusOK, map[string]any{"position": name, "forcedUtil": util})
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	log.Printf("gpuexporter listening on %s (tick=%s)", addr, tick)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
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
