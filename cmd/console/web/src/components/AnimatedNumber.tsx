import { useEffect, useRef, useState } from "react";

interface AnimatedNumberProps {
  value: number;
  format: (v: number) => string;
  durationMs?: number;
  className?: string;
}

// Smoothly tweens between numeric values so metric changes read as "live" motion.
export default function AnimatedNumber({
  value,
  format,
  durationMs = 500,
  className,
}: AnimatedNumberProps) {
  const [display, setDisplay] = useState(value);
  const fromRef = useRef(value);
  const startRef = useRef(0);
  const rafRef = useRef(0);

  useEffect(() => {
    fromRef.current = display;
    startRef.current = performance.now();
    const from = fromRef.current;
    const delta = value - from;

    const tick = (now: number) => {
      const t = Math.min(1, (now - startRef.current) / durationMs);
      const eased = 1 - Math.pow(1 - t, 3);
      setDisplay(from + delta * eased);
      if (t < 1) {
        rafRef.current = requestAnimationFrame(tick);
      }
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafRef.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, durationMs]);

  return <span className={className}>{format(display)}</span>;
}
