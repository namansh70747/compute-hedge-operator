# Pitch: closing the basis-risk gap inside the cluster

## The one-line idea

Ornn prices and hedges GPU compute at the market level. This operator measures what actually
happens to that hedge inside the customer's cluster, where utilization is real and the index
is not. It turns that gap into three numbers a desk can act on: basis risk, hedge P&L, and
idle capacity that can be sold back to the marketplace.

## The problem

Ornn's instruments settle against the OCPI index. A hedge is sized in index terms: so many
GPU-hours of a SKU at a locked price. But a customer's real economic exposure is their own
utilization on their own machines. Those two things are equal only when utilization is full.

The moment utilization drops, the hedge notional no longer matches the real exposure. The
customer is paying to hedge capacity they are not using, or is short capacity they are. That
mismatch is **basis risk**, and it is invisible from the trading side because it only exists
where the GPUs run: inside Kubernetes.

## The model

For each position the operator computes, per hour:

```
utilizedGPUs   = gpuCount * utilization
physicalRev    = spot * utilizedGPUs
hedgePnL       = (hedgedPrice - spot) * gpuCount
lockedTarget   = hedgedPrice * gpuCount
net            = physicalRev + hedgePnL
basisRisk      = lockedTarget - net = spot * (gpuCount - utilizedGPUs)
```

The identity that makes the pitch land: **at full utilization, net equals the locked target
for any price.** That is what a perfect hedge is supposed to do. Below full utilization, the
difference is exactly `spot * idleGPUs` -- the money leaking out of an imperfect hedge. The
operator surfaces that number live, per position, and flags the idle GPUs behind it.

## The value to Ornn

1. **A liquidity feed for the marketplace.** Idle capacity detected in customer clusters is
   the raw supply Ornn's secondary market needs. Every flagged block is a potential trade and
   a potential fee. The operator turns stranded GPUs into marketplace inventory.
2. **A reason to hedge more, and more accurately.** When a customer can see their basis risk
   in dollars per hour, the natural response is to adjust their hedge -- which is Ornn's
   product. The operator makes the value of hedging legible at the cluster level.
3. **Distribution into the cluster.** Ornn's surfaces are the terminal and the API. This puts
   an Ornn-aware component next to the workloads, reading the OCPI index through the same API
   a subscription already provides.

## Why Ornn may not have built this yet

- Ornn's core competence is markets and data, not running controllers inside other people's
  clusters. That is a different operational and trust surface.
- Anything that can change a customer's workloads is a security and liability question. The
  safe version is advisory-first, and that framing is a product decision, not an afterthought.
- The value shows up at the seam between the trading system and the cluster. It is easy to
  miss from either side alone.

This is why the operator is deliberately advisory by default, customer-owned, and least-
privilege. The actions are opt-in per position and never apply to critical workloads.

## Trade-offs and risks (stated plainly)

- **Simulated inputs in the demo.** Prices and utilization are simulated so the demo is free
  and deterministic. Both are behind interfaces; real OCPI and real dcgm-exporter drop in
  without code changes. This is a demo choice, not a design limit.
- **The model is intentionally simple.** It captures quantity basis risk valued at spot. It
  does not model volatility term structure, funding, or price basis between a customer's
  realized rate and the index. Those are natural extensions, not present today.
- **Actions carry risk.** Pausing a workload is powerful and dangerous. That is why it is off
  by default, opt-in, hysteresis-guarded, and blocked for critical positions.
- **Trust boundary.** A third-party operator in a customer cluster must be least-privilege and
  auditable. The RBAC is scoped to exactly the verbs it needs, and every action emits an Event.

## What is built

A working operator with a CRD, a unit-tested economics model, a live Grafana dashboard,
Prometheus alert rules, a Helm chart, least-privilege RBAC, and CI. It runs end to end on a
local kind cluster with one command.
