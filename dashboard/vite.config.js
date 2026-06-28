import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const apiBaseURL = process.env.VITE_PUBLIC_API_BASE_URL || "https://codexgo.yhklab.com";

export default defineConfig({
  base: "/dashboard/",
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          react: ["react", "react-dom", "@tanstack/react-query"],
          charts: ["recharts"],
          icons: ["lucide-react"],
        },
      },
    },
  },
  server: {
    port: 5174,
    proxy: {
      "/api": {
        target: apiBaseURL,
        changeOrigin: true,
      },
    },
  },
});
