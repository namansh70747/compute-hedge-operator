interface GaugeProps {
  value: number;
  size?: number;
  stroke?: number;
  label?: string;
  caption?: string;
}

function colorFor(value: number): string {
  if (value >= 80) return "#34d399";
  if (value >= 50) return "#fbbf24";
  return "#fb7185";
}

// Semi-circular gauge (180deg) for hedge effectiveness.
export default function Gauge({
  value,
  size = 132,
  stroke = 10,
  label,
  caption = "Hedge effectiveness",
}: GaugeProps) {
  const clamped = Math.max(0, Math.min(100, value));
  const color = colorFor(clamped);
  const r = (size - stroke) / 2;
  const cx = size / 2;
  const cy = size / 2;
  const height = size / 2 + stroke;

  const polar = (frac: number) => {
    const angle = Math.PI * (1 - frac);
    return [cx + r * Math.cos(angle), cy - r * Math.sin(angle)] as const;
  };
  const arc = (frac: number) => {
    const [sx, sy] = polar(0);
    const [ex, ey] = polar(frac);
    const large = frac > 0.5 ? 1 : 0;
    return `M${sx.toFixed(1)},${sy.toFixed(1)} A${r},${r} 0 ${large} 1 ${ex.toFixed(1)},${ey.toFixed(1)}`;
  };

  return (
    <div className="flex flex-col items-center">
      <svg width={size} height={height}>
        <path
          d={arc(1)}
          fill="none"
          stroke="rgba(148,163,184,0.16)"
          strokeWidth={stroke}
          strokeLinecap="round"
        />
        <path
          d={arc(clamped / 100)}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeLinecap="round"
          style={{
            transition: "all 0.6s ease",
            filter: `drop-shadow(0 0 6px ${color}66)`,
          }}
        />
        <text
          x={cx}
          y={cy - 4}
          textAnchor="middle"
          className="tabular font-mono"
          fontSize="22"
          fontWeight="700"
          fill="#e6edf7"
        >
          {label ?? `${Math.round(clamped)}%`}
        </text>
      </svg>
      <span className="-mt-1 text-[10px] uppercase tracking-wider text-slate-400">
        {caption}
      </span>
    </div>
  );
}
