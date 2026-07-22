import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev-сервер на :3000, проксирование API на backend :8080 (см. AGENTS.md).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      "/api": {
        // 127.0.0.1 — иначе на macOS Vite ходит на ::1, а Go слушает IPv4 → ECONNREFUSED/500
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
});
