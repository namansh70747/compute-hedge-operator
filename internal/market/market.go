// Package market publishes idle GPU capacity to a secondary marketplace.
//
// Two implementations satisfy Publisher:
//   - LogPublisher records offers locally and makes no external calls (the default,
//     used for the mock prototype).
//   - HTTPPublisher posts offers to a real marketplace endpoint (for example Ornn's),
//     and is only selected when a URL, credentials, and an explicit write flag are set.
//
// The reconciler does not care which one is wired in.
package market

import (
	"context"
	"sync"
	"time"
)

// Offer is a block of idle capacity a position is willing to sublet.
type Offer struct {
	ID              string    `json:"id"`
	Position        string    `json:"position"`
	Namespace       string    `json:"namespace"`
	SKU             string    `json:"sku"`
	GPUCount        int       `json:"gpuCount"`
	PriceUSDPerHour float64   `json:"priceUsdPerHour"`
	AsOf            time.Time `json:"asOf"`
}

// Publisher offers and withdraws idle capacity on a secondary marketplace.
type Publisher interface {
	PublishSupply(ctx context.Context, offer Offer) error
	WithdrawSupply(ctx context.Context, id string) error
}

// LogPublisher keeps the most recent offer in memory and makes no external calls.
type LogPublisher struct {
	mu    sync.Mutex
	last  Offer
	haveL bool
}

// NewLogPublisher builds the no-op mock publisher.
func NewLogPublisher() *LogPublisher { return &LogPublisher{} }

// PublishSupply records the offer locally.
func (p *LogPublisher) PublishSupply(_ context.Context, offer Offer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.last = offer
	p.haveL = true
	return nil
}

// WithdrawSupply clears the recorded offer.
func (p *LogPublisher) WithdrawSupply(_ context.Context, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.haveL = false
	return nil
}

// Last returns the most recent recorded offer, if any.
func (p *LogPublisher) Last() (Offer, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last, p.haveL
}
