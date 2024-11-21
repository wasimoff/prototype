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
      // forward API requests to the broker
      "^/api/broker": "http://localhost:4080",
      // forward storage fetches to the broker
      "^/storage/.*": "http://localhost:4080",
      // forward the websockets
      "^/websocket/.*": {
        target: "http://localhost:4080",
        ws: true,
      }
    }
  },
})
