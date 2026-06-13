import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    css: false,
    testTimeout: 15000,
    pool: "forks",
    // @ts-expect-error Vitest 4 runtime accepts poolOptions, but the shipped
    // d.ts only exposes `pool: string`. Single-fork mode keeps MUI tests stable.
    poolOptions: {
      forks: {
        singleFork: true,
      },
    },
    server: {
      deps: {
        // MUI ships ESM that Node's resolver rejects (directory imports);
        // inlining lets Vite resolve it instead.
        inline: [/@mui\//, /react-transition-group/],
      },
    },
  },
});
