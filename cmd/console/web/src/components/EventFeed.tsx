import type { EventView } from "../api";
import { relTime } from "../lib/format";

interface ReasonStyle {
  color: string;
  icon: string;
}

function reasonStyle(reason: string, type: string): ReasonStyle {
  switch (reason) {
    case "PausedOnPriceSpike":
      return { color: "#fb7185", icon: "⏸" };
    case "ResumedOnPriceRecovery":
      return { color: "#34d399", icon: "▶" };
    case "IdleCapacityAvailable":
      return { color: "#fbbf24", icon: "◧" };
    case "HighBasisRisk":
      return { color: "#f97316", icon: "⚠" };
    default:
      return {
        color: type === "Warning" ? "#fbbf24" : "#38bdf8",
        icon: "•",
      };
  }
}

export default function EventFeed({ events }: { events: EventView[] }) {
  return (
    <div className="flex h-full flex-col rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur">
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <h3 className="text-sm font-semibold text-slate-200">Live event feed</h3>
        <span className="flex items-center gap-1.5 text-[11px] uppercase tracking-widest text-cyan-300">
          <span className="h-2 w-2 animate-pulseDot rounded-full bg-cyan-400" />
          streaming
        </span>
      </div>
      <div className="flex-1 space-y-1 overflow-y-auto p-2">
        {events.length === 0 && (
          <div className="px-3 py-8 text-center text-xs text-slate-500">
            No controller events yet. Inject a spike or force idle to see
            actions here.
          </div>
        )}
        {events.map((e, i) => {
          const s = reasonStyle(e.reason, e.type);
          return (
            <div
              key={`${e.time}-${i}`}
              className="flex items-start gap-3 rounded-lg px-3 py-2 hover:bg-white/[0.03]"
            >
              <span
                className="mt-0.5 flex h-6 w-6 flex-none items-center justify-center rounded-full text-xs"
                style={{
                  color: s.color,
                  backgroundColor: `${s.color}1f`,
                  boxShadow: `0 0 10px ${s.color}55`,
                }}
              >
                {s.icon}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <span
                    className="truncate text-xs font-semibold"
                    style={{ color: s.color }}
                  >
                    {e.reason}
                  </span>
                  <span className="flex-none text-[10px] text-slate-500">
                    {relTime(e.time)}
                  </span>
                </div>
                <div className="truncate text-[11px] text-slate-400">
                  <span className="font-mono text-slate-300">{e.position}</span>
                  {" · "}
                  {e.message}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
