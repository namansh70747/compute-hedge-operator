# Prerequisites and cost

## Path A — Local demo (mock, free)

Everything here runs locally and is free. No cloud account, no GPUs, and no paid data
feed are required.

| Tool | Version | Purpose |
| --- | --- | --- |
| Go | 1.26+ | build binaries and run tests |
| Docker | recent | build the image, back the kind cluster |
| kind | 0.32+ | local Kubernetes cluster in Docker |
| kubectl | 1.30+ | apply manifests, port-forward |
| Make or PowerShell | — | `make demo` or `scripts/demo.ps1` |

Cost: none for the demo. Bundled `mockocpi` and `gpuexporter` replace paid feeds and GPUs.
Prometheus and Grafana run as single local pods. You need a machine that can run a one-node
kind cluster.

## Path B — Install on an existing cluster

| Tool | Version | Purpose |
| --- | --- | --- |
| kubectl | 1.30+ | apply / helm install |
| Helm | 3+ | install the operator chart |
| Docker (optional) | recent | only if you build and push your own image |
| Cluster access | — | a Kubernetes cluster you control |

For **live** utilization the cluster should already run NVIDIA GPU Operator (or equivalent)
and scrape `dcgm-exporter` into Prometheus. This project does not install those for you.

For **live** prices you need an Ornn Data (or compatible) API token and a reachable base URL.
Adapt `ORNN_API_PRICE_PATH` to the subscribed feed; the default template is
`/v1/ocpi/{sku}/spot`.

For **marketplace write-back** you need an HTTP endpoint that accepts this repo's offer JSON
(and you must set `MARKET_WRITE_ENABLED=true`). There is no claimed public Ornn supply API in
this repository.

## Going live (either path)

Copy `.env.example` to `.env` (local) or apply `config/secret.example.yaml` as
`ornn-credentials` (in-cluster). With `*_MODE=auto`, filled credentials flip each source to
live and the console badge to **LIVE**. Blank credentials keep the simulator.

See the README Path A / Path B sections and `.env.example` for the full knob reference.
