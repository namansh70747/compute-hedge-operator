/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        mono: [
          "JetBrains Mono",
          "SFMono-Regular",
          "Menlo",
          "Consolas",
          "monospace",
        ],
      },
      colors: {
        ink: {
          950: "#060a14",
          900: "#0a1120",
          850: "#0d1526",
          800: "#111c33",
          700: "#1a2740",
        },
      },
      boxShadow: {
        glow: "0 0 0 1px rgba(255,255,255,0.04), 0 0 24px -6px var(--tw-shadow-color)",
      },
      keyframes: {
        pulseDot: {
          "0%, 100%": { opacity: "1", transform: "scale(1)" },
          "50%": { opacity: "0.4", transform: "scale(0.7)" },
        },
        flash: {
          "0%": { backgroundColor: "rgba(56,189,248,0.18)" },
          "100%": { backgroundColor: "rgba(255,255,255,0)" },
        },
        marquee: {
          "0%": { transform: "translateX(0)" },
          "100%": { transform: "translateX(-50%)" },
        },
      },
      animation: {
        pulseDot: "pulseDot 1.4s ease-in-out infinite",
        flash: "flash 1.1s ease-out",
        marquee: "marquee 32s linear infinite",
      },
    },
  },
  plugins: [],
};
