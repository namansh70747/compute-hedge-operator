package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	computev1alpha1 "github.com/namansh70747/compute-hedge-operator/api/v1alpha1"
	"github.com/namansh70747/compute-hedge-operator/internal/hedge"
	"github.com/namansh70747/compute-hedge-operator/internal/metrics"
	"github.com/namansh70747/compute-hedge-operator/internal/ocpi"
	"github.com/namansh70747/compute-hedge-operator/internal/telemetry"
)

const (
	defaultIdleThresholdPct = 15
	defaultIdleWindowSecs   = 30
	idleResetMarginPct      = 10
	priceMaxAge             = 2 * time.Minute
	reconcileInterval       = 10 * time.Second
)

// ComputePositionReconciler reconciles a ComputePosition against live price and utilization.
type ComputePositionReconciler struct {
	client.Client
	Recorder  record.EventRecorder
	OCPI      ocpi.Source
	Telemetry telemetry.Source
}

// +kubebuilder:rbac:groups=computehedge.dev,resources=computepositions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=computehedge.dev,resources=computepositions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile pulls the current price and utilization for a position, computes its
// economics, flags idle capacity for the secondary market, and (only when opted in)
// pauses or resumes the referenced batch workload.
func (r *ComputePositionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var pos computev1alpha1.ComputePosition
	if err := r.Get(ctx, req.NamespacedName, &pos); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	hedgedPrice, err := strconv.ParseFloat(pos.Spec.HedgedPriceUSDPerHour, 64)
	if err != nil {
		r.Recorder.Eventf(&pos, "Warning", "InvalidSpec", "hedgedPriceUSDPerHour %q is not a number", pos.Spec.HedgedPriceUSDPerHour)
		return ctrl.Result{RequeueAfter: reconcileInterval}, nil
	}

	util, err := r.Telemetry.Utilization(ctx, pos.Name)
	if err != nil {
		l.Info("utilization unavailable, requeueing", "error", err.Error())
		r.Recorder.Eventf(&pos, "Warning", "TelemetryUnavailable", "could not read utilization: %v", err)
		return ctrl.Result{RequeueAfter: reconcileInterval}, nil
	}

	priceStale := false
	price, err := r.OCPI.Price(ctx, pos.Spec.SKU)
	if err != nil {
		priceStale = true
		if prev, perr := strconv.ParseFloat(pos.Status.SpotPriceUSDPerHour, 64); perr == nil {
			price = ocpi.Price{SKU: pos.Spec.SKU, USDPerHour: prev}
		} else {
			r.Recorder.Eventf(&pos, "Warning", "PriceUnavailable", "no OCPI price for %s: %v", pos.Spec.SKU, err)
			return ctrl.Result{RequeueAfter: reconcileInterval}, nil
		}
	} else if price.Stale(priceMaxAge) {
		priceStale = true
	}

	res := hedge.Compute(hedge.Inputs{
		GPUCount:       pos.Spec.GPUCount,
		UtilizationPct: int32(util + 0.5),
		SpotPriceUSD:   price.USDPerHour,
		HedgedPriceUSD: hedgedPrice,
	})

	availableForSublet := r.evaluateIdle(&pos, util)
	recommendation := buildRecommendation(res, priceStale)

	if err := r.applyOptionalAction(ctx, &pos, price.USDPerHour); err != nil {
		l.Info("optional action failed", "error", err.Error())
	}

	publishMetrics(&pos, util, price.USDPerHour, res, availableForSublet)
	r.writeStatus(&pos, util, price.USDPerHour, res, availableForSublet, recommendation, priceStale)

	if err := r.Status().Update(ctx, &pos); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// evaluateIdle applies a sustained-window plus hysteresis idle detection and returns
// whether the position should be flagged available for the secondary market.
func (r *ComputePositionReconciler) evaluateIdle(pos *computev1alpha1.ComputePosition, util float64) bool {
	threshold := float64(orDefaultInt32(pos.Spec.IdleThresholdPct, defaultIdleThresholdPct))
	window := time.Duration(orDefaultInt32(pos.Spec.IdleWindowSeconds, defaultIdleWindowSecs)) * time.Second
	now := time.Now()

	if util < threshold {
		if pos.Status.IdleSince == nil {
			t := metav1.NewTime(now)
			pos.Status.IdleSince = &t
		}
		sustained := now.Sub(pos.Status.IdleSince.Time) >= window
		if sustained && !pos.Status.AvailableForSublet {
			r.Recorder.Eventf(pos, "Normal", "IdleCapacityAvailable",
				"utilization %.0f%% below %.0f%% for %s; %d GPUs flagged for the secondary market",
				util, threshold, window, pos.Spec.GPUCount)
		}
		return sustained
	}

	// Hysteresis: only clear the idle state once utilization recovers past a margin.
	if util > threshold+idleResetMarginPct {
		pos.Status.IdleSince = nil
		return false
	}
	return pos.Status.AvailableForSublet
}

// applyOptionalAction pauses or resumes the referenced batch workload, but only when
// the position opts in via EnableActions. Critical workloads are never touched.
func (r *ComputePositionReconciler) applyOptionalAction(ctx context.Context, pos *computev1alpha1.ComputePosition, spot float64) error {
	if !pos.Spec.EnableActions || pos.Spec.WorkloadRef == nil {
		return nil
	}
	if pos.Spec.Priority == computev1alpha1.PriorityCritical {
		return nil
	}
	maxSpot, err := strconv.ParseFloat(pos.Spec.MaxSpotPriceUSDPerHour, 64)
	if err != nil || maxSpot <= 0 {
		return nil
	}

	ref := types.NamespacedName{Namespace: pos.Spec.WorkloadRef.Namespace, Name: pos.Spec.WorkloadRef.Name}
	var dep appsv1.Deployment
	if err := r.Get(ctx, ref, &dep); err != nil {
		return err
	}

	switch {
	case spot > maxSpot && !pos.Status.Paused:
		current := int32(1)
		if dep.Spec.Replicas != nil {
			current = *dep.Spec.Replicas
		}
		pos.Status.OriginalReplicas = &current
		zero := int32(0)
		dep.Spec.Replicas = &zero
		if err := r.Update(ctx, &dep); err != nil {
			return err
		}
		pos.Status.Paused = true
		r.Recorder.Eventf(pos, "Normal", "PausedOnPriceSpike",
			"spot %.2f exceeded max %.2f; scaled %s to 0 (opt-in action)", spot, maxSpot, ref.Name)

	case spot <= maxSpot*0.98 && pos.Status.Paused:
		restore := int32(1)
		if pos.Status.OriginalReplicas != nil {
			restore = *pos.Status.OriginalReplicas
		}
		dep.Spec.Replicas = &restore
		if err := r.Update(ctx, &dep); err != nil {
			return err
		}
		pos.Status.Paused = false
		pos.Status.OriginalReplicas = nil
		r.Recorder.Eventf(pos, "Normal", "ResumedOnPriceRecovery",
			"spot %.2f recovered below max %.2f; restored %s to %d", spot, maxSpot, ref.Name, restore)
	}
	return nil
}

func (r *ComputePositionReconciler) writeStatus(
	pos *computev1alpha1.ComputePosition,
	util, spot float64,
	res hedge.Result,
	availableForSublet bool,
	recommendation string,
	priceStale bool,
) {
	now := metav1.Now()
	pos.Status.UtilizationPct = int32(util + 0.5)
	pos.Status.SpotPriceUSDPerHour = money(spot)
	pos.Status.SpotCostUSDPerHour = money(res.SpotCostUSDPerHour)
	pos.Status.HedgePnLUSDPerHour = money(res.HedgePnLUSDPerHour)
	pos.Status.HedgeEffectivenessPct = int32(res.HedgeEffectivenessPct + 0.5)
	pos.Status.BasisRiskUSDPerHour = money(res.BasisRiskUSDPerHour)
	pos.Status.IdleGPUCount = int32(res.IdleGPUs + 0.5)
	pos.Status.AvailableForSublet = availableForSublet
	pos.Status.Recommendation = recommendation
	pos.Status.PriceStale = priceStale
	pos.Status.LastUpdated = &now

	switch {
	case pos.Status.Paused:
		pos.Status.Phase = "Paused"
	case availableForSublet:
		pos.Status.Phase = "IdleAvailable"
	default:
		pos.Status.Phase = "Active"
	}
}

func publishMetrics(pos *computev1alpha1.ComputePosition, util, spot float64, res hedge.Result, sublet bool) {
	lv := []string{pos.Name, pos.Spec.SKU}
	metrics.Utilization.WithLabelValues(lv...).Set(util)
	metrics.SpotPrice.WithLabelValues(lv...).Set(spot)
	metrics.HedgePnL.WithLabelValues(lv...).Set(res.HedgePnLUSDPerHour)
	metrics.BasisRisk.WithLabelValues(lv...).Set(res.BasisRiskUSDPerHour)
	metrics.HedgeEffectiveness.WithLabelValues(lv...).Set(res.HedgeEffectivenessPct)
	metrics.IdleGPUs.WithLabelValues(lv...).Set(res.IdleGPUs)
	metrics.AvailableForSublet.WithLabelValues(lv...).Set(boolToFloat(sublet))
}

func buildRecommendation(res hedge.Result, priceStale bool) string {
	if priceStale {
		return "price feed stale; holding last known value, no action taken"
	}
	if res.BasisRiskUSDPerHour >= 1 {
		return fmt.Sprintf("idle capacity leaking %s/hr; sublet %.0f GPUs or reduce hedge notional",
			money(res.BasisRiskUSDPerHour), res.IdleGPUs)
	}
	return "hedge well matched to real utilization"
}

// SetupWithManager wires the controller. GenerationChangedPredicate keeps status writes
// from retriggering reconciles; periodic requeue drives the polling of price and utilization.
func (r *ComputePositionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1alpha1.ComputePosition{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("computeposition").
		Complete(r)
}

func money(v float64) string { return fmt.Sprintf("%.2f", v) }

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func orDefaultInt32(v, def int32) int32 {
	if v == 0 {
		return def
	}
	return v
}
