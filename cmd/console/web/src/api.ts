export interface PortfolioHistory {
  hedgePnL: number[];
  basisRisk: number[];
  notional: number[];
  supply: number[];
}

export interface Portfolio {
  hedgedNotionalUSDPerHour: number;
  hedgePnLUSDPerHour: number;
  basisRiskUSDPerHour: number;
  idleGPUsAvailable: number;
  marketplaceSupplyUSDPerHour: number;
  positions: number;
  history: PortfolioHistory;
}

export interface PriceView {
  sku: string;
  usdPerHour: number;
  changePct: number;
}

export interface PositionHistory {
  basisRisk: number[];
  utilization: number[];
  pnl: number[];
}

export interface PositionView {
  name: string;
  namespace: string;
  sku: string;
  gpuCount: number;
  priority: string;
  phase: string;
  utilizationPct: number;
  spotUSDPerHour: number;
  hedgedUSDPerHour: number;
  hedgePnLUSDPerHour: number;
  basisRiskUSDPerHour: number;
  hedgeEffectivenessPct: number;
  idleGPUCount: number;
  availableForSublet: boolean;
  priceStale: boolean;
  recommendation: string;
  history: PositionHistory;
}

export interface EventView {
  time: string;
  position: string;
  reason: string;
  type: string;
  message: string;
}

export interface SourceInfo {
  mode: "mock" | "live";
  label: string;
}

export interface DataSources {
  price: SourceInfo;
  telemetry: SourceInfo;
  market: SourceInfo;
}

export interface State {
  asOf: string;
  cluster: string;
  dataSources: DataSources;
  portfolio: Portfolio;
  prices: PriceView[];
  positions: PositionView[];
  events: EventView[];
}

export async function fetchState(signal?: AbortSignal): Promise<State> {
  const res = await fetch("/api/state", { signal, cache: "no-store" });
  if (!res.ok) {
    throw new Error(`state request failed: ${res.status}`);
  }
  return (await res.json()) as State;
}
