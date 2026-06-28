import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { buildDashboardPayload } from "./src/mock/dashboardPayload.js";

function dashboardApiMock() {
  return {
    name: "dashboard-api-mock",
    configureServer(server) {
      server.middlewares.use("/api/dashboard/overview", (req, res) => {
        const url = new URL(req.url || "/", "http://localhost");
        const payload = buildDashboardPayload({
          range: url.searchParams.get("range") || "30d",
          model: url.searchParams.get("model") || "all",
        });

        res.statusCode = 200;
        res.setHeader("content-type", "application/json; charset=utf-8");
        res.end(JSON.stringify(payload));
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), dashboardApiMock()],
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
  },
});
