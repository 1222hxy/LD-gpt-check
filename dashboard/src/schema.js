import { z } from "zod";

const StatusSchema = z.enum(["healthy", "watch", "regression"]);
const VerdictSchema = z.string();

const SummarySchema = z.object({
  submissions: z.number(),
  activeUsers: z.number(),
  averageAccuracy: z.number(),
  averageLatencySeconds: z.number(),
  averageTps: z.number(),
  tokenTotal: z.number(),
});

const TrendPointSchema = z.object({
  date: z.string(),
  submissions: z.number(),
  accuracy: z.number(),
  avgTps: z.number(),
  tokens: z.number(),
});

const ModelBreakdownSchema = z.object({
  model: z.string(),
  submissions: z.number(),
  accuracy: z.number(),
  avgTps: z.number(),
  avgTimeSeconds: z.number(),
});

const QuestionQualitySchema = z.object({
  questionId: z.string(),
  title: z.string(),
  accuracy: z.number(),
  attempts: z.number(),
  avgTimeSeconds: z.number(),
  failureRate: z.number(),
});

const RecentSubmissionSchema = z.object({
  id: z.string(),
  user: z.string(),
  model: z.string(),
  accuracy: z.number(),
  questionCount: z.number(),
  attemptCount: z.number(),
  avgTimeSeconds: z.number(),
  createdAt: z.string(),
  status: StatusSchema,
});

const SegmentSchema = z.object({
  label: z.string(),
  count: z.number(),
  accuracy: z.number(),
});

const HourlyBucketSchema = z.object({
  hour: z.number().int().min(0).max(23),
  submissions: z.number(),
  attempts: z.number(),
  accuracy: z.number(),
  avgLatencySeconds: z.number(),
});

const StatisticalAccuracySchema = z.object({
  mean: z.number(),
  stdDev: z.number(),
  ci95Low: z.number(),
  ci95High: z.number(),
  marginOfError: z.number(),
  sampleSize: z.number(),
});

const LatencySchema = z.object({
  mean: z.number(),
  median: z.number(),
  p90: z.number(),
  p95: z.number(),
  stdDev: z.number(),
});

const RegressionSchema = z.object({
  baselineAccuracy: z.number(),
  observedAccuracy: z.number(),
  delta: z.number(),
  zScore: z.number(),
  pValue: z.number(),
  verdict: VerdictSchema,
});

const PowerSchema = z.object({
  baselineAccuracy: z.number(),
  averageModelSampleSize: z.number(),
  minimumDetectableEffect: z.number(),
  requiredSamples: z.array(z.object({ delta: z.number(), perGroup: z.number() })),
});

const ModelComparisonSchema = z.object({
  model: z.string(),
  sampleSize: z.number(),
  accuracy: z.number(),
  ci95Low: z.number(),
  ci95High: z.number(),
  marginOfError: z.number(),
  posteriorMean: z.number(),
  posteriorLow: z.number(),
  posteriorHigh: z.number(),
  deltaVsBest: z.number(),
  verdict: VerdictSchema,
});

const PairwiseTestSchema = z.object({
  model: z.string(),
  comparedTo: z.string(),
  delta: z.number(),
  zScore: z.number(),
  pValue: z.number(),
  adjustedPValue: z.number(),
  effectSize: z.number(),
  verdict: VerdictSchema,
});

const TestCoverageSchema = z.object({
  suites: z.array(
    z.object({
      label: z.string(),
      passed: z.number(),
      total: z.number(),
      status: z.string(),
    }),
  ),
  totalAttempts: z.number(),
  passRate: z.number(),
  regressionCount: z.number(),
  watchCount: z.number(),
  flakyQuestions: z.number(),
});

const TrendStabilitySchema = z.object({
  submissionStdDev: z.number(),
  accuracyStdDev: z.number(),
  accuracyMean: z.number(),
  upperControlLimit: z.number(),
  lowerControlLimit: z.number(),
  latestZScore: z.number(),
  anomalies: z.array(z.object({ date: z.string(), accuracy: z.number(), zScore: z.number() })),
});

const HourlyAnalysisSchema = z.object({
  hour: z.number(),
  label: z.string(),
  attempts: z.number(),
  submissions: z.number(),
  accuracy: z.number(),
  avgLatencySeconds: z.number(),
  ci95Low: z.number(),
  ci95High: z.number(),
  posteriorLow: z.number(),
  posteriorHigh: z.number(),
  deltaVsDay: z.number(),
  zScore: z.number(),
  pValue: z.number(),
  adjustedPValue: z.number(),
  effectSize: z.number(),
  riskScore: z.number(),
  verdict: VerdictSchema,
});

const TimeSegmentAnalysisSchema = z.object({
  label: z.string(),
  startHour: z.number(),
  endHour: z.number(),
  attempts: z.number(),
  accuracy: z.number(),
  avgLatencySeconds: z.number(),
  deltaVsDay: z.number(),
  zScore: z.number(),
  pValue: z.number(),
  adjustedPValue: z.number(),
  verdict: VerdictSchema,
});

const TimeOfDaySchema = z.object({
  omnibus: z.object({
    statistic: z.number(),
    degreesOfFreedom: z.number(),
    pValue: z.number(),
    verdict: VerdictSchema,
  }),
  hourly: z.array(HourlyAnalysisSchema),
  segments: z.array(TimeSegmentAnalysisSchema),
  worstHours: z.array(HourlyAnalysisSchema),
  degradationWindows: z.array(
    z.object({
      startHour: z.number().optional(),
      endHour: z.number().optional(),
      attempts: z.number(),
      riskScore: z.number(),
      minDelta: z.number(),
      label: z.string(),
    }),
  ),
  summary: z.object({
    worstHour: HourlyAnalysisSchema.nullable(),
    worstSegment: TimeSegmentAnalysisSchema.nullable(),
    affectedAttempts: z.number(),
    overallAccuracy: z.number(),
  }),
});

const StatisticsSchema = z.object({
  accuracy: StatisticalAccuracySchema,
  latency: LatencySchema,
  regression: RegressionSchema,
  power: PowerSchema,
  modelComparisons: z.array(ModelComparisonSchema),
  pairwiseTests: z.array(PairwiseTestSchema),
  testCoverage: TestCoverageSchema,
  trendStability: TrendStabilitySchema,
  timeOfDay: TimeOfDaySchema,
});

export const DashboardOverviewSchema = z.object({
  updatedAt: z.string(),
  filters: z.object({
    range: z.string(),
    model: z.string(),
    models: z.array(z.string()),
  }),
  summary: SummarySchema,
  trend: z.array(TrendPointSchema),
  modelBreakdown: z.array(ModelBreakdownSchema),
  questionQuality: z.array(QuestionQualitySchema),
  recentSubmissions: z.array(RecentSubmissionSchema),
  segments: z.array(SegmentSchema),
  hourlyBuckets: z.array(HourlyBucketSchema),
  statistics: StatisticsSchema,
});
