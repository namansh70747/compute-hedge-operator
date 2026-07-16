package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PrometheusSource reads GPU utilization from Prometheus, typically backed by
// dcgm-exporter (metric DCGM_FI_DEV_GPU_UTIL). The query is a template so the label
// matching can be adapted to any cluster without code changes.
type PrometheusSource struct {
	baseURL string
	query   string
	client  *http.Client
}

// PrometheusConfig configures the live telemetry source.
type PrometheusConfig struct {
	BaseURL string // e.g. http://prometheus:9090
	// Query is a PromQL template; {position} and {namespace} are substituted.
	// Defaults to avg(DCGM_FI_DEV_GPU_UTIL{position="{position}"}).
	Query string
}

// NewPrometheusSource builds a live telemetry source.
func NewPrometheusSource(cfg PrometheusConfig) *PrometheusSource {
	q := cfg.Query
	if q == "" {
		q = `avg(DCGM_FI_DEV_GPU_UTIL{position="{position}"})`
	}
	return &PrometheusSource{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		query:   q,
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value [2]interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Utilization runs the configured instant query for a position and returns the value.
func (s *PrometheusSource) Utilization(ctx context.Context, position, namespace string) (float64, error) {
	q := strings.ReplaceAll(s.query, "{position}", position)
	q = strings.ReplaceAll(q, "{namespace}", namespace)
	endpoint := fmt.Sprintf("%s/api/v1/query?query=%s", s.baseURL, url.QueryEscape(q))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prometheus returned %d for position %q", resp.StatusCode, position)
	}
	var body promResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if body.Status != "success" || len(body.Data.Result) == 0 {
		return 0, fmt.Errorf("no utilization series for position %q", position)
	}
	raw, ok := body.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected prometheus value type for position %q", position)
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
