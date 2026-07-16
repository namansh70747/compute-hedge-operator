// Package telemetry reads GPU utilization for a position.
//
// It talks to the bundled gpuexporter over JSON. On a real cluster this would query
// Prometheus for dcgm-exporter metrics instead; the interface is the same.
package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Source returns the current utilization percent for a named position.
type Source interface {
	Utilization(ctx context.Context, position string) (float64, error)
}

// HTTPSource reads utilization from the gpuexporter JSON API.
type HTTPSource struct {
	baseURL string
	client  *http.Client
}

// NewHTTPSource builds a source pointed at a gpuexporter base URL.
func NewHTTPSource(baseURL string) *HTTPSource {
	return &HTTPSource{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type utilResponse struct {
	Position       string  `json:"position"`
	UtilizationPct float64 `json:"utilizationPct"`
}

// Utilization fetches the current utilization percent for a position.
func (s *HTTPSource) Utilization(ctx context.Context, position string) (float64, error) {
	url := fmt.Sprintf("%s/positions/%s", s.baseURL, position)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("gpuexporter returned %d for position %q", resp.StatusCode, position)
	}
	var body utilResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	return body.UtilizationPct, nil
}
