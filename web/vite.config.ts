import path from "path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    proxy: {
      "/teams": "http://127.0.0.1:3548",
      "/health": "http://127.0.0.1:3548",
      "/stream": "http://127.0.0.1:3548",
      "/metrics": "http://127.0.0.1:3548",
      "/network": "http://127.0.0.1:3548",
      "/config": "http://127.0.0.1:3548",
      "/bootstrap": "http://127.0.0.1:3548",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
