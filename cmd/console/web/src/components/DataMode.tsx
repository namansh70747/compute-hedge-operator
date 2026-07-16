import type { DataSources, SourceInfo } from "../api";

function anyLive(s: DataSources): boolean {
  return (
    s.price.mode === "live" ||
    s.telemetry.mode === "live" ||
    s.market.mode === "live"
  );
}

function liveLabel(s: DataSources): string {
  const live = [s.price, s.telemetry, s.market].filter((x) => x.mode === "live");
  if (live.length === 0) return "";
  const labels = Array.from(new Set(live.map((x) => x.label)));
  return labels.join(" · ");
}

export function ModePill({ sources }: { sources: DataSources }) {
  const live = anyLive(sources);
  const color = live ? "#34d399" : "#fbbf24";
  const label = live ? `LIVE - ${liveLabel(sources)}` : "SIMULATED DATA";
  return (
    <div
      className="flex items-center gap-2 rounded-full border border-white/10 bg-black/30 px-3 py-1.5"
      style={{ boxShadow: `0 0 16px -4px ${color}` }}
      title={live ? "Reading real data sources" : "All data is simulated"}
    >
      <span
        className="h-2.5 w-2.5 animate-pulseDot rounded-full"
        style={{ backgroundColor: color, boxShadow: `0 0 10px ${color}` }}
      />
      <span
        className="text-[11px] font-semibold tracking-widest"
        style={{ color }}
      >
        {label}
      </span>
    </div>
  );
}

function Chip({ name, info }: { name: string; info: SourceInfo }) {
  const live = info.mode === "live";
  const color = live ? "#34d399" : "#64748b";
  return (
    <div className="flex items-center gap-2 rounded-lg border border-white/5 bg-white/[0.02] px-2.5 py-1">
      <span className="text-[10px] uppercase tracking-wider text-slate-500">
        {name}
      </span>
      <span
        className="h-1.5 w-1.5 rounded-full"
        style={{ backgroundColor: color, boxShadow: `0 0 6px ${color}` }}
      />
      <span
        className="font-mono text-[11px]"
        style={{ color: live ? "#a7f3d0" : "#94a3b8" }}
      >
        {live ? info.label : "simulated"}
      </span>
    </div>
  );
}

export function ProvenanceStrip({ sources }: { sources: DataSources }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-[10px] uppercase tracking-widest text-slate-600">
        Data sources
      </span>
      <Chip name="Price" info={sources.price} />
      <Chip name="Telemetry" info={sources.telemetry} />
      <Chip name="Market" info={sources.market} />
    </div>
  );
}
