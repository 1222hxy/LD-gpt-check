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
  user: z.union([
    z.string(),
    z.object({
      anonymous: z.boolean().optional(),
      display_name: z.string().optional(),
      username: z.string().optional(),
      avatar_url: z.string().optional(),
      linuxdo_url: z.string().optional(),
    }),
  ]),
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
  sampleSize: z.number(),
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
  verdict: VerdictSchema,
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
  verdict: VerdictSchema,
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

const ForecastSeriesSchema = z.object({
  slope: z.number(),
  intercept: z.number(),
  rSquared: z.number(),
  pValue: z.number(),
  residualStdDev: z.number(),
  sampleSize: z.number(),
  verdict: VerdictSchema,
  forecast: z.array(z.object({ step: z.number(), value: z.number(), low: z.number(), high: z.number() })),
});

const CorrelationSchema = z.object({
  metric: z.string(),
  x: z.string(),
  y: z.string(),
  expectedDirection: z.string(),
  r: z.number(),
  pValue: z.number(),
  sampleSize: z.number(),
  strength: z.string(),
  verdict: VerdictSchema,
});

const QuestionDiagnosticSchema = z.object({
  questionId: z.string(),
  title: z.string(),
  attempts: z.number(),
  accuracy: z.number(),
  failureRate: z.number(),
  ci95Low: z.number(),
  ci95High: z.number(),
  difficultyZ: z.number(),
  priorityScore: z.number(),
  verdict: VerdictSchema,
});

const ModelRankingSchema = z.object({
  model: z.string(),
  posteriorMean: z.number(),
  posteriorStdDev: z.number(),
  probabilityBest: z.number(),
  expectedLoss: z.number(),
  verdict: VerdictSchema,
});

const RobustnessSchema = z.object({
  recentOutliers: z.array(
    z.object({
      id: z.string(),
      model: z.string(),
      accuracy: z.number(),
      latency: z.number(),
      accuracyRobustZ: z.number(),
      latencyRobustZ: z.number(),
    }),
  ),
  questionOutliers: z.array(
    z.object({
      questionId: z.string(),
      title: z.string(),
      failureRate: z.number(),
      failureRobustZ: z.number(),
    }),
  ),
  baselines: z.object({
    submissionAccuracyMedian: z.number(),
    submissionLatencyMedian: z.number(),
    questionFailureMedian: z.number(),
    submissionSampleSize: z.number(),
    questionSampleSize: z.number(),
  }),
});

const DistributionSummarySchema = z.object({
  min: z.number(),
  q1: z.number(),
  median: z.number(),
  q3: z.number(),
  max: z.number(),
  iqr: z.number(),
  mean: z.number(),
  stdDev: z.number(),
  coefficientOfVariation: z.number(),
  skewness: z.number(),
  excessKurtosis: z.number(),
  tailRisk: z.number(),
  sampleSize: z.number(),
});

const CoverageSchema = z.object({
  submissions: z.number(),
  attempts: z.number(),
  trendDays: z.number(),
  models: z.number(),
  comparedModels: z.number(),
  questions: z.number(),
  activeHours: z.number(),
  accuracySamples: z.number(),
  latencySamples: z.number(),
  hasSubmissions: z.boolean(),
  hasTrend: z.boolean(),
  hasForecast: z.boolean(),
  hasModelComparison: z.boolean(),
  hasQuestionDiagnostics: z.boolean(),
  hasTimeOfDay: z.boolean(),
  hasDistribution: z.boolean(),
});

const DistributionShapeSchema = z.object({
  dailyAccuracy: DistributionSummarySchema,
  dailySubmissions: DistributionSummarySchema,
  recentLatency: DistributionSummarySchema,
  questionFailure: DistributionSummarySchema,
  hourlyAccuracy: DistributionSummarySchema,
});

const DriftSchema = z.object({
  window: z.object({
    priorDays: z.number(),
    recentDays: z.number(),
    priorAccuracy: z.number(),
    recentAccuracy: z.number(),
    delta: z.number(),
    zScore: z.number(),
    pValue: z.number(),
    verdict: VerdictSchema,
  }),
  volume: z.object({
    priorMean: z.number(),
    recentMean: z.number(),
    delta: z.number(),
    tScore: z.number(),
    degreesOfFreedom: z.number(),
    pValue: z.number(),
    verdict: VerdictSchema,
  }),
  ewma: z.object({
    lambda: z.number(),
    latest: z.number(),
    deltaVsMean: z.number(),
    min: z.number(),
    max: z.number(),
    verdict: VerdictSchema,
    series: z.array(z.object({ date: z.string(), value: z.number() })),
  }),
  cusum: z.object({
    latest: z.number(),
    min: z.number(),
    max: z.number(),
    signalScore: z.number(),
    verdict: VerdictSchema,
    series: z.array(z.object({ date: z.string(), value: z.number() })),
  }),
});

const RiskBudgetSchema = z.object({
  targetAccuracy: z.number(),
  failureRate: z.number(),
  failures: z.number(),
  allowedFailures: z.number(),
  excessFailures: z.number(),
  budgetRemaining: z.number(),
  burnRate: z.number(),
  degradedAttemptShare: z.number(),
  auditQuestions: z.number(),
  outlierLoad: z.number(),
  anomalyDays: z.number(),
  verdict: VerdictSchema,
});

const EfficiencyFrontierSchema = z.object({
  model: z.string(),
  accuracy: z.number(),
  avgTps: z.number(),
  avgTimeSeconds: z.number(),
  utilityScore: z.number(),
  dominatedBy: z.array(z.string()),
  onFrontier: z.boolean(),
  verdict: VerdictSchema,
});

const StatisticsSchema = z.object({
  coverage: CoverageSchema,
  accuracy: StatisticalAccuracySchema,
  latency: LatencySchema,
  regression: RegressionSchema,
  power: PowerSchema,
  modelComparisons: z.array(ModelComparisonSchema),
  pairwiseTests: z.array(PairwiseTestSchema),
  testCoverage: TestCoverageSchema,
  trendStability: TrendStabilitySchema,
  timeOfDay: TimeOfDaySchema,
  forecast: z.object({
    accuracy: ForecastSeriesSchema,
    submissions: ForecastSeriesSchema,
  }),
  correlations: z.array(CorrelationSchema),
  questionDiagnostics: z.array(QuestionDiagnosticSchema),
  modelRanking: z.array(ModelRankingSchema),
  robustness: RobustnessSchema,
  distributionShape: DistributionShapeSchema,
  drift: DriftSchema,
  riskBudget: RiskBudgetSchema,
  efficiencyFrontier: z.array(EfficiencyFrontierSchema),
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
