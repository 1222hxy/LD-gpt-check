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
  ]
}
```

Recommended Worker behavior:

- Require an authenticated web session for non-public metrics.
- Use D1 aggregate queries and return only aggregate/user-safe fields.
- Cache short windows for 30 to 60 seconds if traffic grows.
- Keep this endpoint read-only and avoid exposing raw prompts, extracted answers, tokens, or OAuth/session data.
