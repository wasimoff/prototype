import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [ vue() ],
  appType: "mpa", // properly return 404
  resolve: {
    alias: {
      "@app": fileURLToPath(new URL("./src", import.meta.url)),
      "@wasimoff": fileURLToPath(new URL("./lib", import.meta.url)),
    },
  },
  build: {
    target: "esnext",
    sourcemap: "hidden",
  },
  worker: { format: "es" },
  server: {
    headers: {
      // support SharedArrayBuffers
      "Cross-Origin-Embedder-Policy": "require-corp",
      "Cross-Origin-Opener-Policy": "same-origin",
    },
    proxy: {
      // forward websockets
      "^/api/[a-z]+/ws": {
        target: "http://localhost:4080",
        ws: true,
      },
      // forward any API requests to the broker
      "^/api/": "http://localhost:4080",
    }
  },
  optimizeDeps: {
    exclude: [ "pyodide" ],
  },
})
