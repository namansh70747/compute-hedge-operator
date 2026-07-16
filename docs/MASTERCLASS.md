# Compute Hedge Operator — Complete Masterclass & Interview Guide

This is your single study + presentation document. It covers the whole project end to end,
maps it to the Ornn **Software Engineer (platform + observability + reliability)** role,
explains every file and every tool in depth, tells you exactly what to open and say in the
meeting, and lists the questions they may ask with strong answers.

Read it top to bottom once. Then use the "What to open" and "Q&A" sections as your script.

---

## 0. The 30-second pitch (memorize this)

> "Ornn is the financial layer for GPU compute: OCPI pricing, a marketplace, and hedging.
> But a hedge settles against an **index**, while a customer's real exposure is their own
> **GPU utilization inside their cluster**. When GPUs go idle, the hedge stops matching
> reality and money leaks — that gap is **basis risk**, and it lives in Kubernetes where a
> trading desk can't see it. I built a Kubernetes **operator** that sits in the customer's
> cluster, reads live OCPI price and real GPU utilization, marks per-position P&L and basis
> risk, and posts idle capacity back to Ornn's secondary marketplace. It runs fully on mock
> data today and flips to live Ornn data the moment credentials are provided — no code change."

---

## 1. Why this project exists (the business story)

### The problem
- Ornn sells **hedges/futures** that settle against the **OCPI index** (a market-wide GPU price).
- A customer's real world is **their own GPUs on their own SKUs** in their own cluster.
- When utilization drops, the hedge (sized for all GPUs) no longer matches real usage
  (only utilized GPUs earn). The mismatch leaks money = **basis risk**.
- That truth lives **inside Kubernetes**, invisible to a financial desk.

### What the operator does
1. **Idle-capacity detection (primary value).** Sustained low utilization flags GPUs as
   available for the secondary market, with **hysteresis** so the flag doesn't flap.
2. **Basis-risk & hedge-effectiveness accounting.** A small, unit-tested model turns
   utilization + price into per-hour economics.
3. **Optional, opt-in actions.** A batch position can pause its workload on a price spike
   and resume when price recovers. **Off by default. Critical workloads never touched.**

### Why Ornn cares
- Every idle block your operator detects is **supply for Ornn's secondary market** = a
  potential trade and a fee.
- It closes the loop between Ornn's **financial layer** and the **runtime layer** in the cluster.

---

## 2. Where this sits in Ornn's world (say this precisely)

```
ORNN (financial + marketplace layer — their product)
  Ornn Data      → OCPI (GPU $/hr), OTPI (token $/Mtok), memory prices
  Ornn Compute   → primary marketplace + secondary sublet market
  Ornn Exchange  → futures / derivatives / hedging (cash-settled, references OCPI)
        │  HTTP APIs (price, marketplace)
        ▼
CUSTOMER'S KUBERNETES CLUSTER (where GPUs actually run)
  NVIDIA GPU Operator  → drivers, device plugin (nvidia.com/gpu), DCGM exporter
  Scheduler            → places GPU workloads on the right nodes
  Training / batch pods → real GPU consumers
  Prometheus           → scrapes DCGM metrics
  ── YOUR compute-hedge-operator ──
     reads OCPI price + DCGM utilization → marks P&L/basis → posts idle to marketplace
```

Key distinctions to never mix up:
- **Ornn "operator"** = a data-center company that owns GPUs (business term).
- **NVIDIA GPU Operator** = NVIDIA's tool that installs drivers + device plugin + DCGM.
- **Your operator** = a Kubernetes controller you built (the reconciler).
- Ornn does **not** own GPU chips; it's the market + finance layer. Sellers list capacity;
  buyers reserve; idle time can be sublet on the secondary market **through Ornn**.

---

## 3. The pipeline end to end (one pass)

```
1. Someone defines a ComputePosition (YAML): "8 H100s hedged at $2.50/hr"
2. config.Load() decides mock vs live for price, telemetry, marketplace
3. Operator starts; controller-runtime watches ComputePositions
4. Every ~10s Reconcile() runs for each position:
     a. read the ComputePosition
     b. Telemetry.Utilization() → % busy
     c. OCPI.Price(sku)        → $/GPU-hr
     d. hedge.Compute()        → P&L, basis risk, idle GPUs, effectiveness
     e. evaluateIdle()         → sustained-low + hysteresis → availableForSublet
     f. handleMarketplace()    → post/withdraw supply on the flip
     g. applyOptionalAction()  → (opt-in only) pause/resume Deployment on price spike
     h. publishMetrics()       → Prometheus gauges
     i. writeStatus()          → status subresource + Kubernetes Events
5. Console polls the cluster → serves /api/state → React "trading desk" UI
6. Prometheus scrapes operator metrics → Grafana dashboard + alert rules
```

---

## 4. Repo map (what each path is)

```
api/v1alpha1/                  the CRD Go types (ComputePosition) + deepcopy
  computeposition_types.go     Spec/Status structs, kubebuilder markers
  groupversion_info.go         GroupVersion registration (computehedge.dev/v1alpha1)
  zz_generated.deepcopy.go     generated DeepCopy methods (required by K8s)

cmd/                           the runnable programs (one main per binary)
  operator/main.go             starts controller-runtime manager + reconciler
  console/main.go              read-only web API + embedded React SPA
  mockocpi/main.go             fake OCPI price service (demo)
  gpuexporter/main.go          fake per-position GPU utilization (demo, dcgm-style)
  console/web/                 the React + Vite + TS + Tailwind frontend

internal/                      the reusable logic (thin cmd/ wrappers call these)
  config/                      .env loader + mock/live mode resolution + factories
  hedge/                       the basis-risk / P&L math (pure, unit-tested)
  ocpi/                        price source interface: mock HTTP or live Ornn API
  telemetry/                   utilization source: mock HTTP or live Prometheus/DCGM
  market/                      marketplace publisher: log-only or live HTTP
  controller/                  the reconciler (the operator's brain)
  console/                     state aggregation + rolling history for the UI
  metrics/                     Prometheus series definitions
  pricesim/                    mean-reverting price engine for the mock

config/                        raw Kubernetes manifests
  crd/computepositions.yaml    CustomResourceDefinition (schema + printer columns)
  rbac.yaml                    ServiceAccount + ClusterRole + binding (least-privilege)
  manager.yaml                 operator Deployment + metrics Service
  secret.example.yaml          example ornn-credentials Secret

deploy/                        demo workloads + mock services
  samples/computepositions.yaml  two sample positions (critical + batch)
  workloads.yaml               placeholder Deployments (pause/resume targets)
  mockocpi.yaml, gpuexporter.yaml

observability/                 prometheus.yaml, grafana.yaml, grafana-dashboard.json
charts/compute-hedge-operator/ Helm chart (crds, templates, values, Chart.yaml)
.github/workflows/ci.yaml      CI: fmt, vet, test -race, build, golangci-lint, docker, helm lint
Dockerfile                     multi-stage: build web → build Go → distroless image
Makefile                       build/test/deploy/demo/live/mock targets
scripts/demo.ps1               one-shot demo driver
docs/                          README-adjacent docs (this file, PITCH, QA, DEMO_SCRIPT, PREREQS)
```

---

## 5. File-by-file deep dive (what to open and say)

### 5.1 `api/v1alpha1/computeposition_types.go` — the contract
This defines your **Custom Resource**. It's the "shape" of the YAML users write and the
status the operator writes back.

- `ComputePositionSpec` (desired state, set by the user):
  - `SKU` (e.g. H100), `GPUCount`, `HedgedPriceUSDPerHour` (string to avoid float drift).
  - `Priority` (`critical` never paused; `batch` may be paused if opted in).
  - `IdleThresholdPct` (default 15), `IdleWindowSeconds` (default 30).
  - `EnableActions` (opt-in pause/resume), `MaxSpotPriceUSDPerHour`, `WorkloadRef`.
- `ComputePositionStatus` (observed state, written by the operator): utilization, spot,
  P&L, effectiveness, basis risk, idle count, `AvailableForSublet`, `Recommendation`,
  `PriceStale`, `IdleSince`, `Paused`, `OriginalReplicas`, `LastUpdated`.
- kubebuilder markers `+kubebuilder:object:root=true` and `+kubebuilder:subresource:status`
  make it a root object with a **status subresource** (status updates don't bump generation).

Say: *"Prices are stored as strings in the spec to avoid floating-point drift in a
financial object; I parse them at reconcile time."*

### 5.2 `config/crd/computepositions.yaml` — the CRD
The cluster-side schema for the same type. Includes:
- `subresources.status: {}` — enables `status` as a subresource.
- `additionalPrinterColumns` — what `kubectl get computepositions` shows (SKU, GPUs,
  Util%, Spot, PnL/hr, BasisRisk/hr, Sublet, Phase).
- `openAPIV3Schema` — validation (required fields, enums, min/max).

Say: *"The CRD is how Kubernetes learns a brand-new object type; after applying it,
`kubectl get cpos` works and the API server validates every field."*

### 5.3 `internal/config/config.go` — the mock/live brain
The heart of the "pluggable" design.
- `Mode` = `auto | mock | live`. `resolve()` picks live when creds are present (auto),
  or forces it.
- `Load()` reads `.env` (via `dotenv.go`) then environment into a typed `Config`.
- Factories build the right implementation behind each interface:
  - `BuildOCPISource()` → `OrnnDataSource` (live) or `HTTPSource` (mock).
  - `BuildTelemetrySource()` → `PrometheusSource` (live) or `HTTPSource` (mock).
  - `BuildMarketPublisher()` → `HTTPPublisher` (live, gated) or `LogPublisher` (mock).
- `Sources()` returns provenance (`mock`/`live` + label) surfaced in the UI and logs.

Say: *"Golden rule: blank = mock, filled = live. The controller depends on interfaces,
so swapping implementations needs zero code change — just env/secret."*

### 5.4 `internal/config/dotenv.go` — dependency-free .env loader
Reads `KEY=VALUE` lines without overwriting existing env. No third-party dependency.

### 5.5 `internal/hedge/hedge.go` — the money math (know this cold)
Pure functions, unit-tested, no I/O. The identity:
```
utilized      = gpus * util% / 100
idle          = gpus - utilized
physicalRev   = spot * utilized
hedgePnL      = (hedged - spot) * gpus
lockedTarget  = hedged * gpus
net           = physicalRev + hedgePnL
basisRisk     = lockedTarget - net   == spot * (gpus - utilized)
effectiveness = clamp(net / lockedTarget * 100, 0, 100)
```
Key insight: **at 100% utilization basis risk is zero**, regardless of price. Money only
leaks when GPUs sit idle.

Say: *"I kept the economics as a pure package so it's trivially unit-testable and reusable
by the controller, the console, or a future CLI."*

### 5.6 `internal/ocpi/` — price source (interface + 2 impls)
- `ocpi.go`: `Price{SKU, USDPerHour, AsOf}`, `Source` interface, `Stale(maxAge)`.
- `httpsource.go`: mock — GETs `/prices/{sku}` from `mockocpi`.
- `ornndata.go`: live — GETs Ornn Data API with `Authorization: Bearer <token>`,
  path template `/v1/ocpi/{sku}/spot` (configurable), parses `{price, asOf}`.

Say: *"Same `Source` interface, two implementations. The reconciler never knows which."*

### 5.7 `internal/telemetry/` — utilization source (interface + 2 impls)
- `telemetry.go`: `Source` interface, mock `HTTPSource` (GET `/positions/{name}`).
- `prometheus.go`: live — runs a PromQL instant query, default
  `avg(DCGM_FI_DEV_GPU_UTIL{position="{position}"})`, template so any cluster's labels fit.

Say: *"On a real cluster utilization comes from NVIDIA DCGM via Prometheus; the mock
exporter stands in locally. The query is a template, so no code change to match labels."*

### 5.8 `internal/market/` — marketplace publisher (interface + 2 impls)
- `market.go`: `Offer` struct, `Publisher` interface, `LogPublisher` (records last offer
  in memory, no external calls — the safe default).
- `httppublisher.go`: live — POSTs offers, DELETEs on withdraw, **double-gated** by
  `WriteEnabled` so nothing is ever posted unless explicitly turned on.

Say: *"Write-back is off unless both a URL is set and `MARKET_WRITE_ENABLED=true` — a
deliberate safety switch for a system that could otherwise post real supply."*

### 5.9 `internal/controller/computeposition_controller.go` — the operator's brain
- Struct holds injected interfaces: `OCPI`, `Telemetry`, `Market`, plus `client.Client`
  and an event `Recorder`.
- `Reconcile()` is the loop (see section 3). Notable robustness:
  - Price fetch failure → reuse last known price, mark `PriceStale` (don't crash).
  - `apierrors.IsConflict` on status update → requeue (optimistic concurrency).
  - Always `RequeueAfter: reconcileInterval` → periodic polling of price/util.
- `evaluateIdle()` — sustained window + **hysteresis** (`idleResetMarginPct`) so the flag
  doesn't thrash around the threshold.
- `handleMarketplace()` — posts on `false→true`, withdraws on `true→false`, emits Events;
  failures never block a reconcile.
- `applyOptionalAction()` — pause/resume by scaling a Deployment to 0 and back; guarded by
  `EnableActions`, non-critical priority, and a `MaxSpotPriceUSDPerHour`.
- `SetupWithManager()` — `GenerationChangedPredicate` avoids re-reconciling on our own
  status writes.

Say: *"The controller is a thin orchestrator over injected interfaces and a pure math
package — easy to test, safe under failure, and never destructive by default."*

### 5.10 `cmd/operator/main.go` — wiring
Builds the scheme, `config.Load()`, logs resolved source modes, registers metrics, creates
the controller-runtime manager (metrics + health addresses), injects the three sources,
adds healthz/readyz, and starts.

### 5.11 `cmd/console/main.go` + `internal/console/state.go` — read-only UI backend
- Uses `//go:embed all:web/dist` to bake the React build into the Go binary (single
  artifact, no separate web server).
- Polls the cluster on an interval, builds `State` (portfolio aggregates, price ticker,
  per-position views, recent events, data-source provenance), serves `/api/state`.
- SPA fallback handler serves `index.html` for client routes.

Say: *"The console is strictly read-only; it never writes to the cluster. It's the headline
'trading desk' surface, embedded into one Go binary."*

### 5.12 `cmd/mockocpi` + `cmd/gpuexporter` + `internal/pricesim` — the demo doubles
- `mockocpi`: mean-reverting price engine (`pricesim`), `/prices`, `/prices/{sku}`,
  `/spike` for demos, plus Prometheus gauge `ocpi_price_usd_per_gpu_hour`.
- `gpuexporter`: per-position utilization with a controllable `/control/{name}?util=5`,
  exposes dcgm-style metric `DCGM_FI_DEV_GPU_UTIL`.

Say: *"These fakes exist only so the whole thing runs on a laptop with no GPUs and no Ornn
account, while emitting the exact same metric names real infra would."*

### 5.13 `internal/metrics/metrics.go` — Prometheus series
Registers gauges on the controller-runtime registry: utilization, hedge P&L, basis risk,
effectiveness, idle GPUs, available-for-sublet, spot price — all labeled `position, sku`.

### 5.14 `config/rbac.yaml` — least-privilege (see the dedicated RBAC section 8)

### 5.15 `config/manager.yaml` — the operator Deployment
- `serviceAccountName: compute-hedge-operator` (ties to RBAC).
- Hardened `securityContext`: `runAsNonRoot`, `allowPrivilegeEscalation: false`,
  `readOnlyRootFilesystem: true`, `capabilities: drop [ALL]`.
- `*_MODE=auto` envs + `envFrom` an **optional** `ornn-credentials` secret (absent = mock).
- Liveness/readiness probes, CPU/memory requests+limits, metrics Service.

### 5.16 `Dockerfile` — multi-stage, distroless
`node:22` builds the SPA → `golang:1.26` builds all four binaries → `distroless/static:nonroot`
final image running as UID 65532. Small, no shell, non-root.

### 5.17 `.github/workflows/ci.yaml` — CI gates
`gofmt` check, `go vet`, `go test -race`, `go build`, `golangci-lint`, `docker build`,
`helm lint`. `permissions: contents: read` (least-privilege token).

### 5.18 `charts/compute-hedge-operator/` — Helm packaging
CRDs, Deployment/RBAC templates, `values.yaml`, `Chart.yaml` — cluster install path for the
operator (not the full demo stack; mocks and console stay under `deploy/`).

---

## 6. The role you're targeting (and how to prove fit)

Ornn's public **Software Engineer** post asks for:
- design core **platform** features (frontend + backend, broad surface area);
- build for **reliability, observability, performance** as they scale;
- strong **security / operational rigor**;
- nice-to-have: **low-latency/high-throughput**, **compliant financial** systems, zero-to-one.

How this project demonstrates each:

| Role expectation | Where you show it |
|---|---|
| Platform (backend + frontend) | Go services (`cmd/*`) + React/TS console (`cmd/console/web`) |
| Reliability | requeue on conflict, stale-price fallback, hysteresis, health probes |
| Observability | Prometheus metrics, Grafana dashboard, alert rules, K8s Events |
| Performance/scale | interface-based sources, periodic reconcile, bounded history |
| Security/operational rigor | least-privilege RBAC, hardened pod, gated write-back, distroless |
| Compliant/financial | prices as strings, advisory-by-default, opt-in destructive actions |
| Zero-to-one | whole system built from scratch: CRD → controller → UI → CI → Helm |

Say: *"This isn't a toy controller — it's a full platform slice: an API (CRD), a control
loop, an observability stack, a product UI, CI, a Helm chart, and a pluggable integration
seam — exactly the surface area your SE role spans."*

---

## 7. Going live with real Ornn data (the "give me something real" moment)

Nothing in code changes. You provide credentials one of two ways.

### Local (laptop) — `.env`
```
ORNN_API_TOKEN=<token>            # flips price feed to live OCPI
ORNN_API_BASE_URL=https://api.ornnai.com
ORNN_API_PRICE_PATH=/v1/prices    # adapt to the exact subscribed path
PROMETHEUS_URL=http://prometheus:9090   # flips telemetry to live DCGM
TELEMETRY_QUERY=avg(DCGM_FI_DEV_GPU_UTIL{position="{position}"})
MARKET_API_URL=<marketplace endpoint>   # only if they give one
MARKET_WRITE_ENABLED=true                # explicit safety switch
```
Then `make run-operator` / `make run-console`.

### In-cluster — Kubernetes Secret
```
make live    # builds the ornn-credentials Secret from .env, restarts operator+console
make mock    # deletes the secret, back to simulator
```
`manager.yaml` already does `envFrom: secretRef: ornn-credentials (optional: true)`.

### What flips automatically
- Price: mock `mockocpi` → live `OrnnDataSource` (when `ORNN_API_TOKEN` present).
- Telemetry: mock exporter → live `PrometheusSource` (when `PROMETHEUS_URL` present).
- Market: `LogPublisher` → `HTTPPublisher` (when URL + `MARKET_WRITE_ENABLED=true`).
- Console header flips **SIMULATED → LIVE** using `config.Sources()`.

Say: *"Because the controller depends on interfaces and `config` chooses implementations
from env, the same image runs mock on my laptop and live in your cluster with only a Secret."*

### Adapting to their real API shape
If Ornn's JSON differs, the only change is the small response struct in `ornndata.go`
(`{price, asOf}`) or the PromQL template — both are isolated and easy.

---

## 8. RBAC explained from zero (you said you know this less)

### What RBAC is
**Role-Based Access Control** = Kubernetes' permission system. It answers: *"Which identity
can do which verbs on which resources?"* Nothing in a pod can touch the cluster API unless
RBAC allows it.

### The four objects (in your `config/rbac.yaml`)
1. **ServiceAccount** — the identity your operator pod runs as
   (`compute-hedge-operator` in `compute-hedge-system`). The pod references it via
   `serviceAccountName` in `manager.yaml`.
2. **Role vs ClusterRole** — a set of permissions. `Role` = one namespace; **`ClusterRole`**
   = cluster-wide (you use ClusterRole because positions/deployments can be in any namespace).
3. **Verbs** — the allowed actions: `get, list, watch, create, update, patch, delete`.
4. **RoleBinding vs ClusterRoleBinding** — glue that grants the Role/ClusterRole to a
   subject (your ServiceAccount). You use a **ClusterRoleBinding**.

### Your exact permissions (and why each is minimal)
```
computehedge.dev/computepositions         get,list,watch,create,update,patch,delete
computehedge.dev/computepositions/status  get,update,patch          # write status
apps/deployments                          get,list,watch,update,patch   # NO create/delete
""(core)/events                           create,patch              # emit K8s Events
```
- You can **update/patch** Deployments (to scale replicas for pause/resume) but **cannot
  create or delete** them — you can't destroy a customer workload.
- You only touch your own CRD + Events beyond that.

Say: *"Least privilege is deliberate: the widest thing I can do to a customer workload is
scale an opted-in Deployment's replicas. I can't create or delete workloads, and everything
destructive is opt-in per position."*

### Common RBAC questions
- *"Why ClusterRole not Role?"* Positions and target Deployments can live in any namespace;
  a namespaced Role wouldn't cover them.
- *"How does the pod get this identity?"* `serviceAccountName` in the Deployment → K8s mounts
  a token → the API server authenticates requests as that ServiceAccount → RBAC authorizes.
- *"How would you tighten further?"* Scope to specific namespaces with Roles, or restrict
  Deployments by name/label; drop `delete` on the CRD if the controller never deletes.
- *"What's the risk if RBAC is too broad?"* A compromised operator could modify unrelated
  workloads — hence drop everything not strictly needed.

---

## 9. Every tool, in depth (with likely questions & answers)

For each tool: what it is, why it's here, and the questions they may ask.

### 9.1 Kubernetes (core concepts)
- **Node**: a machine (has CPU/mem, and on GPU nodes `nvidia.com/gpu`).
- **Pod**: smallest deployable unit (one+ containers).
- **Deployment**: keeps N replicas of a pod running; you scale it for pause/resume.
- **Scheduler**: places pods on nodes that satisfy resource requests, nodeSelector,
  affinity, taints/tolerations. GPUs are requested via `resources.limits.nvidia.com/gpu`.
- **CRD + Controller = Operator**: extend the API with a new type, then run a control loop
  that drives real state toward the spec.
- **Reconciliation / control loop**: level-triggered — you compute desired vs observed and
  converge; you don't rely on catching every event.

Q: *"What's the difference between a controller and an operator?"*
A: An operator is a controller **plus** a CRD encoding domain knowledge — it automates a
human operator's tasks for a specific application.

Q: *"Level-triggered vs edge-triggered?"* Kubernetes controllers are level-triggered: on
each reconcile you read current state and act, so a missed event still self-heals on the
next requeue.

Q: *"How do GPUs appear in Kubernetes?"* NVIDIA's device plugin (installed by the NVIDIA
GPU Operator) registers `nvidia.com/gpu` on nodes; pods request it; the scheduler places them.

### 9.2 controller-runtime (the operator framework)
- Provides the **Manager** (shared caches, clients, metrics/health servers), the
  **reconciler** interface, **predicates**, and **event recorder**.
- You use `GenerationChangedPredicate` so status writes don't retrigger reconciles.
- `RequeueAfter` drives periodic polling of price/utilization.

Q: *"Why controller-runtime over client-go directly?"* It gives caching informers, leader
election, manager lifecycle, and metrics out of the box — far less boilerplate, fewer bugs.

Q: *"What is the informer cache?"* A local, watch-backed cache so reads don't hammer the API
server; the manager wires this for you.

Q: *"Optimistic concurrency?"* Status updates can conflict; on `IsConflict` I requeue and
retry with fresh state.

### 9.3 Go
- Interfaces (`ocpi.Source`, `telemetry.Source`, `market.Publisher`) enable the mock/live
  swap and easy unit tests.
- `context.Context` for timeouts/cancellation on every outbound HTTP call.
- Standard-library HTTP clients with explicit timeouts (no heavy deps).

Q: *"Why strings for money in the CRD?"* Avoid float rounding/serialization drift in a
financial object; parse to float only at compute time.

Q: *"How do you test the math?"* `hedge` is pure — table-driven unit tests, `go test -race`.

Q: *"Concurrency safety?"* `LogPublisher` and the console cache use a mutex; reconciles are
serialized per object by controller-runtime.

### 9.4 Prometheus
- Pull-based metrics: it **scrapes** `/metrics` endpoints on an interval.
- **Metric types**: counter (monotonic), **gauge** (up/down — what you use), histogram,
  summary. Your series are gauges labeled `position, sku`.
- **PromQL**: query language; your live telemetry runs an instant query
  `avg(DCGM_FI_DEV_GPU_UTIL{position="..."})`.
- **Alert rules**: e.g. `chp_available_for_sublet == 1` fires "idle capacity available."

Q: *"Counter vs gauge?"* Counter only goes up (rates via `rate()`); gauge can move both ways
(price, utilization, P&L) — hence gauges here.

Q: *"Cardinality risk?"* Labels multiply series; you keep labels to `position, sku` to stay
bounded.

Q: *"Push vs pull?"* Prometheus pulls; for short-lived jobs you'd use a pushgateway, but a
long-running operator exposes `/metrics` and is scraped.

### 9.5 Grafana
- Dashboards over Prometheus (time series for P&L, basis risk, utilization, marketplace
  supply). It's the engineer's **drill-down**; the console is the headline.

Q: *"Console vs Grafana — why both?"* Console is a branded, real-time product surface for
non-engineers; Grafana is deep historical analysis and alerting for operators.

### 9.6 DCGM / NVIDIA GPU Operator (live telemetry origin)
- **DCGM** = NVIDIA Data Center GPU Manager; **dcgm-exporter** exposes per-GPU metrics
  (e.g. `DCGM_FI_DEV_GPU_UTIL`) to Prometheus.
- The **NVIDIA GPU Operator** installs drivers, container toolkit, **device plugin**, and
  DCGM exporter across GPU nodes.

Q: *"Do you install the GPU Operator?"* No — it's assumed present on a real GPU cluster
(standard infra). My operator consumes DCGM metrics via Prometheus; the mock exporter
stands in locally.

### 9.7 Docker (multi-stage, distroless)
- Stage 1 builds the SPA (`node:22`), stage 2 builds Go binaries (`golang:1.26`), final is
  `distroless/static:nonroot` (no shell, non-root UID 65532, tiny attack surface).

Q: *"Why distroless?"* Smaller image, no package manager/shell to exploit, runs non-root —
security + size.

Q: *"Why multi-stage?"* Keep build tools out of the runtime image; ship only binaries.

### 9.8 Helm
- Templated Kubernetes packaging (CRDs + Deployment + RBAC + values). The cluster install
  path for the operator; CI runs `helm lint`. Demo companions (mocks, console, Grafana) are
  separate manifests under `deploy/` and `observability/`.

Q: *"Helm vs raw manifests?"* Helm parameterizes (image, replicas, modes) and versions the
release; raw manifests are fine for the demo but Helm scales to environments.

### 9.9 kind (Kubernetes in Docker)
- Runs a real Kubernetes cluster in containers on your laptop for the demo (no cloud, no GPUs).

Q: *"Why kind?"* Fast, disposable, real K8s API — perfect for a local end-to-end demo.

### 9.10 React + Vite + TypeScript + Tailwind (the console UI)
- React SPA polls `/api/state`; components render portfolio tiles, an OCPI ticker,
  per-position cards (utilization rings, effectiveness gauges, basis-risk sparklines),
  an event feed, a marketplace panel, and a SIMULATED/LIVE badge.
- Built by Vite; embedded into the Go binary via `go:embed`.

Q: *"Why embed the SPA in Go?"* One deployable artifact, one process, no separate static
host or CORS — simpler and more reliable.

### 9.11 GitHub Actions (CI)
- Gates: `gofmt`, `go vet`, `go test -race`, `go build`, `golangci-lint`, `docker build`,
  `helm lint`. Token scoped `contents: read`.

Q: *"Why `-race`?"* Detect data races in concurrent code (reconciles, caches) before merge.

### 9.12 Prometheus client_golang / kubebuilder markers / go:embed
- `client_golang` registers metrics on controller-runtime's registry.
- kubebuilder comment markers generate RBAC and mark the status subresource.
- `go:embed` bakes the built SPA into the binary.

---

## 10. What to open on screen, in order (your live script)

1. `deploy/samples/computepositions.yaml` — "Here's the user input: a hedged position."
2. `api/v1alpha1/computeposition_types.go` — "Here's the API contract (spec + status)."
3. `internal/hedge/hedge.go` — "Here's the economics; at 100% util basis risk is zero."
4. `internal/controller/computeposition_controller.go` — "Here's the reconcile loop."
5. `internal/config/config.go` — "Here's how mock flips to live with no code change."
6. `config/rbac.yaml` — "Here's least-privilege: I can scale, never create/delete."
7. `config/manager.yaml` — "Hardened pod, optional credentials secret."
8. Console (browser) — "The headline trading-desk view, SIMULATED/LIVE badge."
9. Grafana — "Engineer drill-down + the idle-capacity alert."
10. `.github/workflows/ci.yaml` + `Dockerfile` — "Shipped like production: CI + distroless."

Demo commands:
```
make demo            # build image, load into kind, apply everything
kubectl get cpos     # see status columns update
curl :8090/api/state # raw JSON behind the console
# force idle to trigger marketplace supply:
kubectl -n compute-hedge-system exec deploy/gpuexporter -- \
  wget -qO- 'http://localhost:8081/control/batch-render?util=5'
make live            # load .env creds → flips to LIVE
```

---

## 11. Q&A bank (rapid-fire, by theme)

### Product / business
- *"Why does this live in Kubernetes?"* Utilization and idle capacity are runtime facts that
  only exist where GPUs run; a financial desk can't see them from outside.
- *"What's basis risk in one line?"* The gap between the index-sized hedge and real utilized
  GPUs — `spot × (gpus − utilized)`.
- *"How does Ornn make money from this?"* Every idle block becomes secondary-market supply =
  a potential trade and fee.

### Architecture
- *"Why interfaces for sources?"* Swap mock/live with zero controller changes; unit-testable.
- *"What if the price feed is down?"* Reuse last known price, mark `PriceStale`, keep serving.
- *"How do you avoid flapping?"* Sustained idle window + hysteresis margin before clearing.
- *"Why is write-back double-gated?"* Posting real marketplace supply is consequential;
  requires URL **and** an explicit `MARKET_WRITE_ENABLED=true`.

### Reliability / operations
- *"How does it scale to many positions?"* Controller-runtime caches + per-object reconcile;
  metrics stay bounded (labels limited to position, sku).
- *"Leader election / HA?"* controller-runtime supports it; single replica for the demo,
  enable leader election for multi-replica.
- *"Health checks?"* liveness/readiness probes on `/healthz` `/readyz`.

### Security
- *"Least privilege proof?"* RBAC: no create/delete on Deployments; hardened pod
  (non-root, read-only FS, drop ALL caps); distroless image.
- *"Secrets handling?"* Credentials via optional K8s Secret / `.env`; never in code or image.

### Testing / CI
- *"What's tested?"* Pure `hedge` math (table-driven), `go test -race`, plus `vet`,
  `golangci-lint`, `docker build`, `helm lint` in CI.

---

## 12. Honesty guardrails (do not overclaim)
- Ornn's **public docs** do **not** publish a marketplace-supply API; your `MARKET_API_URL`
  path is your integration design, ready if they supply an endpoint.
- You **assume** the customer cluster runs NVIDIA GPU Operator + Prometheus; you don't install it.
- The demo uses fakes (`mockocpi`, `gpuexporter`, `pause` pods) because `kind` has no GPUs;
  the architecture is identical to live.

Say plainly: *"It runs on mock today; it flips to live the moment you hand me a token and a
Prometheus URL — and if you give me a marketplace endpoint, supply posting turns on too."*

---

## 13. One-paragraph close (say at the end)
> "So this is a full platform slice for Ornn's runtime gap: a Kubernetes operator that turns
> raw GPU telemetry into position-level economics against OCPI, surfaces idle capacity as
> marketplace supply, and ships with observability, least-privilege security, CI, and a Helm
> chart. It's advisory and safe by default, and it goes live with just credentials. It's
> exactly the platform + observability + reliability work your Software Engineer role is about —
> and I built it end to end."
