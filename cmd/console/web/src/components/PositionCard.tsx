import { useEffect, useRef, useState } from "react";
import type { PositionView } from "../api";
import { usd, signedUsd, pct } from "../lib/format";
import Ring from "./Ring";
import Gauge from "./Gauge";
import Sparkline from "./Sparkline";

interface PhaseStyle {
  label: string;
  dot: string;
  text: string;
  ring: string;
  glow: string;
}

function phaseStyle(phase: string): PhaseStyle {
  switch (phase) {
    case "IdleAvailable":
      return {
        label: "Idle · available",
        dot: "#fbbf24",
        text: "text-amber-300",
        ring: "border-amber-400/40",
        glow: "rgba(251,191,36,0.35)",
      };
    case "Paused":
      return {
        label: "Paused",
        dot: "#fb7185",
        text: "text-rose-300",
        ring: "border-rose-400/40",
        glow: "rgba(251,113,133,0.35)",
      };
    case "Active":
      return {
        label: "Active",
        dot: "#34d399",
        text: "text-emerald-300",
        ring: "border-emerald-400/30",
        glow: "rgba(52,211,153,0.28)",
      };
    default:
      return {
        label: phase || "Pending",
        dot: "#94a3b8",
        text: "text-slate-300",
        ring: "border-white/10",
        glow: "rgba(148,163,184,0.2)",
      };
  }
}

function utilColor(v: number): string {
  if (v >= 60) return "#34d399";
  if (v >= 20) return "#38bdf8";
  return "#fbbf24";
}

function Metric({
  label,
  value,
  valueClass,
}: {
  label: string;
  value: string;
  valueClass?: string;
}) {
  return (
    <div className="rounded-lg border border-white/5 bg-white/[0.02] px-3 py-2">
      <div className="text-[10px] uppercase tracking-wider text-slate-500">
        {label}
      </div>
      <div
        className={`tabular mt-0.5 font-mono text-sm font-semibold ${valueClass ?? "text-slate-100"}`}
      >
        {value}
      </div>
    </div>
  );
}

interface PositionCardProps {
  pos: PositionView;
  telemetryMode?: "mock" | "live";
  priceMode?: "mock" | "live";
}

function SourceChip({ label, mode }: { label: string; mode?: "mock" | "live" }) {
  const live = mode === "live";
  const color = live ? "#34d399" : "#64748b";
  return (
    <span
      className="flex items-center gap-1 rounded border border-white/5 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider"
      style={{ color: live ? "#a7f3d0" : "#94a3b8" }}
      title={`${label}: ${live ? "live" : "simulated"}`}
    >
      <span
        className="h-1.5 w-1.5 rounded-full"
        style={{ backgroundColor: color }}
      />
      {label}
    </span>
  );
}

export default function PositionCard({
  pos,
  telemetryMode,
  priceMode,
}: PositionCardProps) {
  const style = phaseStyle(pos.phase);
  const pnlPositive = pos.hedgePnLUSDPerHour >= 0;
  const spotDelta = pos.spotUSDPerHour - pos.hedgedUSDPerHour;

  const [flash, setFlash] = useState(false);
  const prevPhase = useRef(pos.phase);
  useEffect(() => {
    if (prevPhase.current !== pos.phase) {
      prevPhase.current = pos.phase;
      setFlash(true);
      const t = setTimeout(() => setFlash(false), 1100);
      return () => clearTimeout(t);
    }
  }, [pos.phase]);

  return (
    <div
      className={`relative overflow-hidden rounded-2xl border bg-gradient-to-b from-white/[0.05] to-white/[0.01] p-5 backdrop-blur transition-shadow ${style.ring} ${flash ? "animate-flash" : ""}`}
      style={{ boxShadow: `0 18px 40px -24px ${style.glow}` }}
    >
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base font-semibold text-slate-100">
              {pos.name}
            </h3>
            {pos.priority === "critical" && (
              <span className="rounded border border-cyan-400/30 bg-cyan-400/10 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-cyan-300">
                critical
              </span>
            )}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-400">
            <span className="rounded bg-white/5 px-1.5 py-0.5 font-mono text-slate-300">
              {pos.sku}
            </span>
            <span>{pos.gpuCount} GPUs</span>
            {pos.priceStale && (
              <span className="text-amber-400">· price stale</span>
            )}
            <SourceChip label="util" mode={telemetryMode} />
            <SourceChip label="px" mode={priceMode} />
          </div>
        </div>
        <span
          className={`flex items-center gap-1.5 rounded-full border border-white/10 bg-black/30 px-2.5 py-1 text-[11px] font-semibold ${style.text}`}
        >
          <span
            className="h-2 w-2 animate-pulseDot rounded-full"
            style={{ backgroundColor: style.dot, boxShadow: `0 0 8px ${style.dot}` }}
          />
          {style.label}
        </span>
      </div>

      <div className="mt-4 flex items-center justify-between gap-3">
        <Ring
          value={pos.utilizationPct}
          color={utilColor(pos.utilizationPct)}
          sublabel="Utilization"
        />
        <Gauge value={pos.hedgeEffectivenessPct} />
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2">
        <Metric
          label="Spot vs hedged"
          value={`${usd(pos.spotUSDPerHour)} / ${usd(pos.hedgedUSDPerHour)}`}
        />
        <Metric
          label="Spot delta"
          value={`${signedUsd(spotDelta)}`}
          valueClass={spotDelta > 0 ? "text-rose-300" : "text-emerald-300"}
        />
        <Metric
          label="Hedge P&L"
          value={`${signedUsd(pos.hedgePnLUSDPerHour)}/hr`}
          valueClass={pnlPositive ? "text-emerald-300" : "text-rose-300"}
        />
        <Metric
          label="Basis risk"
          value={`${usd(pos.basisRiskUSDPerHour)}/hr`}
          valueClass="text-amber-300"
        />
      </div>

      <div className="mt-3">
        <div className="flex items-center justify-between text-[10px] uppercase tracking-wider text-slate-500">
          <span>Basis risk trend</span>
          {pos.idleGPUCount > 0 && (
            <span className="text-amber-300">
              {pos.idleGPUCount} idle{pos.availableForSublet ? " · sublet" : ""}
            </span>
          )}
        </div>
        <div className="mt-1 h-10">
          <Sparkline
            data={pos.history.basisRisk}
            color="#fbbf24"
            width={320}
            height={40}
          />
        </div>
      </div>

      {pos.recommendation && (
        <div className="mt-3 rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-xs text-slate-300">
          <span className="mr-1.5 font-semibold text-cyan-300">▸</span>
          {pos.recommendation}
        </div>
      )}

      <div className="mt-2 text-right text-[10px] text-slate-600">
        util {pct(pos.utilizationPct)} · eff {pct(pos.hedgeEffectivenessPct)}
      </div>
    </div>
  );
}
