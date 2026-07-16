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

- Real prices: set `OCPI_MODE=ornn` and provide an Ornn Data subscription token. The client
  in `internal/ocpi/ornndata.go` reads the OCPI index directly.
- Real utilization: point the telemetry source at NVIDIA `dcgm-exporter` metrics instead of
  the bundled simulator. The interface is unchanged.
