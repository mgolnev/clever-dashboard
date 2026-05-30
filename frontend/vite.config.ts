import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev-сервер на :3000, проксирование API на backend :8080 (см. AGENTS.md).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
