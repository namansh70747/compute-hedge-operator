// Package ocpi provides pluggable access to GPU compute spot prices.
//
// Two implementations are provided:
//   - HTTPSource talks to the bundled mock price service (used for local demos).
//   - OrnnDataSource talks to the real Ornn Data API when credentials are configured.
//
// Both satisfy Source, so the controller does not care which one is wired in.
package ocpi

import (
	"context"
	"time"
)

// Price is a single GPU spot price observation.
type Price struct {
	SKU        string    `json:"sku"`
	USDPerHour float64   `json:"usdPerHour"`
	AsOf       time.Time `json:"asOf"`
}

// Source returns the latest spot price for a GPU SKU.
type Source interface {
	Price(ctx context.Context, sku string) (Price, error)
}

// Stale reports whether a price observation is older than the given max age.
func (p Price) Stale(maxAge time.Duration) bool {
	if p.AsOf.IsZero() {
		return true
	}
	return time.Since(p.AsOf) > maxAge
}
