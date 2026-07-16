package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/namansh70747/compute-hedge-operator/internal/ocpi"
)

// PriceFetcher returns the current price per SKU in USD per GPU-hour.
type PriceFetcher interface {
	Fetch(ctx context.Context, skus []string) (map[string]float64, error)
}

// HTTPPrices reads all prices from the mock OCPI service bulk endpoint.
// Used only when the mock list endpoint is available as an enrichment; live mode
// uses SourcePrices instead.
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

// Fetch returns the current price map. The skus argument is ignored; the mock
// service returns every known SKU in one response.
func (h *HTTPPrices) Fetch(ctx context.Context, _ []string) (map[string]float64, error) {
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

// SourcePrices adapts an ocpi.Source (mock or live) to PriceFetcher by fetching
// each requested SKU. This is the path used when credentials flip the console to live.
type SourcePrices struct {
	Source ocpi.Source
}

// NewSourcePrices wraps an ocpi.Source as a PriceFetcher.
func NewSourcePrices(src ocpi.Source) *SourcePrices {
	return &SourcePrices{Source: src}
}

// Fetch returns prices for the given SKUs. Missing or failed SKUs are omitted.
func (s *SourcePrices) Fetch(ctx context.Context, skus []string) (map[string]float64, error) {
	out := map[string]float64{}
	if s.Source == nil {
		return out, nil
	}
	seen := map[string]bool{}
	for _, sku := range skus {
		if sku == "" || seen[sku] {
			continue
		}
		seen[sku] = true
		p, err := s.Source.Price(ctx, sku)
		if err != nil {
			continue
		}
		out[sku] = p.USDPerHour
	}
	return out, nil
}
