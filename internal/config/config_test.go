package config

import (
	"os"
	"testing"

	"github.com/namansh70747/compute-hedge-operator/internal/market"
	"github.com/namansh70747/compute-hedge-operator/internal/ocpi"
	"github.com/namansh70747/compute-hedge-operator/internal/telemetry"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name  string
		mode  Mode
		creds bool
		want  bool
	}{
		{"auto with creds", ModeAuto, true, true},
		{"auto without creds", ModeAuto, false, false},
		{"mock with creds", ModeMock, true, false},
		{"mock without creds", ModeMock, false, false},
		{"live with creds", ModeLive, true, true},
		{"live without creds", ModeLive, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.mode, tc.creds); got != tc.want {
				t.Fatalf("resolve(%s, %v) = %v, want %v", tc.mode, tc.creds, got, tc.want)
			}
		})
	}
}

func TestParseMode(t *testing.T) {
	if parseMode("ornn") != ModeAuto {
		t.Fatalf("unknown mode must fall back to auto, got %s", parseMode("ornn"))
	}
	if parseMode("LIVE") != ModeLive {
		t.Fatalf("LIVE should parse as live")
	}
	if parseMode("mock") != ModeMock {
		t.Fatalf("mock should parse as mock")
	}
}

func TestPriceLiveAndSources(t *testing.T) {
	t.Setenv("DOTENV_PATH", "nonexistent.env")
	clearEnv(t,
		"ORNN_API_TOKEN", "PROMETHEUS_URL", "MARKET_API_URL", "MARKET_WRITE_ENABLED",
		"OCPI_MODE", "TELEMETRY_MODE", "MARKET_MODE",
	)

	t.Setenv("OCPI_MODE", "auto")
	t.Setenv("TELEMETRY_MODE", "auto")
	t.Setenv("MARKET_MODE", "auto")

	cfg := Load()
	if cfg.priceLive() {
		t.Fatal("price should be mock without token")
	}
	if cfg.telemetryLive() {
		t.Fatal("telemetry should be mock without prometheus URL")
	}
	if cfg.marketLive() {
		t.Fatal("market should not be live without URL and write flag")
	}
	src := cfg.Sources()
	if src.Price.Mode != "mock" || src.Price.Label != simulated {
		t.Fatalf("price sources = %+v", src.Price)
	}

	t.Setenv("ORNN_API_TOKEN", "tok")
	t.Setenv("PROMETHEUS_URL", "http://prom:9090")
	t.Setenv("MARKET_API_URL", "http://market")
	t.Setenv("MARKET_WRITE_ENABLED", "false")
	cfg = Load()
	if !cfg.priceLive() {
		t.Fatal("price should be live with token")
	}
	if !cfg.telemetryLive() {
		t.Fatal("telemetry should be live with prometheus URL")
	}
	if cfg.marketLive() {
		t.Fatal("market must stay off when write disabled")
	}

	t.Setenv("MARKET_WRITE_ENABLED", "true")
	cfg = Load()
	if !cfg.marketLive() {
		t.Fatal("market should be live with URL and write enabled")
	}
	src = cfg.Sources()
	if src.Price.Mode != "live" || src.Telemetry.Mode != "live" || src.Market.Mode != "live" {
		t.Fatalf("expected all live, got %+v", src)
	}
}

func TestBuildFactoriesTypes(t *testing.T) {
	t.Setenv("DOTENV_PATH", "nonexistent.env")
	clearEnv(t,
		"ORNN_API_TOKEN", "PROMETHEUS_URL", "MARKET_API_URL", "MARKET_WRITE_ENABLED",
		"OCPI_MODE", "TELEMETRY_MODE", "MARKET_MODE",
	)
	t.Setenv("OCPI_MODE", "mock")
	t.Setenv("TELEMETRY_MODE", "mock")
	t.Setenv("MARKET_MODE", "mock")
	cfg := Load()

	if _, ok := cfg.BuildOCPISource().(*ocpi.HTTPSource); !ok {
		t.Fatal("mock OCPI should be HTTPSource")
	}
	if _, ok := cfg.BuildTelemetrySource().(*telemetry.HTTPSource); !ok {
		t.Fatal("mock telemetry should be HTTPSource")
	}
	if _, ok := cfg.BuildMarketPublisher().(*market.LogPublisher); !ok {
		t.Fatal("mock market should be LogPublisher")
	}

	t.Setenv("OCPI_MODE", "live")
	t.Setenv("ORNN_API_TOKEN", "tok")
	t.Setenv("TELEMETRY_MODE", "live")
	t.Setenv("PROMETHEUS_URL", "http://prom:9090")
	t.Setenv("MARKET_MODE", "live")
	t.Setenv("MARKET_API_URL", "http://market")
	t.Setenv("MARKET_WRITE_ENABLED", "true")
	cfg = Load()

	if _, ok := cfg.BuildOCPISource().(*ocpi.OrnnDataSource); !ok {
		t.Fatal("live OCPI should be OrnnDataSource")
	}
	if _, ok := cfg.BuildTelemetrySource().(*telemetry.PrometheusSource); !ok {
		t.Fatal("live telemetry should be PrometheusSource")
	}
	if _, ok := cfg.BuildMarketPublisher().(*market.HTTPPublisher); !ok {
		t.Fatal("live market should be HTTPPublisher")
	}
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}
