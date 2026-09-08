import { fileURLToPath, URL } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  server: {
    proxy: process.env.WANT_KEEP_DEV_API
      ? {
          "/api/v1": {
            target: process.env.WANT_KEEP_DEV_API,
            changeOrigin: false,
          },
        }
      : undefined,
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
});
