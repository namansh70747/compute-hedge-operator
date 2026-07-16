// Package pricesim generates realistic GPU spot prices for local demos.
//
// Prices follow a mean-reverting random walk (an Ornstein-Uhlenbeck style process)
// around a per-SKU baseline. A demo can inject a temporary spike that decays back
// toward the baseline, which is how the operator's reaction is shown live.
package pricesim

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// SKU baseline prices in USD per GPU-hour. Values are illustrative, not quotes.
var defaultBaselines = map[string]float64{
	"H100":    2.50,
	"H200":    3.20,
	"B200":    5.00,
	"RTX5090": 0.80,
}

// spikeDecay is how much of a transient spike remains after each tick.
const spikeDecay = 0.7

type skuState struct {
	baseline   float64
	base       float64 // mean-reverting base price
	spike      float64 // transient spike offset, decays each tick, not folded into base
	volatility float64 // per-tick noise as a fraction of baseline
	reversion  float64 // mean-reversion strength in [0,1]
}

// Engine holds the live price state for all SKUs.
type Engine struct {
	mu    sync.RWMutex
	rng   *rand.Rand
	state map[string]*skuState
}

// New builds an engine seeded from the current time.
func New() *Engine {
	e := &Engine{
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
		state: make(map[string]*skuState),
	}
	for sku, base := range defaultBaselines {
		e.state[sku] = &skuState{
			baseline:   base,
			base:       base,
			volatility: 0.02,
			reversion:  0.1,
		}
	}
	return e
}

// Step advances every SKU price by one tick.
func (e *Engine) Step() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.state {
		// Mean-reverting base with Gaussian noise scaled by baseline.
		s.base += s.reversion * (s.baseline - s.base)
		s.base += e.rng.NormFloat64() * s.volatility * s.baseline
		if s.base < 0.01 {
			s.base = 0.01
		}
		// Decay any transient spike back toward zero.
		s.spike *= spikeDecay
		if math.Abs(s.spike) < 0.01 {
			s.spike = 0
		}
	}
}

// Spike adds a temporary jump to a SKU price, expressed as a fraction of its baseline.
func (e *Engine) Spike(sku string, fraction float64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.state[sku]
	if !ok {
		return false
	}
	s.spike += s.baseline * fraction
	return true
}

// Price returns the current price for a SKU.
func (e *Engine) Price(sku string) (float64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s, ok := e.state[sku]
	if !ok {
		return 0, false
	}
	return round2(s.base + s.spike), true
}

// Prices returns a copy of all current prices keyed by SKU.
func (e *Engine) Prices() map[string]float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]float64, len(e.state))
	for sku, s := range e.state {
		out[sku] = round2(s.base + s.spike)
	}
	return out
}

// SKUs returns the known SKUs in stable order.
func (e *Engine) SKUs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, 0, len(e.state))
	for sku := range e.state {
		out = append(out, sku)
	}
	sort.Strings(out)
	return out
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
