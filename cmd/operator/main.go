// Command operator runs the ComputePosition controller.
package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	computev1alpha1 "github.com/namansh70747/compute-hedge-operator/api/v1alpha1"
	"github.com/namansh70747/compute-hedge-operator/internal/config"
	"github.com/namansh70747/compute-hedge-operator/internal/controller"
	"github.com/namansh70747/compute-hedge-operator/internal/metrics"
)

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = computev1alpha1.AddToScheme(scheme)
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("setup")

	cfg := config.Load()
	src := cfg.Sources()
	setupLog.Info("resolved data sources",
		"price", src.Price.Mode, "priceLabel", src.Price.Label,
		"telemetry", src.Telemetry.Mode, "telemetryLabel", src.Telemetry.Label,
		"market", src.Market.Mode, "marketLabel", src.Market.Label)

	metrics.Register()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: envDefault("METRICS_ADDR", ":8082")},
		HealthProbeBindAddress: envDefault("HEALTH_ADDR", ":8083"),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	reconciler := &controller.ComputePositionReconciler{
		Client:    mgr.GetClient(),
		Recorder:  mgr.GetEventRecorderFor("computeposition-controller"),
		OCPI:      cfg.BuildOCPISource(),
		Telemetry: cfg.BuildTelemetrySource(),
		Market:    cfg.BuildMarketPublisher(),
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager exited with error")
		os.Exit(1)
	}
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
