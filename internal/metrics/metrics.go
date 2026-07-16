// Package metrics defines the Prometheus series the operator publishes per position.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	labels = []string{"position", "sku"}

	Utilization = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_gpu_utilization_pct",
		Help: "Observed GPU utilization percent for a position.",
	}, labels)

	HedgePnL = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_hedge_pnl_usd_per_hour",
		Help: "Mark-to-market hedge P&L in USD per hour.",
	}, labels)

	BasisRisk = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_basis_risk_usd_per_hour",
		Help: "Unmatched exposure between the index hedge and real utilization, in USD per hour.",
	}, labels)

	HedgeEffectiveness = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_hedge_effectiveness_pct",
		Help: "How close the net economic result is to the locked hedge target, in percent.",
	}, labels)

	IdleGPUs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_idle_gpu_count",
		Help: "Number of idle GPUs in a position.",
	}, labels)

	AvailableForSublet = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_available_for_sublet",
		Help: "1 when a position has idle capacity flagged for the secondary market, else 0.",
	}, labels)

	SpotPrice = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chp_spot_price_usd_per_gpu_hour",
		Help: "OCPI spot price observed by the operator, in USD per GPU-hour.",
	}, labels)
)

// Register adds all operator metrics to the controller-runtime registry.
func Register() {
	ctrlmetrics.Registry.MustRegister(
		Utilization,
		HedgePnL,
		BasisRisk,
		HedgeEffectiveness,
		IdleGPUs,
		AvailableForSublet,
		SpotPrice,
	)
}
