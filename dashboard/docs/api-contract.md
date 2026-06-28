# Dashboard API Contract

The dashboard frontend calls one read-only endpoint:

```text
GET /api/dashboard/overview?range=30d&model=all
```

For local development, `dashboard/vite.config.js` serves this endpoint with mock data. In production, the same response shape can be implemented in the Cloudflare Worker from D1 tables:

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
      "user": "alice",
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
    }
  }
}
```

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
- `timeOfDay.hourly` compares each hour against the rest of the day with two-proportion z-tests, Holm-adjusted p-values, Wilson intervals, beta-posterior ranges, and Cohen's h effect size.
- `timeOfDay.degradationWindows` merges adjacent significantly degraded hours into human-readable risk windows such as `02:00-06:00`.

The local mock implementation uses:

- `jstat` for normal and chi-square distribution CDFs.
- `simple-statistics` for descriptive statistics and quantiles.

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
