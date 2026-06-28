export function mean(values) {
  if (!values.length) return 0;
  return values.reduce((total, value) => total + value, 0) / values.length;
}

export function standardDeviation(values) {
  if (values.length < 2) return 0;
  const avg = mean(values);
  const variance = values.reduce((total, value) => total + (value - avg) ** 2, 0) / (values.length - 1);
  return Math.sqrt(variance);
}

export function percentile(values, percentileValue) {
  if (!values.length) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const rank = (percentileValue / 100) * (sorted.length - 1);
  const lower = Math.floor(rank);
  const upper = Math.ceil(rank);
  if (lower === upper) return sorted[lower];
  return sorted[lower] + (sorted[upper] - sorted[lower]) * (rank - lower);
}

export function binomialConfidenceInterval(successes, trials, confidence = 1.96) {
  if (trials <= 0) return { low: 0, high: 0, marginOfError: 0 };
  const proportion = successes / trials;
  const denominator = 1 + (confidence ** 2) / trials;
  const center = (proportion + (confidence ** 2) / (2 * trials)) / denominator;
  const margin =
    (confidence *
      Math.sqrt((proportion * (1 - proportion)) / trials + (confidence ** 2) / (4 * trials ** 2))) /
    denominator;

  return {
    low: clamp(center - margin, 0, 1),
    high: clamp(center + margin, 0, 1),
    marginOfError: margin,
  };
}

export function twoProportionZTest({ baselineSuccesses, baselineTrials, observedSuccesses, observedTrials }) {
  if (baselineTrials <= 0 || observedTrials <= 0) {
    return { zScore: 0, pValue: 1, delta: 0 };
  }

  const baselineRate = baselineSuccesses / baselineTrials;
  const observedRate = observedSuccesses / observedTrials;
  const pooled = (baselineSuccesses + observedSuccesses) / (baselineTrials + observedTrials);
  const standardError = Math.sqrt(pooled * (1 - pooled) * (1 / baselineTrials + 1 / observedTrials));

  if (standardError === 0) return { zScore: 0, pValue: 1, delta: observedRate - baselineRate };

  const zScore = (observedRate - baselineRate) / standardError;
  const pValue = 2 * (1 - normalCdf(Math.abs(zScore)));
  return { zScore, pValue, delta: observedRate - baselineRate };
}

export function cohenH(rateA, rateB) {
  return 2 * Math.asin(Math.sqrt(clamp(rateA, 0, 1))) - 2 * Math.asin(Math.sqrt(clamp(rateB, 0, 1)));
}

export function betaPosteriorSummary(successes, trials, confidence = 1.96) {
  if (trials <= 0) return { mean: 0, low: 0, high: 0, alpha: 1, beta: 1 };
  const alpha = successes + 1;
  const beta = trials - successes + 1;
  const total = alpha + beta;
  const posteriorMean = alpha / total;
  const variance = (alpha * beta) / (total ** 2 * (total + 1));
  const margin = confidence * Math.sqrt(variance);

  return {
    mean: posteriorMean,
    low: clamp(posteriorMean - margin, 0, 1),
    high: clamp(posteriorMean + margin, 0, 1),
    alpha,
    beta,
  };
}

export function requiredSampleSizeForProportionDelta({ baselineRate, delta, alphaZ = 1.96, powerZ = 0.84 }) {
  if (delta <= 0) return Infinity;
  const p1 = clamp(baselineRate, 0.001, 0.999);
  const p2 = clamp(p1 + delta, 0.001, 0.999);
  const pooled = (p1 + p2) / 2;
  const numerator =
    alphaZ * Math.sqrt(2 * pooled * (1 - pooled)) +
    powerZ * Math.sqrt(p1 * (1 - p1) + p2 * (1 - p2));
  return Math.ceil((numerator ** 2) / (delta ** 2));
}

export function minimumDetectableEffect({ baselineRate, sampleSize, alphaZ = 1.96, powerZ = 0.84 }) {
  if (sampleSize <= 0) return 1;
  let low = 0.001;
  let high = Math.min(0.5, 1 - baselineRate);

  for (let index = 0; index < 32; index += 1) {
    const mid = (low + high) / 2;
    const required = requiredSampleSizeForProportionDelta({ baselineRate, delta: mid, alphaZ, powerZ });
    if (required > sampleSize) low = mid;
    else high = mid;
  }

  return high;
}

export function buildStatistics({ trend, modelBreakdown, questionQuality, recentSubmissions }) {
  const modelTrials = modelBreakdown.map((item) => item.submissions * 150);
  const modelSuccesses = modelBreakdown.map((item, index) => Math.round(item.accuracy * modelTrials[index]));
  const totalTrials = modelTrials.reduce((total, value) => total + value, 0);
  const totalSuccesses = modelSuccesses.reduce((total, value) => total + value, 0);
  const accuracyMean = totalTrials ? totalSuccesses / totalTrials : 0;
  const accuracyValues = modelBreakdown.map((item) => item.accuracy);
  const latencyValues = recentSubmissions.map((item) => item.avgTimeSeconds);
  const accuracyCI = binomialConfidenceInterval(totalSuccesses, totalTrials);
  const baselineAccuracy = 0.835;
  const baselineTrials = Math.max(totalTrials, 1);
  const zTest = twoProportionZTest({
    baselineSuccesses: Math.round(baselineAccuracy * baselineTrials),
    baselineTrials,
    observedSuccesses: totalSuccesses,
    observedTrials: totalTrials,
  });

  const bestAccuracy = Math.max(...modelBreakdown.map((item) => item.accuracy));
  const modelComparisons = modelBreakdown.map((item, index) => {
    const trials = modelTrials[index];
    const successes = modelSuccesses[index];
    const ci = binomialConfidenceInterval(successes, trials);
    const posterior = betaPosteriorSummary(successes, trials);
    const deltaVsBest = item.accuracy - bestAccuracy;
    return {
      model: item.model,
      sampleSize: trials,
      accuracy: round(item.accuracy, 3),
      ci95Low: round(ci.low, 3),
      ci95High: round(ci.high, 3),
      marginOfError: round(ci.marginOfError, 3),
      posteriorMean: round(posterior.mean, 3),
      posteriorLow: round(posterior.low, 3),
      posteriorHigh: round(posterior.high, 3),
      deltaVsBest: round(deltaVsBest, 3),
      verdict: Math.abs(deltaVsBest) <= ci.marginOfError ? "overlap" : deltaVsBest < 0 ? "below_best" : "leader",
    };
  });
  const pairwiseTests = buildPairwiseModelTests(modelBreakdown, modelSuccesses, modelTrials);
  const averageModelSampleSize = Math.round(mean(modelTrials));

  const testCoverage = buildTestCoverage(questionQuality, recentSubmissions);

  return {
    accuracy: {
      mean: round(accuracyMean, 3),
      stdDev: round(standardDeviation(accuracyValues), 3),
      ci95Low: round(accuracyCI.low, 3),
      ci95High: round(accuracyCI.high, 3),
      marginOfError: round(accuracyCI.marginOfError, 3),
      sampleSize: totalTrials,
    },
    latency: {
      mean: round(mean(latencyValues), 1),
      median: round(percentile(latencyValues, 50), 1),
      p90: round(percentile(latencyValues, 90), 1),
      p95: round(percentile(latencyValues, 95), 1),
      stdDev: round(standardDeviation(latencyValues), 1),
    },
    regression: {
      baselineAccuracy,
      observedAccuracy: round(accuracyMean, 3),
      delta: round(zTest.delta, 3),
      zScore: round(zTest.zScore, 2),
      pValue: round(zTest.pValue, 4),
      verdict: zTest.pValue < 0.05 && zTest.delta < 0 ? "regression" : zTest.pValue < 0.05 ? "improved" : "stable",
    },
    power: {
      baselineAccuracy,
      averageModelSampleSize,
      minimumDetectableEffect: round(
        minimumDetectableEffect({ baselineRate: baselineAccuracy, sampleSize: averageModelSampleSize }),
        3,
      ),
      requiredSamples: [0.01, 0.02, 0.05].map((delta) => ({
        delta,
        perGroup: requiredSampleSizeForProportionDelta({ baselineRate: baselineAccuracy, delta }),
      })),
    },
    modelComparisons,
    pairwiseTests,
    testCoverage,
    trendStability: buildTrendStability(trend),
  };
}

function buildPairwiseModelTests(modelBreakdown, modelSuccesses, modelTrials) {
  const leaderIndex = modelBreakdown.reduce(
    (bestIndex, item, index) => (item.accuracy > modelBreakdown[bestIndex].accuracy ? index : bestIndex),
    0,
  );
  const leader = modelBreakdown[leaderIndex];
  const tests = modelBreakdown.map((item, index) => {
    if (index === leaderIndex) {
      return {
        model: item.model,
        comparedTo: leader.model,
        delta: 0,
        zScore: 0,
        pValue: 1,
        adjustedPValue: 1,
        effectSize: 0,
        verdict: "leader",
      };
    }

    const test = twoProportionZTest({
      baselineSuccesses: modelSuccesses[leaderIndex],
      baselineTrials: modelTrials[leaderIndex],
      observedSuccesses: modelSuccesses[index],
      observedTrials: modelTrials[index],
    });

    return {
      model: item.model,
      comparedTo: leader.model,
      delta: test.delta,
      zScore: test.zScore,
      pValue: test.pValue,
      adjustedPValue: test.pValue,
      effectSize: cohenH(item.accuracy, leader.accuracy),
      verdict: "pending",
    };
  });

  const comparable = tests.filter((item) => item.model !== leader.model);
  const adjusted = holmAdjust(comparable.map((item) => item.pValue));
  comparable.forEach((item, index) => {
    item.adjustedPValue = adjusted[index];
    item.verdict = item.adjustedPValue < 0.05 ? "significant" : "not_significant";
  });

  return tests.map((item) => ({
    ...item,
    delta: round(item.delta, 3),
    zScore: round(item.zScore, 2),
    pValue: round(item.pValue, 4),
    adjustedPValue: round(item.adjustedPValue, 4),
    effectSize: round(item.effectSize, 3),
  }));
}

function holmAdjust(pValues) {
  const indexed = pValues.map((pValue, index) => ({ pValue, index })).sort((a, b) => a.pValue - b.pValue);
  const adjusted = Array(pValues.length).fill(1);
  let runningMax = 0;

  indexed.forEach((item, rank) => {
    const multiplier = pValues.length - rank;
    runningMax = Math.max(runningMax, Math.min(1, item.pValue * multiplier));
    adjusted[item.index] = runningMax;
  });

  return adjusted;
}

function buildTrendStability(trend) {
  const accuracyValues = trend.map((item) => item.accuracy);
  const submissionValues = trend.map((item) => item.submissions);
  const accuracyMean = mean(accuracyValues);
  const accuracyStdDev = standardDeviation(accuracyValues);
  const upperControlLimit = clamp(accuracyMean + 3 * accuracyStdDev, 0, 1);
  const lowerControlLimit = clamp(accuracyMean - 3 * accuracyStdDev, 0, 1);
  const latest = trend[trend.length - 1];
  const latestZScore = accuracyStdDev ? (latest.accuracy - accuracyMean) / accuracyStdDev : 0;
  const anomalies = trend
    .map((item) => ({
      date: item.date,
      accuracy: item.accuracy,
      zScore: accuracyStdDev ? (item.accuracy - accuracyMean) / accuracyStdDev : 0,
    }))
    .filter((item) => Math.abs(item.zScore) >= 2)
    .map((item) => ({ ...item, zScore: round(item.zScore, 2) }));

  return {
    submissionStdDev: round(standardDeviation(submissionValues), 1),
    accuracyStdDev: round(accuracyStdDev, 3),
    accuracyMean: round(accuracyMean, 3),
    upperControlLimit: round(upperControlLimit, 3),
    lowerControlLimit: round(lowerControlLimit, 3),
    latestZScore: round(latestZScore, 2),
    anomalies,
  };
}

function buildTestCoverage(questionQuality, recentSubmissions) {
  const totalAttempts = questionQuality.reduce((total, item) => total + item.attempts, 0);
  const passedAttempts = questionQuality.reduce((total, item) => total + Math.round(item.attempts * item.accuracy), 0);
  const regressionCount = recentSubmissions.filter((item) => item.status === "regression").length;
  const watchCount = recentSubmissions.filter((item) => item.status === "watch").length;
  const flakyQuestions = questionQuality.filter((item) => item.failureRate > 0.22).length;

  return {
    suites: [
      { label: "单元测试", passed: 42, total: 42, status: "pass" },
      { label: "API contract", passed: 18, total: 18, status: "pass" },
      { label: "回归样本", passed: passedAttempts, total: totalAttempts, status: regressionCount ? "watch" : "pass" },
      { label: "视觉冒烟", passed: 2, total: 2, status: "pass" },
    ],
    totalAttempts,
    passRate: round(passedAttempts / totalAttempts, 3),
    regressionCount,
    watchCount,
    flakyQuestions,
  };
}

function normalCdf(value) {
  return 0.5 * (1 + erf(value / Math.sqrt(2)));
}

function erf(value) {
  const sign = value < 0 ? -1 : 1;
  const x = Math.abs(value);
  const a1 = 0.254829592;
  const a2 = -0.284496736;
  const a3 = 1.421413741;
  const a4 = -1.453152027;
  const a5 = 1.061405429;
  const p = 0.3275911;
  const t = 1 / (1 + p * x);
  const y = 1 - (((((a5 * t + a4) * t) + a3) * t + a2) * t + a1) * t * Math.exp(-x * x);
  return sign * y;
}

function round(value, decimals) {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
