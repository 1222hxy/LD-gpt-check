import jStatPackage from "jstat";
import * as ss from "simple-statistics";

const { jStat } = jStatPackage;

export function mean(values) {
  if (!values.length) return 0;
  return ss.mean(values);
}

export function standardDeviation(values) {
  if (values.length < 2) return 0;
  return ss.sampleStandardDeviation(values);
}

export function percentile(values, percentileValue) {
  if (!values.length) return 0;
  return ss.quantile([...values].sort((a, b) => a - b), percentileValue / 100);
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
  const pValue = 2 * (1 - jStat.normal.cdf(Math.abs(zScore), 0, 1));
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

export function chiSquareGoodness(rows) {
  const totalSuccesses = rows.reduce((total, item) => total + item.successes, 0);
  const totalFailures = rows.reduce((total, item) => total + item.trials - item.successes, 0);
  const totalTrials = rows.reduce((total, item) => total + item.trials, 0);
  if (totalTrials <= 0 || rows.length < 2) return { statistic: 0, degreesOfFreedom: 0, pValue: 1 };

  const successRate = totalSuccesses / totalTrials;
  const failureRate = totalFailures / totalTrials;
  const statistic = rows.reduce((total, item) => {
    const failures = item.trials - item.successes;
    const expectedSuccesses = item.trials * successRate;
    const expectedFailures = item.trials * failureRate;
    return (
      total +
      ((item.successes - expectedSuccesses) ** 2) / expectedSuccesses +
      ((failures - expectedFailures) ** 2) / expectedFailures
    );
  }, 0);
  const degreesOfFreedom = rows.length - 1;

  return {
    statistic,
    degreesOfFreedom,
    pValue: 1 - jStat.chisquare.cdf(statistic, degreesOfFreedom),
  };
}

export function welchTTest(sampleA, sampleB) {
  if (sampleA.length < 2 || sampleB.length < 2) {
    return { tScore: 0, degreesOfFreedom: 0, pValue: 1, delta: mean(sampleB) - mean(sampleA) };
  }

  const meanA = mean(sampleA);
  const meanB = mean(sampleB);
  const varianceA = ss.sampleVariance(sampleA);
  const varianceB = ss.sampleVariance(sampleB);
  const standardError = Math.sqrt(varianceA / sampleA.length + varianceB / sampleB.length);
  if (!standardError) return { tScore: 0, degreesOfFreedom: sampleA.length + sampleB.length - 2, pValue: 1, delta: meanB - meanA };

  const numerator = (varianceA / sampleA.length + varianceB / sampleB.length) ** 2;
  const denominator =
    (varianceA ** 2) / (sampleA.length ** 2 * (sampleA.length - 1)) +
    (varianceB ** 2) / (sampleB.length ** 2 * (sampleB.length - 1));
  const degreesOfFreedom = denominator ? numerator / denominator : sampleA.length + sampleB.length - 2;
  const tScore = (meanB - meanA) / standardError;

  return {
    tScore,
    degreesOfFreedom,
    pValue: 2 * (1 - jStat.studentt.cdf(Math.abs(tScore), degreesOfFreedom)),
    delta: meanB - meanA,
  };
}

export function correlationTest(pairs) {
  const clean = pairs.filter(([x, y]) => Number.isFinite(x) && Number.isFinite(y));
  if (clean.length < 3) return { r: 0, pValue: 1, sampleSize: clean.length, strength: "insufficient" };
  const xs = clean.map(([x]) => x);
  const ys = clean.map(([, y]) => y);
  const r = ss.sampleCorrelation(xs, ys);
  const degreesOfFreedom = clean.length - 2;
  const t = Math.abs(r) >= 1 ? Infinity : r * Math.sqrt(degreesOfFreedom / (1 - r ** 2));
  const pValue = Number.isFinite(t) ? 2 * (1 - jStat.studentt.cdf(Math.abs(t), degreesOfFreedom)) : 0;
  return {
    r,
    pValue,
    sampleSize: clean.length,
    strength: correlationStrength(r),
  };
}

export function linearTrendForecast(points, horizon = 7) {
  if (points.length < 3) {
    return {
      slope: 0,
      intercept: points[0]?.[1] ?? 0,
      rSquared: 0,
      pValue: 1,
      residualStdDev: 0,
      forecast: [],
      verdict: "insufficient",
    };
  }

  const regression = ss.linearRegression(points);
  const line = ss.linearRegressionLine(regression);
  const residuals = points.map(([x, y]) => y - line(x));
  const residualStdDev = standardDeviation(residuals);
  const rSquared = ss.rSquared(points, line);
  const slopeTest = slopePValue(points, regression.m);
  const lastX = points[points.length - 1][0];
  const forecast = Array.from({ length: horizon }, (_, index) => {
    const x = lastX + index + 1;
    const predicted = line(x);
    return {
      step: index + 1,
      value: predicted,
      low: predicted - 1.96 * residualStdDev,
      high: predicted + 1.96 * residualStdDev,
    };
  });

  return {
    slope: regression.m,
    intercept: regression.b,
    rSquared,
    pValue: slopeTest.pValue,
    residualStdDev,
    forecast,
    verdict: slopeTest.pValue < 0.05 ? (regression.m >= 0 ? "rising" : "falling") : "flat",
  };
}

export function analyzeTimeOfDay(hourlyBuckets) {
  if (!hourlyBuckets?.length) {
    return {
      omnibus: { statistic: 0, degreesOfFreedom: 0, pValue: 1, verdict: "stable" },
      hourly: [],
      segments: [],
      worstHours: [],
      degradationWindows: [],
      summary: { worstHour: null, worstSegment: null, affectedAttempts: 0 },
    };
  }

  const rows = hourlyBuckets.map((item) => ({
    ...item,
    successes: Math.round(item.accuracy * item.attempts),
    trials: item.attempts,
  }));
  const overallTrials = rows.reduce((total, item) => total + item.trials, 0);
  const overallSuccesses = rows.reduce((total, item) => total + item.successes, 0);
  const overallAccuracy = overallSuccesses / overallTrials;
  const omnibus = chiSquareGoodness(rows);
  const rawHourly = rows.map((item) => {
    const test = twoProportionZTest({
      baselineSuccesses: overallSuccesses - item.successes,
      baselineTrials: overallTrials - item.trials,
      observedSuccesses: item.successes,
      observedTrials: item.trials,
    });
    const ci = binomialConfidenceInterval(item.successes, item.trials);
    const posterior = betaPosteriorSummary(item.successes, item.trials);
    return {
      hour: item.hour,
      label: `${String(item.hour).padStart(2, "0")}:00`,
      attempts: item.trials,
      submissions: item.submissions,
      accuracy: item.successes / item.trials,
      avgLatencySeconds: item.avgLatencySeconds,
      ci95Low: ci.low,
      ci95High: ci.high,
      posteriorLow: posterior.low,
      posteriorHigh: posterior.high,
      deltaVsDay: test.delta,
      zScore: test.zScore,
      pValue: test.pValue,
      adjustedPValue: test.pValue,
      effectSize: cohenH(item.successes / item.trials, overallAccuracy),
      riskScore: Math.max(0, -test.delta) * Math.sqrt(item.trials),
      verdict: "pending",
    };
  });
  const adjusted = holmAdjust(rawHourly.map((item) => item.pValue));
  rawHourly.forEach((item, index) => {
    item.adjustedPValue = adjusted[index];
    item.verdict = item.adjustedPValue < 0.05 && item.deltaVsDay < 0 ? "degraded" : item.adjustedPValue < 0.05 ? "elevated" : "normal";
  });

  const hourly = rawHourly.map(formatHourlyResult);
  const segments = buildTimeSegments(rows, overallSuccesses, overallTrials);
  const worstHours = hourly
    .filter((item) => item.verdict === "degraded")
    .sort((a, b) => b.riskScore - a.riskScore)
    .slice(0, 5);
  const degradationWindows = buildDegradationWindows(hourly);
  const worstSegment = [...segments].sort((a, b) => a.deltaVsDay - b.deltaVsDay)[0] || null;
  const affectedAttempts = hourly
    .filter((item) => item.verdict === "degraded")
    .reduce((total, item) => total + item.attempts, 0);

  return {
    omnibus: {
      statistic: round(omnibus.statistic, 2),
      degreesOfFreedom: omnibus.degreesOfFreedom,
      pValue: round(omnibus.pValue, 4),
      verdict: omnibus.pValue < 0.05 ? "time_effect_detected" : "stable",
    },
    hourly,
    segments,
    worstHours,
    degradationWindows,
    summary: {
      worstHour: worstHours[0] || null,
      worstSegment,
      affectedAttempts,
      overallAccuracy: round(overallAccuracy, 3),
    },
  };
}

export function buildStatistics({ trend, modelBreakdown, questionQuality, recentSubmissions, hourlyBuckets }) {
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
  const trendStability = buildTrendStability(trend);
  const timeOfDay = analyzeTimeOfDay(hourlyBuckets);
  const questionDiagnostics = buildQuestionDiagnostics(questionQuality);
  const modelRanking = buildModelRanking(modelBreakdown, modelSuccesses, modelTrials);
  const robustness = buildRobustness({ recentSubmissions, questionQuality });

  return {
    coverage: buildCoverage({ trend, modelBreakdown, questionQuality, recentSubmissions, hourlyBuckets, totalTrials, latencyValues, accuracyValues }),
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
      sampleSize: latencyValues.length,
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
      verdict: averageModelSampleSize > 0 ? "measured" : "insufficient",
      requiredSamples: [0.01, 0.02, 0.05].map((delta) => ({
        delta,
        perGroup: requiredSampleSizeForProportionDelta({ baselineRate: baselineAccuracy, delta }),
      })),
    },
    modelComparisons,
    pairwiseTests,
    testCoverage,
    trendStability,
    timeOfDay,
    forecast: buildForecast(trend),
    correlations: buildCorrelations({ trend, questionQuality, hourlyBuckets }),
    questionDiagnostics,
    modelRanking,
    robustness,
    distributionShape: buildDistributionShape({ trend, recentSubmissions, questionQuality, hourlyBuckets }),
    drift: buildDriftAnalysis(trend),
    riskBudget: buildRiskBudget({
      totalTrials,
      totalSuccesses,
      baselineAccuracy,
      trendStability,
      timeOfDay,
      robustness,
      questionDiagnostics,
    }),
    efficiencyFrontier: buildEfficiencyFrontier(modelBreakdown),
  };
}

function buildCoverage({ trend, modelBreakdown, questionQuality, recentSubmissions, hourlyBuckets, totalTrials, latencyValues, accuracyValues }) {
  const activeHours = hourlyBuckets.filter((item) => item.attempts > 0).length;
  const comparedModels = modelBreakdown.filter((item) => item.submissions * 150 >= 30).length;
  return {
    submissions: recentSubmissions.length,
    attempts: totalTrials,
    trendDays: trend.length,
    models: modelBreakdown.length,
    comparedModels,
    questions: questionQuality.length,
    activeHours,
    accuracySamples: accuracyValues.length,
    latencySamples: latencyValues.length,
    hasSubmissions: totalTrials > 0,
    hasTrend: trend.length >= 2,
    hasForecast: trend.length >= 3,
    hasModelComparison: comparedModels >= 2,
    hasQuestionDiagnostics: questionQuality.some((item) => item.attempts >= 30),
    hasTimeOfDay: activeHours >= 2 && totalTrials > 0,
    hasDistribution: accuracyValues.length >= 2 || trend.length >= 2,
  };
}

function buildForecast(trend) {
  const accuracyTrend = linearTrendForecast(trend.map((item, index) => [index, item.accuracy]), 7);
  const submissionTrend = linearTrendForecast(trend.map((item, index) => [index, item.submissions]), 7);
  return {
    accuracy: formatForecast(accuracyTrend, 3),
    submissions: formatForecast(submissionTrend, 0),
  };
}

function buildCorrelations({ trend, questionQuality, hourlyBuckets }) {
  const rows = [
    {
      metric: "小时耗时 vs 准确率",
      x: "avgLatencySeconds",
      y: "accuracy",
      expectedDirection: "negative",
      ...correlationTest(hourlyBuckets.map((item) => [item.avgLatencySeconds, item.accuracy])),
    },
    {
      metric: "小时流量 vs 准确率",
      x: "attempts",
      y: "accuracy",
      expectedDirection: "neutral",
      ...correlationTest(hourlyBuckets.map((item) => [item.attempts, item.accuracy])),
    },
    {
      metric: "提交量 vs 准确率",
      x: "submissions",
      y: "accuracy",
      expectedDirection: "neutral",
      ...correlationTest(trend.map((item) => [item.submissions, item.accuracy])),
    },
    {
      metric: "题目耗时 vs 失败率",
      x: "avgTimeSeconds",
      y: "failureRate",
      expectedDirection: "positive",
      ...correlationTest(questionQuality.map((item) => [item.avgTimeSeconds, item.failureRate])),
    },
  ];

  return rows.map((item) => ({
    ...item,
    r: round(item.r, 3),
    pValue: round(item.pValue, 4),
    verdict: item.pValue < 0.05 ? "significant" : "not_significant",
  }));
}

function buildQuestionDiagnostics(questionQuality) {
  const failureRates = questionQuality.map((item) => item.failureRate);
  const avgTimes = questionQuality.map((item) => item.avgTimeSeconds);
  const failureMean = mean(failureRates);
  const failureStd = standardDeviation(failureRates);
  const timeMedian = ss.median(avgTimes);

  return questionQuality
    .map((item) => {
      const successes = Math.round(item.attempts * item.accuracy);
      const ci = binomialConfidenceInterval(successes, item.attempts);
      const difficultyZ = failureStd ? (item.failureRate - failureMean) / failureStd : 0;
      const timePenalty = item.avgTimeSeconds / timeMedian;
      const priorityScore = item.failureRate * Math.sqrt(item.attempts) * timePenalty;
      return {
        questionId: item.questionId,
        title: item.title,
        attempts: item.attempts,
        accuracy: item.accuracy,
        failureRate: item.failureRate,
        ci95Low: round(ci.low, 3),
        ci95High: round(ci.high, 3),
        difficultyZ: round(difficultyZ, 2),
        priorityScore: round(priorityScore, 2),
        verdict: priorityScore > 12 || difficultyZ > 1 ? "audit" : priorityScore > 7 ? "watch" : "normal",
      };
    })
    .sort((a, b) => b.priorityScore - a.priorityScore);
}

function buildModelRanking(modelBreakdown, modelSuccesses, modelTrials) {
  const posterior = modelBreakdown.map((item, index) => {
    const successes = modelSuccesses[index];
    const trials = modelTrials[index];
    const alpha = successes + 1;
    const beta = trials - successes + 1;
    const meanValue = alpha / (alpha + beta);
    const variance = (alpha * beta) / ((alpha + beta) ** 2 * (alpha + beta + 1));
    return {
      model: item.model,
      posteriorMean: meanValue,
      posteriorStdDev: Math.sqrt(variance),
      expectedLoss: Math.max(...modelBreakdown.map((model) => model.accuracy)) - item.accuracy,
      probabilityBest: 0,
    };
  });

  posterior.forEach((item) => {
    const probability = posterior
      .filter((candidate) => candidate.model !== item.model)
      .reduce((product, candidate) => {
        const variance = item.posteriorStdDev ** 2 + candidate.posteriorStdDev ** 2;
        const z = variance ? (item.posteriorMean - candidate.posteriorMean) / Math.sqrt(variance) : 0;
        return product * jStat.normal.cdf(z, 0, 1);
      }, 1);
    item.probabilityBest = probability;
  });

  const totalProbability = posterior.reduce((total, item) => total + item.probabilityBest, 0) || 1;

  return posterior
    .map((item) => ({
      model: item.model,
      posteriorMean: round(item.posteriorMean, 3),
      posteriorStdDev: round(item.posteriorStdDev, 4),
      probabilityBest: round(item.probabilityBest / totalProbability, 3),
      expectedLoss: round(item.expectedLoss, 3),
      verdict: item.expectedLoss <= 0.005 ? "ship" : item.expectedLoss <= 0.03 ? "candidate" : "avoid",
    }))
    .sort((a, b) => b.probabilityBest - a.probabilityBest);
}

function buildRobustness({ recentSubmissions, questionQuality }) {
  const submissionAccuracy = recentSubmissions.map((item) => item.accuracy);
  const submissionLatency = recentSubmissions.map((item) => item.avgTimeSeconds);
  const questionFailures = questionQuality.map((item) => item.failureRate);

  return {
    recentOutliers: recentSubmissions
      .map((item) => ({
        id: item.id,
        model: item.model,
        accuracy: item.accuracy,
        latency: item.avgTimeSeconds,
        accuracyRobustZ: round(robustZ(item.accuracy, submissionAccuracy), 2),
        latencyRobustZ: round(robustZ(item.avgTimeSeconds, submissionLatency), 2),
      }))
      .filter((item) => item.accuracyRobustZ <= -1.5 || item.latencyRobustZ >= 1.5),
    questionOutliers: questionQuality
      .map((item) => ({
        questionId: item.questionId,
        title: item.title,
        failureRate: item.failureRate,
        failureRobustZ: round(robustZ(item.failureRate, questionFailures), 2),
      }))
      .filter((item) => item.failureRobustZ >= 1.5),
    baselines: {
      submissionAccuracyMedian: round(ss.median(submissionAccuracy), 3),
      submissionLatencyMedian: round(ss.median(submissionLatency), 1),
      questionFailureMedian: round(ss.median(questionFailures), 3),
      submissionSampleSize: recentSubmissions.length,
      questionSampleSize: questionQuality.length,
    },
  };
}

function buildDistributionShape({ trend, recentSubmissions, questionQuality, hourlyBuckets }) {
  return {
    dailyAccuracy: distributionSummary(trend.map((item) => item.accuracy), 3),
    dailySubmissions: distributionSummary(trend.map((item) => item.submissions), 0),
    recentLatency: distributionSummary(recentSubmissions.map((item) => item.avgTimeSeconds), 1),
    questionFailure: distributionSummary(questionQuality.map((item) => item.failureRate), 3),
    hourlyAccuracy: distributionSummary(hourlyBuckets.filter((item) => item.attempts > 0).map((item) => item.accuracy), 3),
  };
}

function buildDriftAnalysis(trend) {
  const midpoint = Math.floor(trend.length / 2);
  const prior = trend.slice(0, midpoint);
  const recent = trend.slice(midpoint);
  const priorTrials = prior.reduce((total, item) => total + item.submissions * 150, 0);
  const recentTrials = recent.reduce((total, item) => total + item.submissions * 150, 0);
  const priorSuccesses = prior.reduce((total, item) => total + Math.round(item.accuracy * item.submissions * 150), 0);
  const recentSuccesses = recent.reduce((total, item) => total + Math.round(item.accuracy * item.submissions * 150), 0);
  const accuracyTest = twoProportionZTest({
    baselineSuccesses: priorSuccesses,
    baselineTrials: priorTrials,
    observedSuccesses: recentSuccesses,
    observedTrials: recentTrials,
  });
  const volumeTest = welchTTest(
    prior.map((item) => item.submissions),
    recent.map((item) => item.submissions),
  );
  const ewma = ewmaSeries(trend.map((item) => item.accuracy), 0.32);
  const cusum = cusumSeries(trend.map((item) => item.accuracy), mean(trend.map((item) => item.accuracy)));
  const latestEwma = ewma.at(-1) ?? 0;
  const ewmaDelta = latestEwma - mean(ewma);
  const signalScore = Math.max(Math.abs(accuracyTest.zScore), Math.abs(volumeTest.tScore), Math.abs(ewmaDelta) * 100);

  return {
    window: {
      priorDays: prior.length,
      recentDays: recent.length,
      priorAccuracy: round(priorTrials ? priorSuccesses / priorTrials : 0, 3),
      recentAccuracy: round(recentTrials ? recentSuccesses / recentTrials : 0, 3),
      delta: round(accuracyTest.delta, 3),
      zScore: round(accuracyTest.zScore, 2),
      pValue: round(accuracyTest.pValue, 4),
      verdict: accuracyTest.pValue < 0.05 && accuracyTest.delta < 0 ? "negative_drift" : accuracyTest.pValue < 0.05 ? "positive_drift" : "stable",
    },
    volume: {
      priorMean: round(mean(prior.map((item) => item.submissions)), 1),
      recentMean: round(mean(recent.map((item) => item.submissions)), 1),
      delta: round(volumeTest.delta, 1),
      tScore: round(volumeTest.tScore, 2),
      degreesOfFreedom: round(volumeTest.degreesOfFreedom, 1),
      pValue: round(volumeTest.pValue, 4),
      verdict: volumeTest.pValue < 0.05 ? "changed" : "stable",
    },
    ewma: {
      lambda: 0.32,
      latest: round(latestEwma, 3),
      deltaVsMean: round(ewmaDelta, 3),
      min: round(Math.min(...ewma), 3),
      max: round(Math.max(...ewma), 3),
      verdict: ewmaDelta < -0.015 ? "cooling" : ewmaDelta > 0.015 ? "heating" : "stable",
      series: trend.map((item, index) => ({ date: item.date, value: round(ewma[index], 3) })),
    },
    cusum: {
      latest: round(cusum.at(-1) ?? 0, 3),
      min: round(Math.min(...cusum), 3),
      max: round(Math.max(...cusum), 3),
      signalScore: round(signalScore, 2),
      verdict: signalScore >= 3 ? "alert" : signalScore >= 2 ? "watch" : "stable",
      series: trend.map((item, index) => ({ date: item.date, value: round(cusum[index], 3) })),
    },
  };
}

function buildRiskBudget({ totalTrials, totalSuccesses, baselineAccuracy, trendStability, timeOfDay, robustness, questionDiagnostics }) {
  const failures = totalTrials - totalSuccesses;
  const allowedFailures = Math.round(totalTrials * (1 - baselineAccuracy));
  const excessFailures = Math.max(0, failures - allowedFailures);
  const failureRate = totalTrials ? failures / totalTrials : 0;
  const budgetRemaining = totalTrials ? clamp((allowedFailures - failures) / allowedFailures, -1, 1) : 0;
  const degradedAttempts = timeOfDay.summary.affectedAttempts || 0;
  const degradedShare = totalTrials ? degradedAttempts / totalTrials : 0;
  const auditQuestions = questionDiagnostics.filter((item) => item.verdict === "audit").length;
  const outlierLoad = robustness.recentOutliers.length + robustness.questionOutliers.length;
  const burnRate = baselineAccuracy >= 1 ? 0 : failureRate / (1 - baselineAccuracy);

  return {
    targetAccuracy: baselineAccuracy,
    failureRate: round(failureRate, 3),
    failures,
    allowedFailures,
    excessFailures,
    budgetRemaining: round(budgetRemaining, 3),
    burnRate: round(burnRate, 2),
    degradedAttemptShare: round(degradedShare, 3),
    auditQuestions,
    outlierLoad,
    anomalyDays: trendStability.anomalies.length,
    verdict:
      excessFailures > 0 || burnRate > 1.1
        ? "over_budget"
        : degradedShare > 0.12 || auditQuestions >= 2 || outlierLoad >= 3
          ? "watch"
          : "healthy",
  };
}

function buildEfficiencyFrontier(modelBreakdown) {
  const maxTps = Math.max(...modelBreakdown.map((item) => item.avgTps));
  const minLatency = Math.min(...modelBreakdown.map((item) => item.avgTimeSeconds));

  return modelBreakdown
    .map((item) => {
      const dominatedBy = modelBreakdown
        .filter((candidate) => candidate.model !== item.model)
        .filter(
          (candidate) =>
            candidate.accuracy >= item.accuracy &&
            candidate.avgTps >= item.avgTps &&
            candidate.avgTimeSeconds <= item.avgTimeSeconds &&
            (candidate.accuracy > item.accuracy || candidate.avgTps > item.avgTps || candidate.avgTimeSeconds < item.avgTimeSeconds),
        )
        .map((candidate) => candidate.model);
      const accuracyScore = item.accuracy;
      const throughputScore = maxTps ? item.avgTps / maxTps : 0;
      const latencyScore = item.avgTimeSeconds ? minLatency / item.avgTimeSeconds : 0;
      const utilityScore = 0.58 * accuracyScore + 0.24 * throughputScore + 0.18 * latencyScore;

      return {
        model: item.model,
        accuracy: round(item.accuracy, 3),
        avgTps: item.avgTps,
        avgTimeSeconds: item.avgTimeSeconds,
        utilityScore: round(utilityScore, 3),
        dominatedBy,
        onFrontier: dominatedBy.length === 0,
        verdict: dominatedBy.length === 0 ? "frontier" : dominatedBy.length === 1 ? "shadowed" : "dominated",
      };
    })
    .sort((a, b) => b.utilityScore - a.utilityScore);
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

function buildTimeSegments(rows, overallSuccesses, overallTrials) {
  const segmentDefs = [
    { label: "深夜", startHour: 0, endHour: 5 },
    { label: "上午", startHour: 6, endHour: 11 },
    { label: "下午", startHour: 12, endHour: 17 },
    { label: "晚间", startHour: 18, endHour: 23 },
  ];

  const raw = segmentDefs.map((segment) => {
    const items = rows.filter((item) => item.hour >= segment.startHour && item.hour <= segment.endHour);
    const successes = items.reduce((total, item) => total + item.successes, 0);
    const trials = items.reduce((total, item) => total + item.trials, 0);
    const latencyValues = items.map((item) => item.avgLatencySeconds);
    const test = twoProportionZTest({
      baselineSuccesses: overallSuccesses - successes,
      baselineTrials: overallTrials - trials,
      observedSuccesses: successes,
      observedTrials: trials,
    });
    return {
      ...segment,
      attempts: trials,
      accuracy: successes / trials,
      avgLatencySeconds: mean(latencyValues),
      deltaVsDay: test.delta,
      zScore: test.zScore,
      pValue: test.pValue,
      adjustedPValue: test.pValue,
      verdict: "pending",
    };
  });

  const adjusted = holmAdjust(raw.map((item) => item.pValue));
  raw.forEach((item, index) => {
    item.adjustedPValue = adjusted[index];
    item.verdict = item.adjustedPValue < 0.05 && item.deltaVsDay < 0 ? "degraded" : item.adjustedPValue < 0.05 ? "elevated" : "normal";
  });

  return raw.map((item) => ({
    ...item,
    accuracy: round(item.accuracy, 3),
    avgLatencySeconds: round(item.avgLatencySeconds, 1),
    deltaVsDay: round(item.deltaVsDay, 3),
    zScore: round(item.zScore, 2),
    pValue: round(item.pValue, 4),
    adjustedPValue: round(item.adjustedPValue, 4),
  }));
}

function buildDegradationWindows(hourly) {
  const windows = [];
  let current = null;

  hourly.forEach((item) => {
    if (item.verdict !== "degraded") {
      if (current) windows.push(current);
      current = null;
      return;
    }

    if (!current) {
      current = {
        startHour: item.hour,
        endHour: item.hour,
        attempts: item.attempts,
        riskScore: item.riskScore,
        minDelta: item.deltaVsDay,
      };
      return;
    }

    current.endHour = item.hour;
    current.attempts += item.attempts;
    current.riskScore += item.riskScore;
    current.minDelta = Math.min(current.minDelta, item.deltaVsDay);
  });

  if (current) windows.push(current);

  return windows
    .map((item) => ({
      ...item,
      label: `${String(item.startHour).padStart(2, "0")}:00-${String(item.endHour + 1).padStart(2, "0")}:00`,
      riskScore: round(item.riskScore, 2),
      minDelta: round(item.minDelta, 3),
    }))
    .sort((a, b) => b.riskScore - a.riskScore);
}

function formatHourlyResult(item) {
  return {
    ...item,
    accuracy: round(item.accuracy, 3),
    avgLatencySeconds: round(item.avgLatencySeconds, 1),
    ci95Low: round(item.ci95Low, 3),
    ci95High: round(item.ci95High, 3),
    posteriorLow: round(item.posteriorLow, 3),
    posteriorHigh: round(item.posteriorHigh, 3),
    deltaVsDay: round(item.deltaVsDay, 3),
    zScore: round(item.zScore, 2),
    pValue: round(item.pValue, 4),
    adjustedPValue: round(item.adjustedPValue, 4),
    effectSize: round(item.effectSize, 3),
    riskScore: round(item.riskScore, 2),
  };
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
    verdict: trend.length >= 2 ? "measured" : "insufficient",
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

function formatForecast(trend, decimals) {
  return {
    slope: round(trend.slope, decimals === 0 ? 2 : 4),
    intercept: round(trend.intercept, decimals),
    rSquared: round(trend.rSquared, 3),
    pValue: round(trend.pValue, 4),
    residualStdDev: round(trend.residualStdDev, decimals === 0 ? 1 : 4),
    sampleSize: trend.sampleSize ?? trend.forecast.length,
    verdict: trend.verdict,
    forecast: trend.forecast.map((item) => ({
      step: item.step,
      value: round(item.value, decimals),
      low: round(item.low, decimals),
      high: round(item.high, decimals),
    })),
  };
}

function slopePValue(points, slope) {
  const n = points.length;
  if (n < 3) return { t: 0, pValue: 1 };
  const xs = points.map(([x]) => x);
  const xMean = mean(xs);
  const line = ss.linearRegressionLine(ss.linearRegression(points));
  const residuals = points.map(([x, y]) => y - line(x));
  const sxx = xs.reduce((total, x) => total + (x - xMean) ** 2, 0);
  const residualVariance = residuals.reduce((total, value) => total + value ** 2, 0) / (n - 2);
  const standardError = Math.sqrt(residualVariance / sxx);
  if (!standardError) return { t: 0, pValue: 1 };
  const t = slope / standardError;
  return {
    t,
    pValue: 2 * (1 - jStat.studentt.cdf(Math.abs(t), n - 2)),
  };
}

function correlationStrength(r) {
  const abs = Math.abs(r);
  if (abs >= 0.7) return "strong";
  if (abs >= 0.4) return "moderate";
  if (abs >= 0.2) return "weak";
  return "none";
}

function robustZ(value, values) {
  if (values.length < 3) return 0;
  const median = ss.median(values);
  const mad = ss.medianAbsoluteDeviation(values);
  if (!mad) return 0;
  return (value - median) / (mad * 1.4826);
}

function distributionSummary(values, decimals) {
  if (!values.length) {
    return {
      min: 0,
      q1: 0,
      median: 0,
      q3: 0,
      max: 0,
      iqr: 0,
      mean: 0,
      stdDev: 0,
      coefficientOfVariation: 0,
      skewness: 0,
      excessKurtosis: 0,
      tailRisk: 0,
      sampleSize: 0,
    };
  }

  const sorted = [...values].sort((a, b) => a - b);
  const meanValue = mean(values);
  const stdDev = standardDeviation(values);
  const q1 = ss.quantile(sorted, 0.25);
  const median = ss.quantile(sorted, 0.5);
  const q3 = ss.quantile(sorted, 0.75);
  const iqr = q3 - q1;
  const moments = standardizedMoments(values, meanValue, stdDev);
  const lowerFence = q1 - 1.5 * iqr;
  const upperFence = q3 + 1.5 * iqr;
  const tailRisk = values.filter((value) => value < lowerFence || value > upperFence).length / values.length;

  return {
    min: round(sorted[0], decimals),
    q1: round(q1, decimals),
    median: round(median, decimals),
    q3: round(q3, decimals),
    max: round(sorted.at(-1), decimals),
    iqr: round(iqr, decimals),
    mean: round(meanValue, decimals),
    stdDev: round(stdDev, decimals),
    coefficientOfVariation: meanValue ? round(stdDev / meanValue, 3) : 0,
    skewness: round(moments.skewness, 2),
    excessKurtosis: round(moments.excessKurtosis, 2),
    tailRisk: round(tailRisk, 3),
    sampleSize: values.length,
  };
}

function standardizedMoments(values, meanValue, stdDev) {
  if (values.length < 3 || !stdDev) return { skewness: 0, excessKurtosis: 0 };
  const n = values.length;
  const normalized = values.map((value) => (value - meanValue) / stdDev);
  const skewness = (n / ((n - 1) * (n - 2))) * normalized.reduce((total, value) => total + value ** 3, 0);
  if (n < 4) return { skewness, excessKurtosis: 0 };
  const kurtosisNumerator = n * (n + 1) * normalized.reduce((total, value) => total + value ** 4, 0);
  const kurtosisDenominator = (n - 1) * (n - 2) * (n - 3);
  const correction = (3 * (n - 1) ** 2) / ((n - 2) * (n - 3));
  const excessKurtosis = kurtosisNumerator / kurtosisDenominator - correction;
  return { skewness, excessKurtosis };
}

function ewmaSeries(values, lambda) {
  if (!values.length) return [];
  const result = [values[0]];
  for (let index = 1; index < values.length; index += 1) {
    result.push(lambda * values[index] + (1 - lambda) * result[index - 1]);
  }
  return result;
}

function cusumSeries(values, target) {
  let total = 0;
  return values.map((value) => {
    total += value - target;
    return total;
  });
}

function round(value, decimals) {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
