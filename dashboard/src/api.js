import { DashboardOverviewSchema } from "./schema.js";

const API_BASE_URL = import.meta.env.VITE_PUBLIC_API_BASE_URL || "";

export async function fetchDashboardOverview(filters) {
  const params = new URLSearchParams({
    range: filters.range,
    model: filters.model,
  });
  const response = await fetch(`${API_BASE_URL}/api/dashboard/overview?${params}`);

  if (!response.ok) {
    throw new Error(`Dashboard API failed with ${response.status}`);
  }

  const payload = await response.json();
  return DashboardOverviewSchema.parse(payload);
}
