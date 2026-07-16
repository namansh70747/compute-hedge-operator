# compute-hedge-operator

A Kubernetes operator that keeps a hedged block of GPU capacity honest against the
market. It reads the live OCPI spot price and real GPU utilization, then continuously
answers three questions for every position:

- How much of my hedge is actually backed by real usage, and how much is drifting into **basis risk**?
- What is my hedge **P&L** right now, marked against the index?
- Do I have **idle capacity** that should be offered back to the secondary market?

It is advisory by default. It never touches a workload unless a position explicitly opts in.

## Why this exists

Ornn turns GPU compute into a tradable asset: the OCPI index, cash-settled futures, and a
marketplace for capacity. Those instruments settle against an **index**, but a customer's
real exposure is their **own utilization on their own SKUs**. When utilization drops, the
hedge no longer matches reality and the gap leaks money. That gap is basis risk, and it
lives inside the Kubernetes cluster where the GPUs actually run, which is exactly where a
financial desk cannot see it.

This operator sits in that cluster and turns raw telemetry into position-level economics:
basis risk, hedge effectiveness, live P&L, and idle capacity that can be routed back to
Ornn's marketplace to earn fees instead of sitting stranded.

## What it does

- **Idle-capacity detection (primary).** Sustained low utilization on a position flags GPUs
  as available for the secondary market, with hysteresis so the flag does not thrash.
- **Basis-risk and hedge-effectiveness accounting.** A small, unit-tested model derives the
  per-hour economics from utilization and price. At full utilization the net result equals
  the locked hedge target; below full it exposes exactly how much is leaking.
- **Optional, opt-in actions.** A batch position can opt in to pausing its workload on a
  sustained price spike, and resuming when the price recovers. Off by default. Critical
  workloads are never touched.

## Architecture

```
  .env / K8s Secret (optional) --> internal/config resolver (auto | mock | live)
                                          |
ComputePosition (CRD)                     | selects each source
        |                                 v
        v          price     --> mock OCPI service   | real Ornn Data API (token)
  operator         telemetry --> mock gpuexporter    | Prometheus / DCGM (PROMETHEUS_URL)
  (controller-     market    --> log-only (advisory) | Ornn marketplace (MARKET_API_URL)
   runtime)
        |
        +--> status + Kubernetes Events
        +--> Prometheus metrics --> Grafana dashboard + alert rules
        +--> optional pause/resume of an opted-in Deployment
        +--> posts idle capacity to the marketplace when sublet flips

  console (read-only) --reads--> ComputePositions + Events + prices + resolved modes
        |
        +--> live "trading desk" web UI with a SIMULATED/LIVE badge  (headline surface)
```

With no configuration every source is the bundled simulator. The moment real Ornn
credentials appear (in `.env` locally, or a Kubernetes Secret in-cluster) each source
auto-switches to live and the console badge flips from **SIMULATED DATA** to **LIVE**. No
code change is required.

- `cmd/operator` — the controller.
- `cmd/console` — a read-only live control-room web UI (embedded React SPA + small Go API) that reads positions, events, and prices from the cluster. The headline demo surface.
- `cmd/mockocpi` — a local OCPI price service (mean-reverting prices, spike endpoint for demos).
- `cmd/gpuexporter` — simulated per-position GPU utilization (dcgm-style), controllable for demos.
- `internal/hedge` — the basis-risk / hedge-effectiveness math (unit tested).
- `internal/config` — dependency-free `.env` loader, mode resolution (`auto|mock|live`), and the factories that build each source.
- `internal/ocpi` — pluggable price source: mock HTTP service or the real Ornn Data API.
- `internal/telemetry` — utilization source: mock exporter or live Prometheus/DCGM.
- `internal/market` — marketplace publisher: log-only (advisory) or live HTTP write-back.
- `internal/console` — state aggregation and rolling history for the console API.
- `internal/metrics` — Prometheus series.

## Surfaces

- **Compute Hedge Console (headline)** — a bespoke, dark control-room UI: portfolio tiles,
  a live OCPI ticker, per-position cards with utilization rings, hedge-effectiveness gauges
  and basis-risk sparklines, and a live event feed. Read-only; served from the cluster.
- **Grafana (engineer's drill-down)** — the same signals as Prometheus time series, for
  deeper inspection and alert rules.

## Quickstart (local, free)

Requirements: Docker, kind, kubectl, Go 1.26+. See `docs/PREREQS.md`.

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

The console is at http://localhost:8090.

For the engineer's drill-down, port-forward Grafana/Prometheus:

```bash
kubectl -n compute-hedge-system port-forward svc/grafana 3000:3000
kubectl -n compute-hedge-system port-forward svc/prometheus 9090:9090
```

Grafana is login-free at http://localhost:3000 and lands directly on the
"Compute Hedge Operator" dashboard.

Watch positions:

```bash
kubectl get computepositions -A -w
```

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

## Go live in 60 seconds

Every external dependency is pluggable. Blank config runs the mock prototype; real
credentials flip it to live with no code change.

In-cluster (the demo cluster is already running):

```bash
cp .env.example .env         # fill in ORNN_API_TOKEN (and optionally PROMETHEUS_URL / MARKET_API_URL)
make live                    # builds the ornn-credentials Secret from .env and restarts both pods
```

Watch the console header flip from **SIMULATED DATA** to **LIVE - api.ornnai.com**, and the
provenance strip show the real endpoints. Roll back any time with `make mock`.

Local dev (no cluster): `make run-operator` and `make run-console` both read `.env` from the
working directory automatically.

### Environment knobs

Each source resolves independently: `auto` = live when its credentials/URL are present,
otherwise mock. `mock` and `live` force the choice. Full reference in `.env.example`.

| Variable | Purpose | Blank (mock) → filled (live) |
| --- | --- | --- |
| `OCPI_MODE` / `TELEMETRY_MODE` / `MARKET_MODE` | per-source mode | `auto` |
| `ORNN_API_TOKEN` | Ornn Data API token | mock prices → live OCPI index |
| `ORNN_API_BASE_URL`, `ORNN_API_PRICE_PATH` | live price endpoint | — |
| `PROMETHEUS_URL` | Prometheus base URL | mock exporter → live DCGM |
| `TELEMETRY_QUERY` | PromQL template (`{position}`, `{namespace}`) | `avg(DCGM_FI_DEV_GPU_UTIL{position="{position}"})` |
| `MARKET_API_URL`, `MARKET_API_PATH` | marketplace endpoint | advisory → live supply posting |
| `MARKET_WRITE_ENABLED` | safety switch for posting | `false`; must be `true` to post |
| `AUTH_SCHEME`, `AUTH_HEADER` | shared auth for every live client | `Bearer`, `Authorization` |

Marketplace write-back never posts anything unless both `MARKET_API_URL` is set and
`MARKET_WRITE_ENABLED=true`. Live adapters degrade gracefully to last-known/mock on error,
so a flaky endpoint never hard-fails the demo.

## Docs

- `docs/PITCH.md` — the problem, the value, and the trade-offs.
- `docs/DEMO_SCRIPT.md` — the live walkthrough.
- `docs/QA.md` — anticipated questions and answers.
- `docs/PREREQS.md` — prerequisites and cost.
