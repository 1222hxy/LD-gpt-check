import { describe, expect, it } from "vitest";
import { buildDashboardExport, buildHourlyCsv } from "./export.js";
import { buildDashboardPayload } from "./mock/dashboardPayload.js";

describe("dashboard export helpers", () => {
  it("builds a compact JSON export payload", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const exported = buildDashboardExport(payload, { range: "30d", model: "all" });

    expect(exported.summary.submissions).toBe(payload.summary.submissions);
    expect(exported.statistics.timeOfDay.hourly).toHaveLength(24);
  });

  it("builds hourly CSV with every hour", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const csv = buildHourlyCsv(payload.statistics.timeOfDay.hourly);

    expect(csv.split("\n")).toHaveLength(25);
    expect(csv).toContain("adjusted_p_value");
    expect(csv).toContain("05:00");
  });
});
