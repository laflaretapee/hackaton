import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    coverage: {
      reporter: ["text", "json-summary"],
      thresholds: { lines: 80, statements: 80, branches: 80, functions: 80 },
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.test.*", "src/main.tsx", "src/test-setup.ts"],
    },
  },
});
