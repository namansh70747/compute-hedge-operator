package hedge

import (
	"math"
	"testing"
)

func almost(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestFullUtilizationLocksTarget(t *testing.T) {
	// At full utilization the net economic result equals the locked target for any spot price.
	for _, spot := range []float64{1.0, 2.5, 4.0, 10.0} {
		r := Compute(Inputs{GPUCount: 10, UtilizationPct: 100, SpotPriceUSD: spot, HedgedPriceUSD: 2.5})
		if !almost(r.NetEconomicUSDPerHour, r.LockedTargetUSDPerHour) {
			t.Fatalf("spot=%v: net=%v want lockedTarget=%v", spot, r.NetEconomicUSDPerHour, r.LockedTargetUSDPerHour)
		}
		if !almost(r.BasisRiskUSDPerHour, 0) {
			t.Fatalf("spot=%v: basis risk=%v want 0 at full utilization", spot, r.BasisRiskUSDPerHour)
		}
		if !almost(r.HedgeEffectivenessPct, 100) {
			t.Fatalf("spot=%v: effectiveness=%v want 100", spot, r.HedgeEffectivenessPct)
		}
	}
}

func TestBasisRiskEqualsIdleValuedAtSpot(t *testing.T) {
	// Basis risk must equal the idle GPUs valued at the current spot price.
	r := Compute(Inputs{GPUCount: 10, UtilizationPct: 50, SpotPriceUSD: 3.0, HedgedPriceUSD: 2.5})
	wantBasis := 3.0 * (10 - 5) // idle 5 GPUs at spot 3
	if !almost(r.BasisRiskUSDPerHour, wantBasis) {
		t.Fatalf("basis risk=%v want %v", r.BasisRiskUSDPerHour, wantBasis)
	}
	if !almost(r.IdleGPUs, 5) {
		t.Fatalf("idle gpus=%v want 5", r.IdleGPUs)
	}
	if !almost(r.HedgePnLUSDPerHour, (2.5-3.0)*10) {
		t.Fatalf("hedge pnl=%v want %v", r.HedgePnLUSDPerHour, (2.5-3.0)*10)
	}
}

func TestEffectivenessTracksPriceNotJustUtilization(t *testing.T) {
	// Effectiveness should differ from raw utilization when spot != hedged.
	r := Compute(Inputs{GPUCount: 10, UtilizationPct: 50, SpotPriceUSD: 3.0, HedgedPriceUSD: 2.5})
	if almost(r.HedgeEffectivenessPct, 50) {
		t.Fatalf("effectiveness=%v should not equal utilization when priced differently", r.HedgeEffectivenessPct)
	}
}

func TestClampsUtilization(t *testing.T) {
	over := Compute(Inputs{GPUCount: 4, UtilizationPct: 150, SpotPriceUSD: 2, HedgedPriceUSD: 2})
	if !almost(over.UtilizedGPUs, 4) {
		t.Fatalf("utilized=%v want clamped to 4", over.UtilizedGPUs)
	}
	under := Compute(Inputs{GPUCount: 4, UtilizationPct: -20, SpotPriceUSD: 2, HedgedPriceUSD: 2})
	if !almost(under.UtilizedGPUs, 0) {
		t.Fatalf("utilized=%v want clamped to 0", under.UtilizedGPUs)
	}
}

func TestZeroHedgedPriceDoesNotPanic(t *testing.T) {
	r := Compute(Inputs{GPUCount: 4, UtilizationPct: 50, SpotPriceUSD: 2, HedgedPriceUSD: 0})
	if r.HedgeEffectivenessPct != 0 {
		t.Fatalf("effectiveness=%v want 0 when locked target is zero", r.HedgeEffectivenessPct)
	}
}
