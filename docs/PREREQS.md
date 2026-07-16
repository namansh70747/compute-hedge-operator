# Prerequisites and cost

Everything here runs locally and is free. No cloud account, no GPUs, and no paid data
feed are required for the demo.

## Tools

| Tool | Version used | Purpose |
| --- | --- | --- |
| Go | 1.26+ | build the binaries and run tests |
| Docker | recent | build the image, back the kind cluster |
| kind | 0.32+ | local Kubernetes cluster in Docker |
| kubectl | 1.30+ | apply manifests, port-forward |
| Helm | 3+ (optional) | install the operator chart |

## Cost

- The demo uses a bundled mock OCPI price service and a simulated GPU utilization exporter,
  so there is no data-feed cost and no GPU cost.
- Prometheus and Grafana run as single local pods.
- The only requirement is a machine that can run a one-node kind cluster.

## Going beyond the demo

Every source is pluggable and auto-detected. Copy `.env.example` to `.env`, fill in what you
have, and run `make live` (in-cluster) or `make run-operator` / `make run-console` (local).
Nothing else changes.

- **Real prices:** set `ORNN_API_TOKEN` (with `ORNN_API_BASE_URL` / `ORNN_API_PRICE_PATH`).
  With `OCPI_MODE=auto` the presence of the token flips the price feed to the live OCPI
  index via `internal/ocpi/ornndata.go`.
- **Real utilization:** set `PROMETHEUS_URL` (optionally `TELEMETRY_QUERY`) to read NVIDIA
  `dcgm-exporter` metrics through Prometheus. The mock exporter is used when it is blank.
- **Marketplace write-back:** set `MARKET_API_URL` and `MARKET_WRITE_ENABLED=true` to post
  idle capacity as real supply. Off by default so nothing is ever posted by accident.
- **Any auth scheme:** `AUTH_SCHEME` / `AUTH_HEADER` adapt every live HTTP client to whatever
  token format Ornn provides.

See the full knob reference in `.env.example` and the "Go live in 60 seconds" section of the
README.
