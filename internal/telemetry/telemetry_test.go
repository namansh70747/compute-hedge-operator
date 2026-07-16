package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSourceUtilization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/positions/batch-render" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(utilResponse{Position: "batch-render", UtilizationPct: 12})
	}))
	defer srv.Close()

	src := NewHTTPSource(srv.URL)
	u, err := src.Utilization(context.Background(), "batch-render", "ns")
	if err != nil {
		t.Fatal(err)
	}
	if u != 12 {
		t.Fatalf("util = %v", u)
	}
}

func TestPrometheusSourceUtilization(t *testing.T) {
	var sawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		sawQuery = r.URL.Query().Get("query")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []map[string]any{
					{"value": []any{1.0, "87.5"}},
				},
			},
		})
	}))
	defer srv.Close()

	src := NewPrometheusSource(PrometheusConfig{
		BaseURL: srv.URL,
		Query:   `avg(DCGM_FI_DEV_GPU_UTIL{position="{position}",namespace="{namespace}"})`,
	})
	u, err := src.Utilization(context.Background(), "train-a", "gpu-ns")
	if err != nil {
		t.Fatal(err)
	}
	if u != 87.5 {
		t.Fatalf("util = %v", u)
	}
	if !strings.Contains(sawQuery, `position="train-a"`) {
		t.Fatalf("query missing position: %s", sawQuery)
	}
	if !strings.Contains(sawQuery, `namespace="gpu-ns"`) {
		t.Fatalf("query missing namespace: %s", sawQuery)
	}
}

func TestPrometheusSourceEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"resultType": "vector", "result": []any{}},
		})
	}))
	defer srv.Close()

	src := NewPrometheusSource(PrometheusConfig{BaseURL: srv.URL})
	if _, err := src.Utilization(context.Background(), "missing", "ns"); err == nil {
		t.Fatal("expected error on empty result")
	}
}
