import type { Portfolio } from "../api";
import { usd, signedUsd, usdCompact } from "../lib/format";
import Sparkline from "./Sparkline";

interface TileProps {
  label: string;
  value: string;
  accent: string;
  data: number[];
  hint?: string;
  valueClass?: string;
}

function Tile({ label, value, accent, data, hint, valueClass }: TileProps) {
  return (
    <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-white/[0.03] p-4 backdrop-blur">
      <div
        className="absolute inset-x-0 top-0 h-[2px]"
        style={{ background: accent, boxShadow: `0 0 14px ${accent}` }}
      />
      <div className="flex items-start justify-between">
        <span className="text-[11px] font-medium uppercase tracking-wider text-slate-400">
          {label}
        </span>
        {hint && (
          <span className="text-[11px] font-medium text-slate-400">{hint}</span>
        )}
      </div>
      <div
        className={`tabular mt-2 font-mono text-2xl font-semibold leading-none ${valueClass ?? "text-slate-50"}`}
      >
        {value}
      </div>
      <div className="mt-3 h-9">
        <Sparkline data={data} color={accent} width={220} height={36} />
      </div>
    </div>
  );
}

export default function PortfolioBar({ portfolio }: { portfolio: Portfolio }) {
  const pnl = portfolio.hedgePnLUSDPerHour;
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
      <Tile
        label="Hedged notional"
        value={`${usdCompact(portfolio.hedgedNotionalUSDPerHour)}/hr`}
        accent="#38bdf8"
        data={portfolio.history.notional}
        hint={`${portfolio.positions} positions`}
      />
      <Tile
        label="Hedge P&L"
        value={`${signedUsd(pnl)}/hr`}
        accent={pnl >= 0 ? "#34d399" : "#fb7185"}
        valueClass={pnl >= 0 ? "text-emerald-300" : "text-rose-300"}
        data={portfolio.history.hedgePnL}
      />
      <Tile
        label="Basis risk"
        value={`${usd(portfolio.basisRiskUSDPerHour)}/hr`}
        accent="#fbbf24"
        data={portfolio.history.basisRisk}
      />
      <Tile
        label="Idle GPUs for sublet"
        value={`${portfolio.idleGPUsAvailable}`}
        accent="#a78bfa"
        data={portfolio.history.supply.map((v) => v)}
        hint="available"
      />
      <Tile
        label="Marketplace supply"
        value={`${usdCompact(portfolio.marketplaceSupplyUSDPerHour)}/hr`}
        accent="#22d3ee"
        data={portfolio.history.supply}
        hint="reclaimable"
      />
    </div>
  );
}
