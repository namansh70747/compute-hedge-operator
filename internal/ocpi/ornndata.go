package ocpi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OrnnDataSource reads prices from the real Ornn Data API.
//
// OCPI is distributed to institutional subscribers (for example over the Bloomberg
// Terminal and Ornn's own API at api.ornnai.com). This client is the integration point:
// it needs a base URL and a bearer token from an Ornn Data subscription. The response
// path and field names are configurable so it can be adapted to the subscribed feed
// without code changes.
type OrnnDataSource struct {
	baseURL   string
	token     string
	pricePath string
	client    *http.Client
}

// OrnnDataConfig configures the real price source.
type OrnnDataConfig struct {
	BaseURL string // e.g. https://api.ornnai.com
	Token   string // bearer token from an Ornn Data subscription
	// PricePath is a template for the price endpoint; {sku} is substituted.
	// Defaults to "/v1/ocpi/{sku}/spot".
	PricePath string
}

// NewOrnnDataSource builds a client for the real Ornn Data API.
func NewOrnnDataSource(cfg OrnnDataConfig) *OrnnDataSource {
	path := cfg.PricePath
	if path == "" {
		path = "/v1/ocpi/{sku}/spot"
	}
	return &OrnnDataSource{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		token:     cfg.Token,
		pricePath: path,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

type ornnPriceResponse struct {
	Price float64 `json:"price"`
	AsOf  string  `json:"asOf"`
}

// Price fetches the current OCPI spot price for a SKU.
func (s *OrnnDataSource) Price(ctx context.Context, sku string) (Price, error) {
	if s.token == "" {
		return Price{}, fmt.Errorf("ornn data api token not configured")
	}
	url := s.baseURL + strings.ReplaceAll(s.pricePath, "{sku}", sku)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Price{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Price{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Price{}, fmt.Errorf("ornn data api returned %d for sku %q", resp.StatusCode, sku)
	}
	var body ornnPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Price{}, err
	}
	asOf := time.Now()
	if body.AsOf != "" {
		if t, err := time.Parse(time.RFC3339, body.AsOf); err == nil {
			asOf = t
		}
	}
	return Price{SKU: sku, USDPerHour: body.Price, AsOf: asOf}, nil
}
