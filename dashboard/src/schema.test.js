import { describe, expect, it } from "vitest";
import { buildDashboardPayload } from "./mock/dashboardPayload.js";
import { DashboardOverviewSchema } from "./schema.js";

describe("dashboard schema", () => {
  it("accepts the generated dashboard payload", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const parsed = DashboardOverviewSchema.parse(payload);

    expect(parsed.statistics.timeOfDay.hourly).toHaveLength(24);
    expect(parsed.hourlyBuckets).toHaveLength(24);
    expect(parsed.filters.channels.some((channel) => channel.key === "official:deepseek")).toBe(true);
    expect(parsed.channels.some((channel) => channel.kind === "domestic_official")).toBe(true);
    expect(parsed.statistics.forecast.accuracy.forecast).toHaveLength(7);
    expect(parsed.statistics.modelRanking.length).toBe(parsed.modelBreakdown.length);
    expect(parsed.statistics.drift.cusum.series).toHaveLength(parsed.trend.length);
    expect(parsed.statistics.efficiencyFrontier.length).toBe(parsed.modelBreakdown.length);
  });

  it("rejects payloads missing required statistical sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    delete payload.statistics.timeOfDay;

    expect(() => DashboardOverviewSchema.parse(payload)).toThrow();
  });
});
