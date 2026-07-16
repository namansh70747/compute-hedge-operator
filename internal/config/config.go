// Package config centralizes all runtime configuration and selects mock or live data
// sources. The guiding rule: with no credentials it runs the mock prototype; the moment
// real Ornn credentials/URLs are present, each source auto-switches to live. No code
// change is required to go live.
package config

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/namansh70747/compute-hedge-operator/internal/market"
	"github.com/namansh70747/compute-hedge-operator/internal/ocpi"
	"github.com/namansh70747/compute-hedge-operator/internal/telemetry"
)

// Mode selects how a data source is chosen.
type Mode string

const (
	// ModeAuto picks live when credentials are present, otherwise mock.
	ModeAuto Mode = "auto"
	// ModeMock forces the bundled simulator.
	ModeMock Mode = "mock"
	// ModeLive forces the real integration.
	ModeLive Mode = "live"
)

const simulated = "simulated"

// SourceInfo describes the resolved mode and a human label for a single data source.
type SourceInfo struct {
	Mode  string `json:"mode"`  // "mock" or "live"
	Label string `json:"label"` // endpoint host when live, otherwise "simulated"
}

// Sources is the provenance summary surfaced to the console and logs.
type Sources struct {
	Price     SourceInfo `json:"price"`
	Telemetry SourceInfo `json:"telemetry"`
	Market    SourceInfo `json:"market"`
}

// Config is the fully-resolved runtime configuration.
type Config struct {
	Cluster      string
	PollInterval time.Duration

	OCPIMode      Mode
	OCPIMockURL   string
	OrnnBaseURL   string
	OrnnToken     string
	OrnnPricePath string

	TelemetryMode  Mode
	GPUExporterURL string
	PrometheusURL  string
	TelemetryQuery string

	MarketMode         Mode
	MarketURL          string
	MarketPath         string
	MarketWriteEnabled bool

	AuthScheme string
	AuthHeader string
}

// Load reads a .env file (if present) and then the environment into a Config.
func Load() Config {
	LoadDotEnv(envDefault("DOTENV_PATH", ".env"))
	return Config{
		Cluster:      envDefault("CLUSTER_NAME", "kind-compute-hedge"),
		PollInterval: envDuration("POLL_INTERVAL", 2*time.Second),

		OCPIMode:      parseMode(envDefault("OCPI_MODE", "auto")),
		OCPIMockURL:   envDefault("OCPI_URL", "http://mockocpi:8080"),
		OrnnBaseURL:   envDefault("ORNN_API_BASE_URL", "https://api.ornnai.com"),
		OrnnToken:     os.Getenv("ORNN_API_TOKEN"),
		OrnnPricePath: os.Getenv("ORNN_API_PRICE_PATH"),

		TelemetryMode:  parseMode(envDefault("TELEMETRY_MODE", "auto")),
		GPUExporterURL: envDefault("GPU_EXPORTER_URL", "http://gpuexporter:8081"),
		PrometheusURL:  os.Getenv("PROMETHEUS_URL"),
		TelemetryQuery: os.Getenv("TELEMETRY_QUERY"),

		MarketMode:         parseMode(envDefault("MARKET_MODE", "auto")),
		MarketURL:          os.Getenv("MARKET_API_URL"),
		MarketPath:         os.Getenv("MARKET_API_PATH"),
		MarketWriteEnabled: envBool("MARKET_WRITE_ENABLED", false),

		AuthScheme: envDefault("AUTH_SCHEME", "Bearer"),
		AuthHeader: envDefault("AUTH_HEADER", "Authorization"),
	}
}

// priceLive reports whether the live Ornn price feed should be used.
func (c Config) priceLive() bool { return resolve(c.OCPIMode, c.OrnnToken != "") }

// telemetryLive reports whether the live Prometheus telemetry source should be used.
func (c Config) telemetryLive() bool { return resolve(c.TelemetryMode, c.PrometheusURL != "") }

// marketLive reports whether offers are actually posted to a real marketplace.
func (c Config) marketLive() bool {
	return resolve(c.MarketMode, c.MarketURL != "") && c.MarketWriteEnabled && c.MarketURL != ""
}

// BuildOCPISource returns the live or mock price source.
func (c Config) BuildOCPISource() ocpi.Source {
	if c.priceLive() {
		return ocpi.NewOrnnDataSource(ocpi.OrnnDataConfig{
			BaseURL:   c.OrnnBaseURL,
			Token:     c.OrnnToken,
			PricePath: c.OrnnPricePath,
		})
	}
	return ocpi.NewHTTPSource(c.OCPIMockURL)
}

// BuildTelemetrySource returns the live or mock telemetry source.
func (c Config) BuildTelemetrySource() telemetry.Source {
	if c.telemetryLive() {
		return telemetry.NewPrometheusSource(telemetry.PrometheusConfig{
			BaseURL: c.PrometheusURL,
			Query:   c.TelemetryQuery,
		})
	}
	return telemetry.NewHTTPSource(c.GPUExporterURL)
}

// BuildMarketPublisher returns the live publisher only when writes are enabled and a URL
// is set, otherwise the no-op log publisher.
func (c Config) BuildMarketPublisher() market.Publisher {
	if c.marketLive() {
		return market.NewHTTPPublisher(market.HTTPConfig{
			BaseURL:      c.MarketURL,
			Path:         c.MarketPath,
			AuthHeader:   c.AuthHeader,
			AuthScheme:   c.AuthScheme,
			Token:        c.OrnnToken,
			WriteEnabled: c.MarketWriteEnabled,
		})
	}
	return market.NewLogPublisher()
}

// Sources summarizes the resolved provenance of each data source.
func (c Config) Sources() Sources {
	return Sources{
		Price:     info(c.priceLive(), c.OrnnBaseURL),
		Telemetry: info(c.telemetryLive(), c.PrometheusURL),
		Market:    info(c.marketLive(), c.MarketURL),
	}
}

func info(live bool, endpoint string) SourceInfo {
	if !live {
		return SourceInfo{Mode: "mock", Label: simulated}
	}
	return SourceInfo{Mode: "live", Label: hostLabel(endpoint)}
}

func hostLabel(raw string) string {
	if raw == "" {
		return "live"
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
}

func resolve(mode Mode, credsPresent bool) bool {
	switch mode {
	case ModeLive:
		return true
	case ModeMock:
		return false
	default:
		return credsPresent
	}
}

func parseMode(v string) Mode {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "live":
		return ModeLive
	case "mock":
		return ModeMock
	default:
		return ModeAuto
	}
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
