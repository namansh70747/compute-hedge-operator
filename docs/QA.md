# Anticipated questions

**Is this real data?**
The demo uses a bundled mock price service and a simulated utilization exporter so it is free
and deterministic. Both sit behind interfaces. Setting `OCPI_MODE=ornn` with a subscription
token reads the real OCPI index, and pointing the telemetry source at dcgm-exporter reads real
utilization. No code changes are needed.

**How is basis risk actually computed?**
Per hour: `basisRisk = spot * (gpuCount - utilizedGPUs)`. It is the quantity mismatch between
the hedge notional and real utilization, valued at the current spot price. At full utilization
it is zero and the net economic result equals the locked hedge target for any price. That
identity is unit tested in `internal/hedge`.

**Why an operator and not a dashboard or a script?**
The signal is reactive and lives next to the workloads. A controller reconciles continuously,
carries state safely across restarts in the CR status, emits Kubernetes Events, and exposes
metrics the same way the rest of the platform is observed. It is the native shape for this.

**Isn't running in a customer cluster a security risk?**
Yes, which is why it is advisory by default. It never changes a workload unless a position
opts in, never touches critical workloads, and its RBAC is scoped to exactly the verbs it
needs. Every action emits an auditable Event. The powerful capability is opt-in and guarded.

**What stops the sublet flag or the pause from flapping?**
Hysteresis. Idle requires sustained low utilization over a window before flagging, and only
clears once utilization recovers past a margin. The pause resumes only after the price falls a
margin below the cap. Status updates do not retrigger reconciles; a periodic requeue drives
polling.

**What happens if the price feed is down?**
The operator marks the price stale, holds the last known value, refuses to take actions, and
says so in the recommendation. It degrades to advisory and safe.

**How does this make Ornn money rather than just help a customer?**
Idle capacity detected here is supply for Ornn's secondary market; each block is a potential
matched trade and fee. Visible basis risk pushes customers to hedge more accurately, which is
Ornn's product. And it puts an Ornn-aware component in the cluster, reading the index through
the API a subscription already provides.

**What are the limits of the model?**
It captures quantity basis risk valued at spot. It does not yet model volatility term
structure, funding, or the price basis between a customer's realized rate and the index. Those
are additive extensions on top of the same interfaces.

**How would this scale to many positions and clusters?**
The controller is stateless beyond CR status and scrapes are cheap. Positions are namespaced
CRs; effectiveness and basis risk aggregate cleanly in Prometheus. A fleet view is a Grafana
query away.

**What was deliberately left out for the demo?**
Leader election and multi-replica HA, a real futures ledger, and persistence of historical
P&L. All are straightforward to add and none change the core argument.
