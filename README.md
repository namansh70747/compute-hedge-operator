# compute-hedge-operator

A Kubernetes operator that keeps a hedged block of GPU capacity honest against the
market. It reads the OCPI spot price and GPU utilization, then continuously answers
three questions for every position:

- How much of my hedge is actually backed by real usage, and how much is drifting into **basis risk**?
- What is my hedge **P&L** right now, marked against the index?
- Do I have **idle capacity** that should be offered back to the secondary market?

It is advisory by default. It never touches a workload unless a position explicitly opts in.

**License: MIT** (see [LICENSE](LICENSE)).

## One build, two realities

| | No credentials (demo) | Credentials present (real) |
| --- | --- | --- |
| Price | Bundled `mockocpi` simulator | Ornn Data API (`ORNN_API_TOKEN`) |
| GPU utilization | Bundled `gpuexporter` | Prometheus / DCGM (`PROMETHEUS_URL`) |
| Marketplace | Log-only (advisory) | HTTP POST when `MARKET_WRITE_ENABLED=true` |
| Console badge | **SIMULATED DATA** | **LIVE** (same sources as the operator) |

Same image, same code. Supply a `.env` or the `ornn-credentials` Secret and every surface
flips to live with no code change. Live HTTP paths and JSON shapes are **configurable
adapters** — set `ORNN_API_PRICE_PATH` (must include `{sku}` if per-SKU) to match whatever
endpoint you are given. This repo does not claim a public Ornn marketplace contract;
write-back is off unless you set a URL and the write flag.

## Path A — Local demo (mock, free)

Requirements: Docker, kind, kubectl, Go 1.26+. See [docs/PREREQS.md](docs/PREREQS.md).

```powershell
pwsh -File scripts/demo.ps1
```

or, on Linux/macOS:

```bash
make demo
```

Then open the headline console:

```bash
kubectl -n compute-hedge-system port-forward svc/console 8090:8090
```

Console: http://localhost:8090 (badge shows **SIMULATED DATA**).

Optional Grafana / Prometheus:

```bash
kubectl -n compute-hedge-system port-forward svc/grafana 3000:3000
kubectl -n compute-hedge-system port-forward svc/prometheus 9090:9090
```

Watch positions:

```bash
kubectl get computepositions -A -w
```

## Path B — Install on your Kubernetes cluster

Installs the **operator only** (CRD + RBAC + Deployment + metrics Service). Mocks and the
console are not required. For live GPU utilization your cluster should already scrape
NVIDIA `dcgm-exporter` into Prometheus.

### Using a published image (GHCR)

Images are published to GHCR on version tags (`v*`). First publish needs a GitHub token with
`workflow` and `packages` scopes when pushing [`.github/workflows/release.yaml`](.github/workflows/release.yaml).

```bash
helm upgrade --install compute-hedge charts/compute-hedge-operator \
  --namespace compute-hedge-system --create-namespace \
  --set image.repository=ghcr.io/namansh70747/compute-hedge-operator \
  --set image.tag=0.1.0
```

Or install a packaged chart from a [GitHub Release](https://github.com/namansh70747/compute-hedge-operator/releases) `.tgz`.

### Build-from-source fallback

```bash
docker build -t <your-registry>/compute-hedge-operator:0.1.0 .
docker push <your-registry>/compute-hedge-operator:0.1.0

helm upgrade --install compute-hedge charts/compute-hedge-operator \
  --namespace compute-hedge-system --create-namespace \
  --set image.repository=<your-registry>/compute-hedge-operator \
  --set image.tag=0.1.0
```

Raw manifests (same idea): edit the image in `config/manager.yaml`, then
`kubectl apply -f config/crd/computepositions.yaml -f config/rbac.yaml -f config/manager.yaml`.

### Go live on Path B

```bash
cp config/secret.example.yaml config/secret.yaml
# fill ORNN_API_TOKEN, PROMETHEUS_URL, optional MARKET_* …
kubectl -n compute-hedge-system apply -f config/secret.yaml
kubectl -n compute-hedge-system rollout restart deploy/compute-hedge-operator
```

Or create the Secret via Helm:

```bash
helm upgrade --install compute-hedge charts/compute-hedge-operator \
  --namespace compute-hedge-system --create-namespace \
  --set credentials.create=true \
  --set credentials.ornnApiToken=<token> \
  --set credentials.prometheusUrl=http://prometheus.monitoring.svc:9090
```

Apply positions:

```bash
kubectl apply -f deploy/samples/computepositions.yaml
# or your own ComputePosition YAML
kubectl get computepositions -A -w
```

Optional console on the same image:

```bash
# set image in deploy/console.yaml to match your registry/tag, then:
kubectl apply -f deploy/console.yaml
kubectl -n compute-hedge-system port-forward svc/console 8090:8090
```

## Architecture

```
  .env / K8s Secret (optional) --> internal/config resolver (auto | mock | live)
                                          |
ComputePosition (CRD)                     | selects each source
        |                                 v
        v          price     --> mock OCPI service   | real Ornn Data API (token)
  operator         telemetry --> mock gpuexporter    | Prometheus / DCGM (PROMETHEUS_URL)
  (controller-     market    --> log-only (advisory) | marketplace HTTP (MARKET_API_URL)
   runtime)
        |
        +--> status + Kubernetes Events
        +--> Prometheus metrics --> Grafana (demo) / your scrape config
        +--> optional pause/resume of an opted-in Deployment
        +--> posts idle capacity when sublet flips (write-gated)

  console (read-only) --reads--> ComputePositions + Events + same price source as operator
        |
        +--> trading-desk UI with SIMULATED / LIVE badge
```

## Environment knobs

Each source resolves independently: `auto` = live when its credentials/URL are present,
otherwise mock. Full reference in [`.env.example`](.env.example).

| Variable | Purpose | Blank → filled |
| --- | --- | --- |
| `OCPI_MODE` / `TELEMETRY_MODE` / `MARKET_MODE` | per-source mode (`auto\|mock\|live`) | `auto` |
| `ORNN_API_TOKEN` | Ornn Data API token | mock prices → live OCPI |
| `ORNN_API_BASE_URL`, `ORNN_API_PRICE_PATH` | live price endpoint (`{sku}` substituted) | — |
| `PROMETHEUS_URL` | Prometheus base URL | mock exporter → live DCGM |
| `TELEMETRY_QUERY` | PromQL (`{position}`, `{namespace}`) | default DCGM avg query |
| `MARKET_API_URL`, `MARKET_API_PATH` | marketplace endpoint | advisory → live supply |
| `MARKET_WRITE_ENABLED` | safety switch | `false`; must be `true` to post |
| `AUTH_SCHEME`, `AUTH_HEADER` | shared auth for marketplace HTTP | `Bearer`, `Authorization` |

Marketplace write-back never posts unless both `MARKET_API_URL` is set and
`MARKET_WRITE_ENABLED=true`.

Local flip on the kind demo: `cp .env.example .env`, fill tokens, `make live` / `make mock`.

## Metrics

| Metric | Meaning |
| --- | --- |
| `chp_gpu_utilization_pct` | Observed utilization per position |
| `chp_spot_price_usd_per_gpu_hour` | OCPI price observed per position |
| `chp_hedge_pnl_usd_per_hour` | Mark-to-market hedge P&L |
| `chp_basis_risk_usd_per_hour` | Unmatched exposure between hedge and real utilization |
| `chp_hedge_effectiveness_pct` | Net economics as a share of the locked target |
| `chp_idle_gpu_count` | Idle GPUs in a position |
| `chp_available_for_sublet` | 1 when idle capacity is flagged for the secondary market |

## Security

Least-privilege RBAC: the operator can get/list/watch/update/patch Deployments (for optional
pause/resume) but **cannot create or delete** them. Pod runs non-root, read-only root
filesystem, all capabilities dropped. Distroless image.

## Docs

- [docs/PITCH.md](docs/PITCH.md) — problem, value, trade-offs
- [docs/DEMO_SCRIPT.md](docs/DEMO_SCRIPT.md) — live walkthrough
- [docs/QA.md](docs/QA.md) — anticipated questions
- [docs/PREREQS.md](docs/PREREQS.md) — tools for Path A vs Path B
- [docs/MASTERCLASS.md](docs/MASTERCLASS.md) — deep project walkthrough

## License

MIT © 2026 Naman Sharma. See [LICENSE](LICENSE).
