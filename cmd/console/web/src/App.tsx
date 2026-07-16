import { useEffect, useRef, useState } from "react";
import { fetchState, type State } from "./api";
import PortfolioBar from "./components/PortfolioBar";
import PriceTicker from "./components/PriceTicker";
import PositionCard from "./components/PositionCard";
import EventFeed from "./components/EventFeed";

type Conn = "connecting" | "live" | "stale";

function useClock(): string {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, []);
  return now.toUTCString().slice(17, 25);
}

function Brand() {
  return (
    <div className="flex items-center gap-3">
      <div className="relative flex h-10 w-10 items-center justify-center rounded-xl border border-cyan-400/30 bg-cyan-400/10">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <path
            d="M3 17l5-6 4 4 4-8 5 10"
            stroke="#38bdf8"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </div>
      <div>
        <h1 className="text-lg font-semibold leading-tight text-slate-50">
          Compute Hedge{" "}
          <span className="text-cyan-300">Console</span>
        </h1>
        <p className="text-[11px] uppercase tracking-widest text-slate-500">
          GPU hedge risk · live from the cluster
        </p>
      </div>
    </div>
  );
}

export default function App() {
  const [state, setState] = useState<State | null>(null);
  const [conn, setConn] = useState<Conn>("connecting");
  const clock = useClock();
  const misses = useRef(0);

  useEffect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const s = await fetchState();
        if (!alive) return;
        setState(s);
        setConn("live");
        misses.current = 0;
      } catch {
        misses.current += 1;
        if (misses.current >= 2) setConn("stale");
      }
    };
    tick();
    const id = setInterval(tick, 2000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  const connMeta =
    conn === "live"
      ? { color: "#34d399", label: "LIVE" }
      : conn === "stale"
        ? { color: "#fb7185", label: "RECONNECTING" }
        : { color: "#fbbf24", label: "CONNECTING" };

  return (
    <div className="min-h-full">
      <div className="grid-overlay">
        <div className="mx-auto max-w-[1600px] px-5 pb-10 pt-5">
          <header className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-white/10 bg-white/[0.03] px-5 py-4 backdrop-blur">
            <Brand />
            <div className="flex items-center gap-6">
              <div className="text-right">
                <div className="text-[10px] uppercase tracking-wider text-slate-500">
                  Cluster
                </div>
                <div className="font-mono text-sm text-slate-200">
                  {state?.cluster ?? "—"}
                </div>
              </div>
              <div className="text-right">
                <div className="text-[10px] uppercase tracking-wider text-slate-500">
                  UTC
                </div>
                <div className="tabular font-mono text-sm text-slate-200">
                  {clock}
                </div>
              </div>
              <div
                className="flex items-center gap-2 rounded-full border border-white/10 bg-black/30 px-3 py-1.5"
                style={{ boxShadow: `0 0 16px -4px ${connMeta.color}` }}
              >
                <span
                  className="h-2.5 w-2.5 animate-pulseDot rounded-full"
                  style={{
                    backgroundColor: connMeta.color,
                    boxShadow: `0 0 10px ${connMeta.color}`,
                  }}
                />
                <span
                  className="text-[11px] font-semibold tracking-widest"
                  style={{ color: connMeta.color }}
                >
                  {connMeta.label}
                </span>
              </div>
            </div>
          </header>

          {state ? (
            <>
              <section className="mt-4">
                <PortfolioBar portfolio={state.portfolio} />
              </section>

              <section className="mt-4">
                <PriceTicker prices={state.prices} />
              </section>

              <div className="mt-4 grid gap-4 xl:grid-cols-[1fr_360px]">
                <section>
                  <div className="mb-2 flex items-center justify-between">
                    <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-400">
                      Positions
                    </h2>
                    <span className="text-xs text-slate-500">
                      {state.positions.length} tracked
                    </span>
                  </div>
                  <div className="grid gap-4 md:grid-cols-2">
                    {state.positions.map((p) => (
                      <PositionCard key={p.name} pos={p} />
                    ))}
                  </div>
                </section>

                <aside className="h-[560px] xl:sticky xl:top-4">
                  <EventFeed events={state.events} />
                </aside>
              </div>
            </>
          ) : (
            <div className="mt-24 text-center text-slate-500">
              <div className="mx-auto mb-4 h-10 w-10 animate-pulseDot rounded-full bg-cyan-400/40" />
              Connecting to the console API…
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
