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
ComputePosition (CRD)
        |
        v
  operator (controller-runtime)  --reads-->  OCPI price source (mock service | real Ornn Data API)
        |                        --reads-->  gpuexporter (dcgm-style utilization)
        |
        +--> status + Kubernetes Events
        +--> Prometheus metrics --> Grafana dashboard + alert rules
        +--> optional pause/resume of an opted-in Deployment
```

- `cmd/operator` — the controller.
- `cmd/mockocpi` — a local OCPI price service (mean-reverting prices, spike endpoint for demos).
- `cmd/gpuexporter` — simulated per-position GPU utilization (dcgm-style), controllable for demos.
- `internal/hedge` — the basis-risk / hedge-effectiveness math (unit tested).
- `internal/ocpi` — pluggable price source: mock HTTP service or the real Ornn Data API.
- `internal/telemetry` — utilization source.
- `internal/metrics` — Prometheus series.

## Quickstart (local, free)

Requirements: Docker, kind, kubectl, Go 1.26+. See `docs/PREREQS.md`.

```powershell
pwsh -File scripts/demo.ps1
```

or, on Linux/macOS:

```bash
make demo
```

Then port-forward the dashboards:

```bash
kubectl -n compute-hedge-system port-forward svc/grafana 3000:3000
kubectl -n compute-hedge-system port-forward svc/prometheus 9090:9090
```

Grafana is at http://localhost:3000 (dashboard: "Compute Hedge Operator").

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

## Real data

The price source is an interface. `scripts/demo.ps1` wires the bundled mock service so the
demo is deterministic and free. Set `OCPI_MODE=ornn` with an Ornn Data subscription token to
read the real OCPI index instead; no code changes are required.

## Docs

- `docs/PITCH.md` — the problem, the value, and the trade-offs.
- `docs/DEMO_SCRIPT.md` — the live walkthrough.
- `docs/QA.md` — anticipated questions and answers.
- `docs/PREREQS.md` — prerequisites and cost.
