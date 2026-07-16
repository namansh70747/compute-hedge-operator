# Demo script (about 20 minutes of the call)

The goal is to show, live, that idle GPUs and basis risk are measurable inside the cluster
and routable back to the marketplace. Keep the story first, the terminal second.

## Before the call

Bring the stack up once so nothing is cold:

```powershell
pwsh -File scripts/demo.ps1
```

Open four terminals:

1. `kubectl -n compute-hedge-system port-forward svc/console 8090:8090`
2. `kubectl -n compute-hedge-system port-forward svc/grafana 3000:3000`
3. `kubectl -n compute-hedge-system port-forward svc/prometheus 9090:9090`
4. `kubectl get computepositions -A -w`

Open the **console** at http://localhost:8090 — this is the headline surface you drive the
whole demo from. Keep Grafana (http://localhost:3000, lands directly on the "Compute Hedge
Operator" dashboard) and Prometheus alerts (http://localhost:9090/alerts) as the engineer's
drill-down.

## 1. Frame the problem (2 min, no terminal)

"Ornn hedges compute against the OCPI index. But a customer's real exposure is their own
utilization on their own SKUs. Those only match at full utilization. The gap is basis risk,
and it lives in the cluster, where the trading system cannot see it. I built the piece that
sees it."

## 2. Show the console (2 min)

Open http://localhost:8090. Walk the room top-down:

- The **portfolio bar**: hedged notional, live hedge P&L, aggregate basis risk, idle GPUs
  available for sublet, and reclaimable marketplace supply — each with a live sparkline.
- The **OCPI ticker**: spot price per SKU with up/down change.
- The **position cards**: utilization ring, hedge-effectiveness gauge, spot-vs-hedged,
  P&L, and a basis-risk trend. The plain-English `recommendation` shows at the bottom.
- The **live event feed** on the right: this is where controller actions will appear.

The same data is authoritative in the cluster if anyone wants proof:

```bash
kubectl get computepositions -A
kubectl describe computeposition training-cluster
```

## 3. Idle capacity into marketplace supply (5 min)

Force the batch position idle:

```powershell
pwsh -File scripts/setutil.ps1 -Position batch-render -Util 3
```

Watch on the console, in order:

- The `batch-render` card's utilization ring drops toward zero.
- After the idle window, the phase badge flips to a glowing amber `Idle · available`, the
  card shows idle GPUs flagged for sublet, and the portfolio "Idle GPUs" / "Marketplace
  supply" tiles tick up.
- An `IdleCapacityAvailable` event appears in the live feed (and the
  `IdleCapacityAvailable` alert fires in Prometheus for the drill-down).

Say: "That flag is marketplace supply. Every idle block detected here is inventory Ornn can
match and earn a fee on."

Bring it back:

```powershell
pwsh -File scripts/setutil.ps1 -Position batch-render -Util -1
```

The hysteresis means it only clears once utilization recovers past a margin, so it does not
flicker.

## 4. Basis risk and hedge P&L on a price move (6 min)

Spike the H200 price:

```powershell
pwsh -File scripts/spike.ps1 -Sku H200 -Fraction 0.8
```

Watch the console:

- The OCPI ticker for H200 jumps green, then decays back.
- Hedge P&L on the affected cards and the portfolio tile moves as spot rises.
- Basis risk widens if utilization is below full, because the idle GPUs are now valued at a
  higher spot — the basis-risk sparkline visibly bows out.

Say: "At full utilization the net result is locked to the hedge target regardless of price.
The number you see moving is precisely the cost of the hedge not matching real usage."

## 5. Opt-in action (3 min)

`batch-render` opts in to pausing on a sustained spike (`enableActions: true`,
`maxSpotPriceUSDPerHour: "3.60"`). With the spike above the cap:

```bash
kubectl get deploy batch-render -n default -w
```

- The operator scales `batch-render` to zero, the console card flips to a red `Paused`
  badge, and a `PausedOnPriceSpike` event lands in the live feed.
- When the price decays below the cap, it restores replicas, the badge returns to `Active`,
  and a `ResumedOnPriceRecovery` event appears.

Say: "This is off by default and never applies to critical workloads. It is opt-in per
position, hysteresis-guarded, and every action is an auditable Event. The default posture is
advisory."

## 6. Close (2 min)

"Everything here is one command on a laptop, no GPUs and no paid feed. The price source is an
interface: flip `OCPI_MODE=ornn` and it reads the real index. The point is the seam -- turning
cluster telemetry into position economics and marketplace supply."

## Reset between runs

```powershell
pwsh -File scripts/teardown.ps1
```
