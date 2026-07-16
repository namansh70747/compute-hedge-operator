package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PriceFetcher returns the current price per SKU in USD per GPU-hour.
type PriceFetcher interface {
	Fetch(ctx context.Context) (map[string]float64, error)
}

// HTTPPrices reads all prices from the mock OCPI service.
type HTTPPrices struct {
	url    string
	client *http.Client
}

// NewHTTPPrices builds a fetcher pointed at a price service base URL.
func NewHTTPPrices(base string) *HTTPPrices {
	return &HTTPPrices{
		url:    base + "/prices",
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Fetch returns the current price map.
func (h *HTTPPrices) Fetch(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("price service returned %d", resp.StatusCode)
	}
	out := map[string]float64{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
