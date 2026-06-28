# Dashboard API Contract

The dashboard frontend calls one read-only endpoint:

```text
GET /api/dashboard/overview?range=30d&model=all
```

For local development, `dashboard/vite.config.js` proxies this endpoint to a real Cloudflare Worker API. In production, the Worker builds the response from D1 tables:

- `benchmark_submissions`
- `benchmark_question_results`
- `benchmark_attempts`
- `users`

## Query Parameters

| Name | Values | Default | Description |
| --- | --- | --- | --- |
| `range` | `7d`, `30d`, `90d` | `30d` | Time window for trend and aggregate metrics. |
| `model` | `all` or model id | `all` | Optional model filter. |

## Response

```json
{
  "updatedAt": "2026-06-28T13:40:00.000Z",
  "filters": {
    "range": "30d",
    "model": "all",
    "models": ["gpt-5.5", "gpt-5.5-mini", "o4-mini"]
  },
  "summary": {
    "submissions": 1284,
    "activeUsers": 214,
    "averageAccuracy": 0.842,
    "averageLatencySeconds": 8.7,
    "averageTps": 35.8,
    "tokenTotal": 9124300
  },
  "trend": [
    {
      "date": "2026-06-01",
      "submissions": 43,
      "accuracy": 0.81,
      "avgTps": 33.4,
      "tokens": 281000
    }
  ],
  "modelBreakdown": [
    {
      "model": "gpt-5.5",
      "submissions": 511,
      "accuracy": 0.882,
      "avgTps": 39.4,
      "avgTimeSeconds": 7.9
    }
  ],
  "questionQuality": [
    {
      "questionId": "math-014",
      "title": "Ratio reasoning",
      "accuracy": 0.76,
      "attempts": 144,
      "avgTimeSeconds": 10.2,
      "failureRate": 0.24
    }
  ],
  "recentSubmissions": [
    {
      "id": "sub_01",
      "user": {
        "anonymous": false,
        "display_name": "alice",
        "username": "alice",
        "avatar_url": "https://cdn.ldstatic.com/user_avatar/linux.do/alice/288/170339_2.png",
        "linuxdo_url": "https://linux.do/u/alice/summary"
      },
      "model": "gpt-5.5",
      "accuracy": 0.9,
      "questionCount": 50,
      "attemptCount": 150,
      "avgTimeSeconds": 7.4,
      "createdAt": "2026-06-28T13:20:00.000Z",
      "status": "healthy"
    }
  ],
  "segments": [
    {
      "label": "macOS",
      "count": 612,
      "accuracy": 0.861
    }
  ],
  "hourlyBuckets": [
    {
      "hour": 2,
      "submissions": 44,
      "attempts": 6600,
      "accuracy": 0.792,
      "avgLatencySeconds": 11.4
    }
  ],
  "statistics": {
    "accuracy": {
      "mean": 0.842,
      "stdDev": 0.032,
      "ci95Low": 0.836,
      "ci95High": 0.848,
      "marginOfError": 0.006,
      "sampleSize": 192600
    },
    "latency": {
      "mean": 8.7,
      "median": 8.1,
      "p90": 12.4,
      "p95": 13.8,
      "stdDev": 2.1
    },
    "regression": {
      "baselineAccuracy": 0.835,
      "observedAccuracy": 0.842,
      "delta": 0.007,
      "zScore": 2.9,
      "pValue": 0.0037,
      "verdict": "improved"
    },
    "modelComparisons": [
      {
        "model": "gpt-5.5",
        "sampleSize": 76650,
        "accuracy": 0.882,
        "ci95Low": 0.88,
        "ci95High": 0.884,
        "marginOfError": 0.002,
        "posteriorMean": 0.882,
        "posteriorLow": 0.88,
        "posteriorHigh": 0.884,
        "deltaVsBest": 0,
        "verdict": "leader"
      }
    ],
    "pairwiseTests": [
      {
        "model": "gpt-5.5-mini",
        "comparedTo": "gpt-5.5",
        "delta": -0.064,
        "zScore": -12.8,
        "pValue": 0.0001,
        "adjustedPValue": 0.0003,
        "effectSize": -0.15,
        "verdict": "significant"
      }
    ],
    "power": {
      "baselineAccuracy": 0.835,
      "averageModelSampleSize": 48150,
      "minimumDetectableEffect": 0.006,
      "requiredSamples": [
        {
          "delta": 0.02,
          "perGroup": 6742
        }
      ]
    },
    "testCoverage": {
      "suites": [
        {
          "label": "API contract",
          "passed": 18,
          "total": 18,
          "status": "pass"
        }
      ],
      "totalAttempts": 6120,
      "passRate": 0.78,
      "regressionCount": 1,
      "watchCount": 4,
      "flakyQuestions": 2
    },
    "trendStability": {
      "submissionStdDev": 8.4,
      "accuracyStdDev": 0.021,
      "accuracyMean": 0.842,
      "upperControlLimit": 0.905,
      "lowerControlLimit": 0.779,
      "latestZScore": -0.42,
      "anomalies": []
    },
    "timeOfDay": {
      "omnibus": {
        "statistic": 168.4,
        "degreesOfFreedom": 23,
        "pValue": 0.0001,
        "verdict": "time_effect_detected"
      },
      "hourly": [
        {
          "hour": 2,
          "label": "02:00",
          "attempts": 6600,
          "submissions": 44,
          "accuracy": 0.792,
          "avgLatencySeconds": 11.4,
          "ci95Low": 0.782,
          "ci95High": 0.802,
          "posteriorLow": 0.782,
          "posteriorHigh": 0.802,
          "deltaVsDay": -0.05,
          "zScore": -9.2,
          "pValue": 0.0001,
          "adjustedPValue": 0.0001,
          "effectSize": -0.14,
          "riskScore": 4.06,
          "verdict": "degraded"
        }
      ],
      "segments": [
        {
          "label": "深夜",
          "startHour": 0,
          "endHour": 5,
          "attempts": 38600,
          "accuracy": 0.805,
          "avgLatencySeconds": 10.8,
          "deltaVsDay": -0.037,
          "zScore": -10.1,
          "pValue": 0.0001,
          "adjustedPValue": 0.0001,
          "verdict": "degraded"
        }
      ],
      "worstHours": [],
      "degradationWindows": [
        {
          "startHour": 2,
          "endHour": 5,
          "attempts": 26400,
          "riskScore": 14.4,
          "minDelta": -0.052,
          "label": "02:00-06:00"
        }
      ],
      "summary": {
        "worstHour": null,
        "worstSegment": null,
        "affectedAttempts": 26400,
        "overallAccuracy": 0.842
      }
    },
    "forecast": {
      "accuracy": {
        "slope": 0.0018,
        "intercept": 0.82,
        "rSquared": 0.61,
        "pValue": 0.004,
        "residualStdDev": 0.012,
        "verdict": "rising",
        "forecast": [
          {
            "step": 1,
            "value": 0.861,
            "low": 0.837,
            "high": 0.885
          }
        ]
      },
      "submissions": {
        "slope": 1.42,
        "intercept": 45,
        "rSquared": 0.73,
        "pValue": 0.001,
        "residualStdDev": 4.8,
        "verdict": "rising",
        "forecast": []
      }
    },
    "correlations": [
      {
        "metric": "小时耗时 vs 准确率",
        "x": "avgLatencySeconds",
        "y": "accuracy",
        "expectedDirection": "negative",
        "r": -0.63,
        "pValue": 0.0012,
        "sampleSize": 24,
        "strength": "moderate",
        "verdict": "significant"
      }
    ],
    "questionDiagnostics": [
      {
        "questionId": "judge-011",
        "title": "反例识别",
        "attempts": 1380,
        "accuracy": 0.691,
        "failureRate": 0.309,
        "ci95Low": 0.666,
        "ci95High": 0.715,
        "difficultyZ": 1.42,
        "priorityScore": 14.2,
        "verdict": "audit"
      }
    ],
    "modelRanking": [
      {
        "model": "gpt-5.5",
        "posteriorMean": 0.882,
        "posteriorStdDev": 0.0012,
        "probabilityBest": 0.97,
        "expectedLoss": 0,
        "verdict": "ship"
      }
    ],
    "robustness": {
      "recentOutliers": [
        {
          "id": "sub_008",
          "model": "o4-mini",
          "accuracy": 0.742,
          "latency": 13.2,
          "accuracyRobustZ": -1.8,
          "latencyRobustZ": 1.9
        }
      ],
      "questionOutliers": [
        {
          "questionId": "judge-011",
          "title": "反例识别",
          "failureRate": 0.309,
          "failureRobustZ": 1.7
        }
      ],
      "baselines": {
        "submissionAccuracyMedian": 0.846,
        "submissionLatencyMedian": 8.4,
        "questionFailureMedian": 0.211
      }
    },
    "distributionShape": {
      "dailyAccuracy": {
        "min": 0.801,
        "q1": 0.829,
        "median": 0.842,
        "q3": 0.858,
        "max": 0.884,
        "iqr": 0.029,
        "mean": 0.843,
        "stdDev": 0.018,
        "coefficientOfVariation": 0.021,
        "skewness": -0.12,
        "excessKurtosis": -0.42,
        "tailRisk": 0.033
      },
      "dailySubmissions": {
        "min": 36,
        "q1": 47,
        "median": 54,
        "q3": 63,
        "max": 76,
        "iqr": 16,
        "mean": 55,
        "stdDev": 10,
        "coefficientOfVariation": 0.182,
        "skewness": 0.21,
        "excessKurtosis": -0.67,
        "tailRisk": 0
      },
      "recentLatency": {
        "min": 6.7,
        "q1": 7.5,
        "median": 8.4,
        "q3": 10.2,
        "max": 13.2,
        "iqr": 2.7,
        "mean": 8.8,
        "stdDev": 1.9,
        "coefficientOfVariation": 0.216,
        "skewness": 0.64,
        "excessKurtosis": 0.12,
        "tailRisk": 0.125
      },
      "questionFailure": {
        "min": 0.079,
        "q1": 0.142,
        "median": 0.211,
        "q3": 0.258,
        "max": 0.309,
        "iqr": 0.116,
        "mean": 0.207,
        "stdDev": 0.078,
        "coefficientOfVariation": 0.377,
        "skewness": -0.18,
        "excessKurtosis": -1.1,
        "tailRisk": 0
      },
      "hourlyAccuracy": {
        "min": 0.792,
        "q1": 0.824,
        "median": 0.842,
        "q3": 0.857,
        "max": 0.872,
        "iqr": 0.033,
        "mean": 0.839,
        "stdDev": 0.022,
        "coefficientOfVariation": 0.026,
        "skewness": -0.52,
        "excessKurtosis": -0.31,
        "tailRisk": 0.083
      }
    },
    "drift": {
      "window": {
        "priorDays": 15,
        "recentDays": 15,
        "priorAccuracy": 0.846,
        "recentAccuracy": 0.838,
        "delta": -0.008,
        "zScore": -4.1,
        "pValue": 0.0001,
        "verdict": "negative_drift"
      },
      "volume": {
        "priorMean": 52.2,
        "recentMean": 61.7,
        "delta": 9.5,
        "tScore": 3.2,
        "degreesOfFreedom": 27.4,
        "pValue": 0.0034,
        "verdict": "changed"
      },
      "ewma": {
        "lambda": 0.32,
        "latest": 0.839,
        "deltaVsMean": -0.01,
        "min": 0.821,
        "max": 0.862,
        "verdict": "stable",
        "series": [
          {
            "date": "2026-06-01",
            "value": 0.842
          }
        ]
      },
      "cusum": {
        "latest": -0.024,
        "min": -0.031,
        "max": 0.018,
        "signalScore": 4.1,
        "verdict": "alert",
        "series": []
      }
    },
    "riskBudget": {
      "targetAccuracy": 0.835,
      "failureRate": 0.158,
      "failures": 30431,
      "allowedFailures": 31779,
      "excessFailures": 0,
      "budgetRemaining": 0.042,
      "burnRate": 0.96,
      "degradedAttemptShare": 0.137,
      "auditQuestions": 2,
      "outlierLoad": 3,
      "anomalyDays": 1,
      "verdict": "watch"
    },
    "efficiencyFrontier": [
      {
        "model": "gpt-5.5",
        "accuracy": 0.882,
        "avgTps": 39.4,
        "avgTimeSeconds": 7.9,
        "utilityScore": 0.872,
        "dominatedBy": [],
        "onFrontier": true,
        "verdict": "frontier"
      }
    ]
  }
}
```

`recentSubmissions[].user` may be a legacy string or a display object. For anonymous uploads, return `{ "anonymous": true, "display_name": "匿名", "username": "", "avatar_url": "", "linuxdo_url": "" }`. Anonymous mode hides identity only; the submission's benchmark data remains visible and statistically included.

## Statistical Fields

- `accuracy.ci95Low` and `accuracy.ci95High` use a Wilson score interval for binomial pass/fail attempts.
- `latency.p90` and `latency.p95` are percentile values from recent submissions.
- `regression` compares the current window against a baseline using a two-proportion z-test.
- `modelComparisons` gives per-model Wilson intervals, approximate beta-posterior credible ranges, and whether the interval overlaps with the best model.
- `pairwiseTests` compares each model with the best observed model using a two-proportion z-test, Holm-adjusted p-values, and Cohen's h effect size.
- `power` estimates the minimum detectable effect and per-group sample requirements for common accuracy deltas at 80% power.
- `testCoverage` summarizes automated checks, regression samples, and visual smoke checks shown in the dashboard.
- `trendStability` provides control-chart style limits and recent z-scores for trend monitoring.
- `hourlyBuckets` groups attempts by local hour of day. Production should derive it from `benchmark_submissions.created_at` or the attempt timestamp if stored.
- `timeOfDay.omnibus` uses a chi-square test across hourly pass/fail buckets to detect whether time of day matters overall.
- `forecast` runs an ordinary least squares trend for accuracy and submission volume, returning slope tests, R-squared, residual standard deviation, and seven forward points.
- `correlations` runs Pearson correlation tests for latency, volume, and question difficulty relationships. `pValue` is derived from a Student t distribution.
- `questionDiagnostics` ranks questions by Wilson interval, failure-rate z-score, sample size, and time penalty so review work starts with the highest risk items.
- `modelRanking` estimates beta posterior means and an approximate probability of being the best model; use it for release candidate ranking, not as a sole promotion gate.
- `robustness` uses median and median absolute deviation baselines to surface recent submission and question outliers.
- `distributionShape` summarizes quartiles, IQR, coefficient of variation, skewness, excess kurtosis, and Tukey-fence tail risk for core distributions.
- `drift.window` compares the first and second half of the selected range with a two-proportion z-test; `drift.volume` uses Welch's t-test for submission volume changes.
- `drift.ewma` and `drift.cusum` expose smoothed accuracy and cumulative deviation series for monitoring drift shape, not only endpoint deltas.
- `riskBudget` converts the target accuracy into allowed failures, burn rate, degraded-attempt share, and review load so operators can see whether the window is over budget.
- `efficiencyFrontier` computes a Pareto-style model frontier across accuracy, TPS, and latency plus a weighted utility score for release tradeoffs.
- `timeOfDay.hourly` compares each hour against the rest of the day with two-proportion z-tests, Holm-adjusted p-values, Wilson intervals, beta-posterior ranges, and Cohen's h effect size.
- `timeOfDay.degradationWindows` merges adjacent significantly degraded hours into human-readable risk windows such as `02:00-06:00`.

The Worker implementation uses:

- D1 aggregate queries for summary, trends, model breakdowns, question quality, recent submissions, channel segments, and hourly buckets.
- Worker-side derived statistics for intervals, rankings, pairwise comparisons, drift, risk budgeting, and distribution summaries.

## Frontend Safeguards

- Browser API responses are parsed through `DashboardOverviewSchema` in `dashboard/src/schema.js` before rendering.
- Invalid or incomplete API payloads fail fast instead of producing misleading charts.
- The dashboard can export a compact JSON statistical snapshot and hourly CSV from the current filter state.
- `range` and `model` filters are mirrored into the URL so screenshots and shared links preserve context.

Recommended Worker behavior:

- Require an authenticated web session for non-public metrics.
- Use D1 aggregate queries and return only aggregate/user-safe fields.
- Cache short windows for 30 to 60 seconds if traffic grows.
- Keep this endpoint read-only and avoid exposing raw prompts, extracted answers, tokens, or OAuth/session data.
