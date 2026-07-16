// Command mockocpi serves simulated OCPI GPU spot prices for local demos.
//
// It exposes a small JSON API the operator reads, Prometheus metrics for the
// dashboard, and a spike endpoint used to drive the demo live.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/namansh70747/compute-hedge-operator/internal/pricesim"
)

func main() {
	addr := envDefault("LISTEN_ADDR", ":8080")
	tick := envDuration("TICK_INTERVAL", 2*time.Second)

	engine := pricesim.New()

	priceGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocpi_price_usd_per_gpu_hour",
		Help: "Simulated OCPI spot price in USD per GPU-hour.",
	}, []string{"sku"})
	prometheus.MustRegister(priceGauge)

	go func() {
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for range ticker.C {
			engine.Step()
			for sku, price := range engine.Prices() {
				priceGauge.WithLabelValues(sku).Set(price)
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /prices", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, engine.Prices())
	})

	mux.HandleFunc("GET /prices/{sku}", func(w http.ResponseWriter, r *http.Request) {
		sku := r.PathValue("sku")
		price, ok := engine.Price(sku)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown sku"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sku":        sku,
			"usdPerHour": price,
			"asOf":       time.Now().UTC().Format(time.RFC3339),
		})
	})

	// POST /spike/{sku}?fraction=0.5 injects a temporary price jump for demos.
	mux.HandleFunc("POST /spike/{sku}", func(w http.ResponseWriter, r *http.Request) {
		sku := r.PathValue("sku")
		fraction := 0.5
		if q := r.URL.Query().Get("fraction"); q != "" {
			if f, err := strconv.ParseFloat(q, 64); err == nil {
				fraction = f
			}
		}
		if !engine.Spike(sku, fraction) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown sku"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sku": sku, "spiked": fraction})
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	log.Printf("mockocpi listening on %s (tick=%s, skus=%v)", addr, tick, engine.SKUs())
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
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
