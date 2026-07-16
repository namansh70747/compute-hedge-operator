import type { Portfolio, SourceInfo } from "../api";
import { usd } from "../lib/format";

interface MarketPanelProps {
  portfolio: Portfolio;
  market: SourceInfo;
}

export default function MarketPanel({ portfolio, market }: MarketPanelProps) {
  const live = market.mode === "live";
  const accent = portfolio.idleGPUsAvailable > 0 ? "#a78bfa" : "#64748b";
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4 backdrop-blur">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-200">Marketplace supply</h3>
        <span
          className="rounded-full border border-white/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider"
          style={{ color: live ? "#34d399" : "#94a3b8" }}
        >
          {live ? "posting live" : "advisory"}
        </span>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2">
        <div className="rounded-lg border border-white/5 bg-white/[0.02] px-3 py-2">
          <div className="text-[10px] uppercase tracking-wider text-slate-500">
            Idle GPUs offered
          </div>
          <div
            className="tabular mt-0.5 font-mono text-xl font-semibold"
            style={{ color: accent }}
          >
            {portfolio.idleGPUsAvailable}
          </div>
        </div>
        <div className="rounded-lg border border-white/5 bg-white/[0.02] px-3 py-2">
          <div className="text-[10px] uppercase tracking-wider text-slate-500">
            Reclaimable $/hr
          </div>
          <div className="tabular mt-0.5 font-mono text-xl font-semibold text-cyan-300">
            {usd(portfolio.marketplaceSupplyUSDPerHour)}
          </div>
        </div>
      </div>
      <p className="mt-3 text-[11px] leading-relaxed text-slate-500">
        {live
          ? `Idle capacity is posted to ${market.label} as marketplace supply.`
          : "Idle capacity is flagged as marketplace supply. Add a marketplace endpoint to post it live."}
      </p>
    </div>
  );
}
