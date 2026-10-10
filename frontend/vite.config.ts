import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

const backend = process.env.BACKEND_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 3000,
    proxy: {
      "/api": backend,
      "/auth": backend,
      "/photos": backend,
      // Label codes opened from a QR code, e.g. /A107.
      "^/[LlIiSsAa][1-9][0-9]*$": backend,
    },
  },
  test: {
    environment: "jsdom",
  },
});
