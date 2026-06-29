import { describe, expect, it } from "vitest";
import {
  binomialConfidenceInterval,
  betaPosteriorSummary,
  buildStatistics,
  analyzeTimeOfDay,
  chiSquareGoodness,
  cohenH,
  correlationTest,
  linearTrendForecast,
  mean,
  minimumDetectableEffect,
  percentile,
  requiredSampleSizeForProportionDelta,
  standardDeviation,
  twoProportionZTest,
  welchTTest,
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

  it("summarizes a beta posterior with a bounded credible interval", () => {
    const posterior = betaPosteriorSummary(84, 100);

    expect(posterior.mean).toBeCloseTo(85 / 102, 4);
    expect(posterior.low).toBeGreaterThan(0.75);
    expect(posterior.high).toBeLessThan(0.91);
    expect(posterior.low).toBeLessThan(posterior.mean);
    expect(posterior.high).toBeGreaterThan(posterior.mean);
  });

  it("computes practical effect size and sample-size requirements", () => {
    const effect = cohenH(0.79, 0.84);
    const sampleSize = requiredSampleSizeForProportionDelta({ baselineRate: 0.84, delta: 0.02 });
    const mde = minimumDetectableEffect({ baselineRate: 0.84, sampleSize });

    expect(effect).toBeLessThan(0);
    expect(sampleSize).toBeGreaterThan(1000);
    expect(mde).toBeLessThanOrEqual(0.021);
  });

  it("detects time-of-day degradation and adjusted hourly risk", () => {
    const hourlyBuckets = Array.from({ length: 24 }, (_, hour) => ({
      hour,
      submissions: 20,
      attempts: 1000,
      accuracy: hour >= 2 && hour <= 4 ? 0.72 : 0.84,
      avgLatencySeconds: hour >= 2 && hour <= 4 ? 12 : 8,
    }));
    const result = analyzeTimeOfDay(hourlyBuckets);

    expect(result.omnibus.verdict).toBe("time_effect_detected");
    expect(result.worstHours[0].hour).toBeGreaterThanOrEqual(2);
    expect(result.worstHours[0].adjustedPValue).toBeLessThan(0.05);
    expect(result.degradationWindows[0].label).toBe("02:00-05:00");
  });

  it("uses chi-square omnibus testing across time buckets", () => {
    const result = chiSquareGoodness([
      { successes: 90, trials: 100 },
      { successes: 70, trials: 100 },
      { successes: 88, trials: 100 },
    ]);

    expect(result.statistic).toBeGreaterThan(0);
    expect(result.degreesOfFreedom).toBe(2);
    expect(result.pValue).toBeLessThan(0.01);
  });

  it("tests Pearson correlation with a Student t p-value", () => {
    const result = correlationTest([
      [1, 2],
      [2, 4],
      [3, 6],
      [4, 8],
      [5, 10],
    ]);

    expect(result.r).toBeCloseTo(1, 6);
    expect(result.pValue).toBeLessThan(0.001);
    expect(result.strength).toBe("strong");
  });

  it("forecasts a linear trend with residual bounds", () => {
    const result = linearTrendForecast(
      [
        [0, 10],
        [1, 12],
        [2, 14],
        [3, 16],
        [4, 18],
      ],
      3,
    );

    expect(result.slope).toBeCloseTo(2, 6);
    expect(result.rSquared).toBeCloseTo(1, 6);
    expect(result.forecast).toHaveLength(3);
    expect(result.forecast[0].value).toBeCloseTo(20, 6);
  });

  it("detects a mean shift with Welch's t-test", () => {
    const result = welchTTest([10, 11, 9, 10, 12], [18, 19, 20, 17, 18]);

    expect(result.delta).toBeGreaterThan(7);
    expect(result.tScore).toBeGreaterThan(8);
    expect(result.pValue).toBeLessThan(0.001);
  });
});

describe("dashboard statistics payload", () => {
  it("adds confidence, latency, regression, and test coverage sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });

    expect(payload.statistics.accuracy.sampleSize).toBeGreaterThan(0);
    expect(payload.statistics.accuracy.ci95Low).toBeLessThan(payload.statistics.accuracy.ci95High);
    expect(payload.statistics.latency.p95).toBeGreaterThanOrEqual(payload.statistics.latency.median);
    expect(payload.statistics.regression.verdict).toMatch(/stable|improved|regression/);
    expect(payload.statistics.power.minimumDetectableEffect).toBeGreaterThan(0);
    expect(payload.statistics.pairwiseTests.length).toBe(payload.modelBreakdown.length);
    expect(payload.statistics.trendStability.upperControlLimit).toBeGreaterThan(
      payload.statistics.trendStability.lowerControlLimit,
    );
    expect(payload.hourlyBuckets).toHaveLength(24);
    expect(payload.statistics.timeOfDay.hourly).toHaveLength(24);
    expect(payload.statistics.timeOfDay.omnibus.verdict).toMatch(/stable|time_effect_detected/);
    expect(payload.statistics.testCoverage.suites).toHaveLength(4);
    expect(payload.statistics.forecast.accuracy.forecast).toHaveLength(7);
    expect(payload.statistics.correlations).toHaveLength(4);
    expect(payload.statistics.modelRanking).toHaveLength(payload.modelBreakdown.length);
    expect(payload.statistics.questionDiagnostics).toHaveLength(payload.questionQuality.length);
    expect(payload.statistics.robustness.baselines.submissionAccuracyMedian).toBeGreaterThan(0);
    expect(payload.statistics.distributionShape.dailyAccuracy.iqr).toBeGreaterThanOrEqual(0);
    expect(payload.statistics.drift.ewma.series).toHaveLength(payload.trend.length);
    expect(payload.statistics.riskBudget.allowedFailures).toBeGreaterThan(0);
    expect(payload.statistics.efficiencyFrontier).toHaveLength(payload.modelBreakdown.length);
  });

  it("keeps model comparison scoped to the selected model", () => {
    const payload = buildDashboardPayload({ range: "7d", model: "gpt-5.5" });

    expect(payload.modelBreakdown).toHaveLength(1);
    expect(payload.statistics.modelComparisons).toHaveLength(1);
    expect(payload.statistics.modelComparisons[0].model).toBe("gpt-5.5");
    expect(payload.statistics.pairwiseTests).toHaveLength(0);
    expect(payload.statistics.modelRanking[0].verdict).toBe("insufficient");
    expect(payload.statistics.efficiencyFrontier[0].verdict).toBe("insufficient");
  });

  it("keeps DeepSeek official traffic out of unknown bridge mock data", () => {
    const payload = buildDashboardPayload({ range: "30d", channel: "official:deepseek" });

    expect(payload.filters.channel).toBe("official:deepseek");
    expect(payload.modelBreakdown.map((item) => item.model)).toEqual(["deepseek-r1"]);
    expect(payload.recentSubmissions.every((item) => item.codexChannel === "domestic_official")).toBe(true);
    expect(payload.recentSubmissions.every((item) => item.model === "deepseek-r1")).toBe(true);
    expect(payload.channels.find((item) => item.key === "official:deepseek")?.kind).toBe("domestic_official");
  });

  it("returns explicit insufficient statistics for incompatible model and channel filters", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "gpt-5.5", channel: "official:deepseek" });

    expect(payload.modelBreakdown).toHaveLength(0);
    expect(payload.recentSubmissions).toHaveLength(0);
    expect(payload.statistics.coverage.hasSubmissions).toBe(false);
    expect(payload.statistics.regression.verdict).toBe("insufficient");
    expect(payload.statistics.riskBudget.verdict).toBe("insufficient");
    expect(payload.statistics.pairwiseTests).toHaveLength(0);
  });

  it("keeps all statistical sections finite for empty payloads", () => {
    const stats = buildStatistics({
      trend: [],
      modelBreakdown: [],
      questionQuality: [],
      recentSubmissions: [],
      hourlyBuckets: [],
    });

    expect(stats.accuracy.sampleSize).toBe(0);
    expect(stats.latency.sampleSize).toBe(0);
    expect(stats.timeOfDay.omnibus.verdict).toBe("insufficient");
    expect(stats.drift.window.verdict).toBe("insufficient");
    expect(JSON.stringify(stats)).not.toMatch(/NaN|Infinity/);
  });

  it("can rebuild statistics from payload sections", () => {
    const payload = buildDashboardPayload({ range: "30d", model: "all" });
    const rebuilt = buildStatistics(payload);

    expect(rebuilt.accuracy.mean).toBe(payload.statistics.accuracy.mean);
    expect(rebuilt.testCoverage.totalAttempts).toBe(payload.statistics.testCoverage.totalAttempts);
  });
});
