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
    // Run test files sequentially in a single worker. Vitest 4 removed
    // poolOptions.forks.singleFork; fileParallelism:false is the top-level
    // replacement (it pins maxWorkers to 1) and keeps the MUI suites stable.
    fileParallelism: false,
    server: {
      deps: {
        // MUI ships ESM that Node's resolver rejects (directory imports);
        // inlining lets Vite resolve it instead.
        inline: [/@mui\//, /react-transition-group/],
      },
    },
  },
});
