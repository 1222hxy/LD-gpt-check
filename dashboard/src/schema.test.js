import { describe, expect, it } from "vitest";
import { buildDashboardPayload } from "./mock/dashboardPayload.js";
import { DashboardOverviewSchema } from "./schema.js";

describe("dashboard schema", () => {
  it("accepts the generated dashboard payload", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const parsed = DashboardOverviewSchema.parse(payload);

    expect(parsed.statistics.timeOfDay.hourly).toHaveLength(24);
    expect(parsed.hourlyBuckets).toHaveLength(24);
    expect(parsed.statistics.forecast.accuracy.forecast).toHaveLength(7);
    expect(parsed.statistics.modelRanking.length).toBe(parsed.modelBreakdown.length);
  });

  it("rejects payloads missing required statistical sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    delete payload.statistics.timeOfDay;

    expect(() => DashboardOverviewSchema.parse(payload)).toThrow();
  });
});
