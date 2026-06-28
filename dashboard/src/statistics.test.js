import { describe, expect, it } from "vitest";
import {
  binomialConfidenceInterval,
  buildStatistics,
  mean,
  percentile,
  standardDeviation,
  twoProportionZTest,
} from "./statistics.js";
import { buildDashboardPayload } from "./mock/dashboardPayload.js";

describe("statistics helpers", () => {
  it("calculates mean, sample standard deviation, and percentile", () => {
    const values = [2, 4, 4, 4, 5, 5, 7, 9];

    expect(mean(values)).toBe(5);
    expect(standardDeviation(values)).toBeCloseTo(2.138, 3);
    expect(percentile(values, 50)).toBe(4.5);
    expect(percentile(values, 95)).toBeCloseTo(8.3, 1);
  });

  it("builds a bounded Wilson confidence interval", () => {
    const interval = binomialConfidenceInterval(84, 100);

    expect(interval.low).toBeGreaterThan(0.74);
    expect(interval.high).toBeLessThan(0.9);
    expect(interval.marginOfError).toBeGreaterThan(0);
  });

  it("detects a significant two-proportion delta", () => {
    const result = twoProportionZTest({
      baselineSuccesses: 840,
      baselineTrials: 1000,
      observedSuccesses: 790,
      observedTrials: 1000,
    });

    expect(result.delta).toBeCloseTo(-0.05, 3);
    expect(result.pValue).toBeLessThan(0.05);
  });
});

describe("dashboard statistics payload", () => {
  it("adds confidence, latency, regression, and test coverage sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });

    expect(payload.statistics.accuracy.sampleSize).toBeGreaterThan(0);
    expect(payload.statistics.accuracy.ci95Low).toBeLessThan(payload.statistics.accuracy.ci95High);
    expect(payload.statistics.latency.p95).toBeGreaterThanOrEqual(payload.statistics.latency.median);
    expect(payload.statistics.regression.verdict).toMatch(/stable|improved|regression/);
    expect(payload.statistics.testCoverage.suites).toHaveLength(4);
  });

  it("keeps model comparison scoped to the selected model", () => {
    const payload = buildDashboardPayload({ range: "7d", model: "gpt-5.5" });

    expect(payload.modelBreakdown).toHaveLength(1);
    expect(payload.statistics.modelComparisons).toHaveLength(1);
    expect(payload.statistics.modelComparisons[0].model).toBe("gpt-5.5");
  });

  it("can rebuild statistics from payload sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const rebuilt = buildStatistics(payload);

    expect(rebuilt.accuracy.mean).toBe(payload.statistics.accuracy.mean);
    expect(rebuilt.testCoverage.totalAttempts).toBe(payload.statistics.testCoverage.totalAttempts);
  });
});
