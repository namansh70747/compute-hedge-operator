import type { PriceView } from "../api";
import { usd, signedPct } from "../lib/format";

function Chip({ p }: { p: PriceView }) {
  const up = p.changePct > 0.001;
  const down = p.changePct < -0.001;
  const color = up ? "text-emerald-300" : down ? "text-rose-300" : "text-slate-400";
  const arrow = up ? "▲" : down ? "▼" : "•";
  return (
    <div className="flex items-center gap-3 whitespace-nowrap px-5 py-2">
      <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
        {p.sku}
      </span>
      <span className="tabular font-mono text-sm font-semibold text-slate-100">
        {usd(p.usdPerHour)}
        <span className="text-slate-500">/GPU-hr</span>
      </span>
      <span className={`tabular font-mono text-xs font-semibold ${color}`}>
        {arrow} {signedPct(p.changePct)}
      </span>
      <span className="text-slate-700">|</span>
    </div>
  );
}

export default function PriceTicker({ prices }: { prices: PriceView[] }) {
  if (!prices.length) {
    return null;
  }
  const loop = [...prices, ...prices];
  return (
    <div className="relative overflow-hidden rounded-xl border border-white/10 bg-white/[0.02]">
      <div className="pointer-events-none absolute left-0 top-0 z-10 flex h-full items-center bg-ink-950/80 px-3">
        <span className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-cyan-300">
          <span className="h-2 w-2 animate-pulseDot rounded-full bg-cyan-400" />
          OCPI
        </span>
      </div>
      <div className="flex w-max animate-marquee items-center pl-24">
        {loop.map((p, i) => (
          <Chip key={`${p.sku}-${i}`} p={p} />
        ))}
      </div>
    </div>
  );
}
