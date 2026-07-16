// Package hedge holds the basis-risk and hedge-effectiveness math for a compute position.
//
// Model (operator/seller view):
//   - The operator locks in a per-GPU price via a short position settled against the OCPI index.
//   - Physical revenue is earned only on GPUs that are actually utilized.
//   - When utilization is below full, the hedge notional (all GPUs) no longer matches the
//     real economic exposure (utilized GPUs). That mismatch is basis risk.
//
// Identity used throughout:
//
//	net           = physicalRevenue + hedgePnL
//	lockedTarget  = hedgedPrice * gpuCount        // revenue if perfectly hedged and fully used
//	basisRisk     = lockedTarget - net = spot * (gpuCount - utilizedGPUs)
//
// At full utilization net == lockedTarget and basis risk is zero, regardless of price.
package hedge

// Inputs are the per-hour, per-GPU figures for a single position.
type Inputs struct {
	GPUCount       int32
	UtilizationPct int32
	SpotPriceUSD   float64
	HedgedPriceUSD float64
}

// Result holds the derived per-hour economics for a position.
type Result struct {
	UtilizedGPUs              float64
	IdleGPUs                  float64
	SpotCostUSDPerHour        float64
	PhysicalRevenueUSDPerHour float64
	HedgePnLUSDPerHour        float64
	LockedTargetUSDPerHour    float64
	NetEconomicUSDPerHour     float64
	BasisRiskUSDPerHour       float64
	HedgeEffectivenessPct     float64
}

// Compute derives the position economics from live inputs.
func Compute(in Inputs) Result {
	util := clampPct(in.UtilizationPct)
	gpus := float64(in.GPUCount)

	utilized := gpus * float64(util) / 100.0
	idle := gpus - utilized

	spotCost := in.SpotPriceUSD * gpus
	physicalRevenue := in.SpotPriceUSD * utilized
	hedgePnL := (in.HedgedPriceUSD - in.SpotPriceUSD) * gpus
	lockedTarget := in.HedgedPriceUSD * gpus
	net := physicalRevenue + hedgePnL
	basisRisk := lockedTarget - net

	effectiveness := 0.0
	if lockedTarget > 0 {
		effectiveness = clampFloat(net/lockedTarget*100.0, 0, 100)
	}

	return Result{
		UtilizedGPUs:              utilized,
		IdleGPUs:                  idle,
		SpotCostUSDPerHour:        spotCost,
		PhysicalRevenueUSDPerHour: physicalRevenue,
		HedgePnLUSDPerHour:        hedgePnL,
		LockedTargetUSDPerHour:    lockedTarget,
		NetEconomicUSDPerHour:     net,
		BasisRiskUSDPerHour:       basisRisk,
		HedgeEffectivenessPct:     effectiveness,
	}
}

func clampPct(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
