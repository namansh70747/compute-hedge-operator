// Package console aggregates live ComputePosition state for the web console.
package console

import (
	"context"
	"sort"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1alpha1 "github.com/namansh70747/compute-hedge-operator/api/v1alpha1"
)

const (
	historyPoints = 60
	maxEvents     = 25
)

// Portfolio holds the aggregate view across all positions.
type Portfolio struct {
	HedgedNotionalUSDPerHour    float64          `json:"hedgedNotionalUSDPerHour"`
	HedgePnLUSDPerHour          float64          `json:"hedgePnLUSDPerHour"`
	BasisRiskUSDPerHour         float64          `json:"basisRiskUSDPerHour"`
	IdleGPUsAvailable           int              `json:"idleGPUsAvailable"`
	MarketplaceSupplyUSDPerHour float64          `json:"marketplaceSupplyUSDPerHour"`
	Positions                   int              `json:"positions"`
	History                     PortfolioHistory `json:"history"`
}

// PortfolioHistory holds rolling series for the portfolio tiles.
type PortfolioHistory struct {
	HedgePnL  []float64 `json:"hedgePnL"`
	BasisRisk []float64 `json:"basisRisk"`
	Notional  []float64 `json:"notional"`
	Supply    []float64 `json:"supply"`
}

// PriceView is one SKU price plus its change since the previous poll.
type PriceView struct {
	SKU        string  `json:"sku"`
	USDPerHour float64 `json:"usdPerHour"`
	ChangePct  float64 `json:"changePct"`
}

// PositionHistory holds rolling series for a position's sparklines.
type PositionHistory struct {
	BasisRisk   []float64 `json:"basisRisk"`
	Utilization []float64 `json:"utilization"`
	PnL         []float64 `json:"pnl"`
}

// PositionView is the console's per-position projection.
type PositionView struct {
	Name                  string          `json:"name"`
	Namespace             string          `json:"namespace"`
	SKU                   string          `json:"sku"`
	GPUCount              int             `json:"gpuCount"`
	Priority              string          `json:"priority"`
	Phase                 string          `json:"phase"`
	UtilizationPct        int             `json:"utilizationPct"`
	SpotUSDPerHour        float64         `json:"spotUSDPerHour"`
	HedgedUSDPerHour      float64         `json:"hedgedUSDPerHour"`
	HedgePnLUSDPerHour    float64         `json:"hedgePnLUSDPerHour"`
	BasisRiskUSDPerHour   float64         `json:"basisRiskUSDPerHour"`
	HedgeEffectivenessPct int             `json:"hedgeEffectivenessPct"`
	IdleGPUCount          int             `json:"idleGPUCount"`
	AvailableForSublet    bool            `json:"availableForSublet"`
	PriceStale            bool            `json:"priceStale"`
	Recommendation        string          `json:"recommendation"`
	History               PositionHistory `json:"history"`
}

// EventView is a recent controller event.
type EventView struct {
	Time     string `json:"time"`
	Position string `json:"position"`
	Reason   string `json:"reason"`
	Type     string `json:"type"`
	Message  string `json:"message"`
}

// State is the full payload served to the console UI.
type State struct {
	AsOf      string         `json:"asOf"`
	Cluster   string         `json:"cluster"`
	Portfolio Portfolio      `json:"portfolio"`
	Prices    []PriceView    `json:"prices"`
	Positions []PositionView `json:"positions"`
	Events    []EventView    `json:"events"`
}

// Builder assembles State from the cluster and price feed.
type Builder struct {
	client     client.Client
	prices     PriceFetcher
	cluster    string
	hist       *Store
	prevPrices map[string]float64
}

// NewBuilder constructs a Builder.
func NewBuilder(c client.Client, prices PriceFetcher, cluster string) *Builder {
	return &Builder{
		client:     c,
		prices:     prices,
		cluster:    cluster,
		hist:       NewStore(historyPoints),
		prevPrices: map[string]float64{},
	}
}

// Build produces a fresh State and advances the rolling history.
func (b *Builder) Build(ctx context.Context) (State, error) {
	var list computev1alpha1.ComputePositionList
	if err := b.client.List(ctx, &list); err != nil {
		return State{}, err
	}

	priceMap, _ := b.prices.Fetch(ctx)

	state := State{
		AsOf:    time.Now().UTC().Format(time.RFC3339),
		Cluster: b.cluster,
		Prices:  b.buildPrices(priceMap),
	}

	var pf Portfolio
	pf.Positions = len(list.Items)

	for i := range list.Items {
		pos := &list.Items[i]
		view := b.buildPosition(pos)
		state.Positions = append(state.Positions, view)

		pf.HedgedNotionalUSDPerHour += view.HedgedUSDPerHour * float64(view.GPUCount)
		pf.HedgePnLUSDPerHour += view.HedgePnLUSDPerHour
		pf.BasisRiskUSDPerHour += view.BasisRiskUSDPerHour
		if view.AvailableForSublet {
			pf.IdleGPUsAvailable += view.IdleGPUCount
			pf.MarketplaceSupplyUSDPerHour += float64(view.IdleGPUCount) * view.SpotUSDPerHour
		}
	}

	sort.Slice(state.Positions, func(i, j int) bool {
		return state.Positions[i].Name < state.Positions[j].Name
	})

	b.hist.Push("pf/pnl", pf.HedgePnLUSDPerHour)
	b.hist.Push("pf/basis", pf.BasisRiskUSDPerHour)
	b.hist.Push("pf/notional", pf.HedgedNotionalUSDPerHour)
	b.hist.Push("pf/supply", pf.MarketplaceSupplyUSDPerHour)
	pf.History = PortfolioHistory{
		HedgePnL:  b.hist.Snapshot("pf/pnl"),
		BasisRisk: b.hist.Snapshot("pf/basis"),
		Notional:  b.hist.Snapshot("pf/notional"),
		Supply:    b.hist.Snapshot("pf/supply"),
	}
	state.Portfolio = pf

	state.Events = b.buildEvents(ctx)
	return state, nil
}

func (b *Builder) buildPrices(priceMap map[string]float64) []PriceView {
	prices := make([]PriceView, 0, len(priceMap))
	for sku, price := range priceMap {
		change := 0.0
		if prev, ok := b.prevPrices[sku]; ok && prev > 0 {
			change = (price - prev) / prev * 100
		}
		prices = append(prices, PriceView{SKU: sku, USDPerHour: price, ChangePct: round2(change)})
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i].SKU < prices[j].SKU })
	if len(priceMap) > 0 {
		b.prevPrices = priceMap
	}
	return prices
}

func (b *Builder) buildPosition(pos *computev1alpha1.ComputePosition) PositionView {
	name := pos.Name
	view := PositionView{
		Name:                  name,
		Namespace:             pos.Namespace,
		SKU:                   pos.Spec.SKU,
		GPUCount:              int(pos.Spec.GPUCount),
		Priority:              string(pos.Spec.Priority),
		Phase:                 orText(pos.Status.Phase, "Pending"),
		UtilizationPct:        int(pos.Status.UtilizationPct),
		SpotUSDPerHour:        parseFloat(pos.Status.SpotPriceUSDPerHour),
		HedgedUSDPerHour:      parseFloat(pos.Spec.HedgedPriceUSDPerHour),
		HedgePnLUSDPerHour:    parseFloat(pos.Status.HedgePnLUSDPerHour),
		BasisRiskUSDPerHour:   parseFloat(pos.Status.BasisRiskUSDPerHour),
		HedgeEffectivenessPct: int(pos.Status.HedgeEffectivenessPct),
		IdleGPUCount:          int(pos.Status.IdleGPUCount),
		AvailableForSublet:    pos.Status.AvailableForSublet,
		PriceStale:            pos.Status.PriceStale,
		Recommendation:        pos.Status.Recommendation,
	}

	b.hist.Push("pos/"+name+"/basis", view.BasisRiskUSDPerHour)
	b.hist.Push("pos/"+name+"/util", float64(view.UtilizationPct))
	b.hist.Push("pos/"+name+"/pnl", view.HedgePnLUSDPerHour)
	view.History = PositionHistory{
		BasisRisk:   b.hist.Snapshot("pos/" + name + "/basis"),
		Utilization: b.hist.Snapshot("pos/" + name + "/util"),
		PnL:         b.hist.Snapshot("pos/" + name + "/pnl"),
	}
	return view
}

func (b *Builder) buildEvents(ctx context.Context) []EventView {
	var evlist corev1.EventList
	if err := b.client.List(ctx, &evlist); err != nil {
		return nil
	}
	var out []EventView
	events := evlist.Items
	filtered := make([]corev1.Event, 0, len(events))
	for i := range events {
		if events[i].InvolvedObject.Kind == "ComputePosition" {
			filtered = append(filtered, events[i])
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return eventTime(&filtered[i]).After(eventTime(&filtered[j]))
	})
	if len(filtered) > maxEvents {
		filtered = filtered[:maxEvents]
	}
	for i := range filtered {
		e := &filtered[i]
		out = append(out, EventView{
			Time:     eventTime(e).UTC().Format(time.RFC3339),
			Position: e.InvolvedObject.Name,
			Reason:   e.Reason,
			Type:     e.Type,
			Message:  e.Message,
		})
	}
	return out
}

func eventTime(e *corev1.Event) time.Time {
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time
	}
	if !e.EventTime.IsZero() {
		return e.EventTime.Time
	}
	return e.CreationTimestamp.Time
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func orText(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
