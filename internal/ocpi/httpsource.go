package ocpi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPSource reads prices from the bundled mock price service.
type HTTPSource struct {
	baseURL string
	client  *http.Client
}

// NewHTTPSource builds a source pointed at a mock price service base URL.
func NewHTTPSource(baseURL string) *HTTPSource {
	return &HTTPSource{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// Price fetches the current price for a SKU from the mock service.
func (s *HTTPSource) Price(ctx context.Context, sku string) (Price, error) {
	url := fmt.Sprintf("%s/prices/%s", s.baseURL, sku)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Price{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return Price{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Price{}, fmt.Errorf("price service returned %d for sku %q", resp.StatusCode, sku)
	}
	var p Price
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return Price{}, err
	}
	return p, nil
}
