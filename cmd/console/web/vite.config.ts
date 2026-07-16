import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The console binary serves the built SPA from the same origin, so base is "/".
// During local `npm run dev`, /api and /healthz are proxied to the Go backend.
export default defineConfig({
  plugins: [react()],
  base: "/",
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8090",
      "/healthz": "http://localhost:8090",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
