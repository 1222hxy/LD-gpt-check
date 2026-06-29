import { buildStatistics } from "../statistics.js";

const MODELS = ["gpt-5.5", "gpt-5.5-mini", "o4-mini", "deepseek-r1"];
const MODEL_PROFILES = {
  "gpt-5.5": { accuracy: 0.885, tps: 39.4, latency: 7.8, share: 0.36 },
  "gpt-5.5-mini": { accuracy: 0.824, tps: 54.1, latency: 5.9, share: 0.28 },
  "o4-mini": { accuracy: 0.803, tps: 43.8, latency: 8.9, share: 0.22 },
  "deepseek-r1": { accuracy: 0.847, tps: 31.6, latency: 11.2, share: 0.14 },
};

const QUESTIONS = [
  ["logic-021", "多约束逻辑排序"],
  ["math-014", "比例与方程推理"],
  ["code-008", "边界条件修复"],
  ["text-017", "长文本事实抽取"],
  ["tool-006", "工具调用路径选择"],
  ["judge-011", "反例识别"],
];

const USERS = ["chen", "luna", "mika", "tang", "ops", "yu", "river", "lin"];
const SEGMENTS = ["macOS", "Linux", "Windows", "CI Runner"];
const PROVIDERS = [
  { codexChannel: "official", codexBridgeName: "", codexProviderBaseURL: "https://api.openai.com/v1", codexProviderHost: "api.openai.com", channelLabel: "官方 API (api.openai.com)" },
  { codexChannel: "bridge", codexBridgeName: "Krill AI", codexProviderBaseURL: "https://api.krill-ai.com/codex/v1", codexProviderHost: "api.krill-ai.com", channelLabel: "Krill AI (api.krill-ai.com)" },
  { codexChannel: "unknown_bridge", codexBridgeName: "", codexProviderBaseURL: "https://relay.example.com/v1", codexProviderHost: "relay.example.com", channelLabel: "未识别中转站 (relay.example.com)" },
];
const DAY_MS = 24 * 60 * 60 * 1000;

export function buildDashboardPayload({ range = "30d", model = "all" } = {}) {
  const days = range === "7d" ? 7 : range === "90d" ? 90 : 30;
  const selectedModels = model === "all" || !MODELS.includes(model) ? MODELS : [model];
  const trend = buildTrend(days, selectedModels);
  const modelBreakdown = buildModelBreakdown(selectedModels, days);
  const questionQuality = buildQuestionQuality(days, selectedModels.length);
  const recentSubmissions = buildRecentSubmissions(selectedModels);
  const userBridgeUsage = buildUserBridgeUsage(recentSubmissions);
  const segments = buildSegments(selectedModels.length);
  const summary = summarize(trend, modelBreakdown, segments);
  const hourlyBuckets = buildHourlyBuckets(days, selectedModels);
  const statistics = buildStatistics({ trend, modelBreakdown, questionQuality, recentSubmissions, hourlyBuckets });

  return {
    updatedAt: new Date().toISOString(),
    filters: {
      range,
      model,
      models: MODELS,
    },
    summary,
    trend,
    modelBreakdown,
    questionQuality,
    recentSubmissions,
    userBridgeUsage,
    segments,
    hourlyBuckets,
    statistics,
  };
}

function buildTrend(days, models) {
  const now = Date.now();
  return Array.from({ length: days }, (_, index) => {
    const date = new Date(now - (days - index - 1) * DAY_MS);
    const wave = Math.sin(index / 3.2) * 0.035;
    const modelFactor = average(models.map((item) => MODEL_PROFILES[item].accuracy));
    const submissions = Math.round((28 + index * 0.8 + models.length * 13) * (1 + Math.sin(index / 4.7) * 0.14));
    const accuracy = clamp(modelFactor + wave + (index % 6) * 0.002, 0.68, 0.94);
    const avgTps = average(models.map((item) => MODEL_PROFILES[item].tps)) + Math.cos(index / 3) * 2.4;
    return {
      date: date.toISOString().slice(0, 10),
      submissions,
      accuracy: round(accuracy, 3),
      avgTps: round(avgTps, 1),
      tokens: Math.round(submissions * (5800 + (index % 5) * 460)),
    };
  });
}

function buildModelBreakdown(models, days) {
  return models.map((model, index) => {
    const profile = MODEL_PROFILES[model];
    const daysFactor = Math.sqrt(days / 30);
    return {
      model,
      submissions: Math.round(420 * profile.share * daysFactor + 70 + index * 23),
      accuracy: round(profile.accuracy + (index % 2 ? -0.006 : 0.004), 3),
      avgTps: round(profile.tps + index * 0.8, 1),
      avgTimeSeconds: round(profile.latency + index * 0.4, 1),
    };
  });
}

function buildQuestionQuality(days, modelCount) {
  return QUESTIONS.map(([questionId, title], index) => {
    const attempts = Math.round(days * modelCount * (5.5 + index * 1.2));
    const accuracy = clamp(0.91 - index * 0.045 + Math.sin(days + index) * 0.012, 0.58, 0.93);
    return {
      questionId,
      title,
      accuracy: round(accuracy, 3),
      attempts,
      avgTimeSeconds: round(6.4 + index * 1.05 + modelCount * 0.45, 1),
      failureRate: round(1 - accuracy, 3),
    };
  }).sort((a, b) => a.accuracy - b.accuracy);
}

function buildRecentSubmissions(models) {
  const now = Date.now();
  return Array.from({ length: 8 }, (_, index) => {
    const currentModel = models[index % models.length];
    const profile = MODEL_PROFILES[currentModel];
    const accuracy = clamp(profile.accuracy + Math.sin(index * 1.7) * 0.04, 0.65, 0.96);
    const username = USERS[index % USERS.length];
    const anonymous = index % 5 === 3;
    const provider = PROVIDERS[index % PROVIDERS.length];
    return {
      id: `sub_${String(index + 1).padStart(3, "0")}`,
      user: anonymous
        ? { anonymous: true, display_name: "匿名", username: "", avatar_url: "", linuxdo_url: "" }
        : {
            anonymous: false,
            display_name: username,
            username,
            avatar_url: `https://cdn.ldstatic.com/user_avatar/linux.do/${username}/288/170339_2.png`,
            linuxdo_url: `https://linux.do/u/${username}/summary`,
          },
      model: currentModel,
      accuracy: round(accuracy, 3),
      questionCount: 50,
      attemptCount: 150,
      avgTimeSeconds: round(profile.latency + (index % 4) * 0.8, 1),
      createdAt: new Date(now - index * 42 * 60 * 1000).toISOString(),
      status: accuracy > 0.86 ? "healthy" : accuracy > 0.78 ? "watch" : "regression",
      ...provider,
    };
  });
}

function buildUserBridgeUsage(recentSubmissions) {
  const byUser = new Map();
  for (const submission of recentSubmissions) {
    const displayName = submission.user?.anonymous
      ? `anonymous:${submission.channelLabel || submission.codexProviderHost || "unknown"}`
      : submission.user?.username || submission.user?.display_name || "user";
    const current = byUser.get(displayName) || {
      user: submission.user,
      submissions: 0,
      accuracyWeightedSum: 0,
      lastSubmissionAt: "",
      channels: new Map(),
    };
    current.submissions += 1;
    current.accuracyWeightedSum += submission.accuracy;
    if (submission.createdAt > current.lastSubmissionAt) current.lastSubmissionAt = submission.createdAt;
    const channelKey = submission.channelLabel || submission.codexProviderHost || "unknown";
    const channel = current.channels.get(channelKey) || {
      codexChannel: submission.codexChannel,
      codexBridgeName: submission.codexBridgeName,
      codexProviderBaseURL: submission.codexProviderBaseURL,
      codexProviderHost: submission.codexProviderHost,
      channelLabel: submission.channelLabel,
      submissions: 0,
      accuracyWeightedSum: 0,
    };
    channel.submissions += 1;
    channel.accuracyWeightedSum += submission.accuracy;
    current.channels.set(channelKey, channel);
    byUser.set(displayName, current);
  }
  return [...byUser.values()].map((item) => {
    const channels = [...item.channels.values()]
      .map((channel) => {
        const { accuracyWeightedSum, ...rest } = channel;
        return {
          ...rest,
          accuracy: round(accuracyWeightedSum / channel.submissions, 3),
        };
      })
      .sort((a, b) => b.submissions - a.submissions)
      .slice(0, 3);
    const primary = channels[0] || {};
    return {
      user: item.user,
      submissions: item.submissions,
      accuracy: round(item.accuracyWeightedSum / item.submissions, 3),
      lastSubmissionAt: item.lastSubmissionAt,
      codexChannel: primary.codexChannel,
      codexBridgeName: primary.codexBridgeName,
      codexProviderBaseURL: primary.codexProviderBaseURL,
      codexProviderHost: primary.codexProviderHost,
      channelLabel: primary.channelLabel,
      channelCount: channels.length,
      channels,
    };
  });
}

function buildSegments(modelCount) {
  return SEGMENTS.map((label, index) => ({
    label,
    count: Math.round(86 + modelCount * 41 + index * 34),
    accuracy: round(0.795 + index * 0.018 + modelCount * 0.006, 3),
  }));
}

function buildHourlyBuckets(days, models) {
  const modelAccuracy = average(models.map((item) => MODEL_PROFILES[item].accuracy));
  const modelLatency = average(models.map((item) => MODEL_PROFILES[item].latency));
  const scale = Math.sqrt(days / 30);
  return Array.from({ length: 24 }, (_, hour) => {
    const volumeCurve = 1 + 0.24 * Math.sin((hour - 8) / 24 * Math.PI * 2) + 0.18 * Math.sin((hour - 15) / 24 * Math.PI * 2);
    const circadianPenalty = circadianAccuracyPenalty(hour);
    const latencyPenalty = circadianLatencyPenalty(hour);
    const submissions = Math.max(8, Math.round((32 + models.length * 7) * scale * volumeCurve));
    const attempts = submissions * 150;
    const accuracy = clamp(modelAccuracy - circadianPenalty + Math.cos(hour / 2.7) * 0.006, 0.68, 0.93);

    return {
      hour,
      submissions,
      attempts,
      accuracy: round(accuracy, 3),
      avgLatencySeconds: round(modelLatency + latencyPenalty + Math.sin(hour / 3) * 0.4, 1),
    };
  });
}

function circadianAccuracyPenalty(hour) {
  if (hour >= 2 && hour <= 5) return 0.045;
  if (hour >= 14 && hour <= 16) return 0.026;
  if (hour >= 22 || hour <= 1) return 0.015;
  return 0;
}

function circadianLatencyPenalty(hour) {
  if (hour >= 2 && hour <= 5) return 2.2;
  if (hour >= 14 && hour <= 16) return 1.3;
  if (hour >= 22 || hour <= 1) return 0.8;
  return 0;
}

function summarize(trend, modelBreakdown, segments) {
  const submissions = sum(trend.map((item) => item.submissions));
  const tokenTotal = sum(trend.map((item) => item.tokens));
  return {
    submissions,
    activeUsers: Math.round(sum(segments.map((item) => item.count)) * 0.42),
    averageAccuracy: round(weightedAverage(modelBreakdown, "accuracy", "submissions"), 3),
    averageLatencySeconds: round(weightedAverage(modelBreakdown, "avgTimeSeconds", "submissions"), 1),
    averageTps: round(weightedAverage(modelBreakdown, "avgTps", "submissions"), 1),
    tokenTotal,
  };
}

function weightedAverage(items, valueKey, weightKey) {
  const totalWeight = sum(items.map((item) => item[weightKey]));
  return sum(items.map((item) => item[valueKey] * item[weightKey])) / totalWeight;
}

function average(values) {
  return sum(values) / values.length;
}

function sum(values) {
  return values.reduce((total, value) => total + value, 0);
}

function round(value, decimals) {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
