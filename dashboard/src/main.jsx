import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import "@fontsource/noto-sans-sc/chinese-simplified-400.css";
import "@fontsource/noto-sans-sc/chinese-simplified-600.css";
import {
  Activity,
  BarChart3,
  Clock3,
  Download,
  Gauge,
  RefreshCcw,
  Search,
  ShieldCheck,
  Sparkles,
  TrendingUp,
  Users,
} from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { fetchDashboardOverview } from "./api.js";
import { downloadDashboardExport } from "./export.js";
import { DEFAULT_FILTERS, parseFilters, writeFiltersToUrl } from "./filters.js";
import "./styles.css";

const queryClient = new QueryClient();
const RANGE_OPTIONS = [
  { label: "7 天", value: "7d" },
  { label: "30 天", value: "30d" },
  { label: "90 天", value: "90d" },
];
const SEGMENT_COLORS = ["#0f766e", "#b45309", "#2563eb", "#be123c"];

function App() {
  const [filters, setFilters] = useState(() =>
    typeof window === "undefined" ? DEFAULT_FILTERS : parseFilters(window.location.search),
  );
  const dashboard = useQuery({
    queryKey: ["dashboard-overview", filters],
    queryFn: () => fetchDashboardOverview(filters),
    refetchInterval: 60_000,
  });

  useEffect(() => {
    writeFiltersToUrl(filters);
  }, [filters]);

  useEffect(() => {
    if (!dashboard.data) return;
    const normalizedFilters = parseFilters(new URLSearchParams(filters).toString(), dashboard.data.filters.models);
    if (normalizedFilters.model !== filters.model || normalizedFilters.range !== filters.range) {
      setFilters(normalizedFilters);
    }
  }, [dashboard.data, filters]);

  if (dashboard.isLoading) return <LoadingState />;
  if (dashboard.isError) return <ErrorState onRetry={dashboard.refetch} />;

  const data = dashboard.data;

  return (
    <main className="min-h-screen bg-[#f7f7f2] text-ink">
      <TopBar
        updatedAt={data.updatedAt}
        onRefresh={dashboard.refetch}
        isRefreshing={dashboard.isFetching}
        onExport={() => downloadDashboardExport(data, filters)}
      />
      <section className="mx-auto grid w-full max-w-[1440px] gap-5 px-4 py-5 sm:px-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:px-8">
        <Sidebar filters={filters} models={data.filters.models} onChange={setFilters} />
        <div className="grid min-w-0 gap-5">
          <SummaryGrid summary={data.summary} />
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <StatisticsPanel statistics={data.statistics} />
            <TestPanel coverage={data.statistics.testCoverage} />
          </div>
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <PairwisePanel tests={data.statistics.pairwiseTests} />
            <PowerPanel statistics={data.statistics} />
          </div>
          <TimeOfDayPanel analysis={data.statistics.timeOfDay} />
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1.1fr)_minmax(360px,0.9fr)]">
            <ForecastPanel forecast={data.statistics.forecast} />
            <CorrelationPanel correlations={data.statistics.correlations} />
          </div>
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.9fr)]">
            <ModelRankingPanel ranking={data.statistics.modelRanking} />
            <RobustnessPanel robustness={data.statistics.robustness} />
          </div>
          <QuestionDiagnosticsPanel diagnostics={data.statistics.questionDiagnostics} />
          <div className="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
            <RiskBudgetPanel budget={data.statistics.riskBudget} />
            <DriftPanel drift={data.statistics.drift} />
          </div>
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.9fr)]">
            <DistributionPanel shape={data.statistics.distributionShape} />
            <EfficiencyFrontierPanel frontier={data.statistics.efficiencyFrontier} />
          </div>
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(360px,0.9fr)]">
            <TrendPanel trend={data.trend} />
            <ModelPanel models={data.modelBreakdown} />
          </div>
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <QualityPanel questions={data.questionQuality} />
            <SegmentPanel segments={data.segments} />
          </div>
          <RecentPanel submissions={data.recentSubmissions} />
        </div>
      </section>
    </main>
  );
}

function TopBar({ updatedAt, onRefresh, isRefreshing, onExport }) {
  return (
    <header className="sticky top-0 z-20 border-b border-stone-200 bg-[#f7f7f2]/92 backdrop-blur-xl">
      <div className="mx-auto flex min-h-[72px] w-full max-w-[1440px] flex-wrap items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-md bg-ink text-white shadow-soft">
            <BarChart3 size={20} aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold leading-tight sm:text-2xl">LD-gpt-check 数据看板</h1>
            <p className="mt-1 text-xs text-stone-500">更新于 {formatDateTime(updatedAt)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button className="icon-button" type="button" title="刷新数据" onClick={onRefresh}>
            <RefreshCcw className={isRefreshing ? "animate-spin" : ""} size={18} aria-hidden="true" />
          </button>
          <button className="command-button" type="button" onClick={onExport}>
            <Download size={17} aria-hidden="true" />
            导出
          </button>
        </div>
      </div>
    </header>
  );
}

function Sidebar({ filters, models, onChange }) {
  return (
    <aside className="h-max rounded-md border border-stone-200 bg-white p-4 shadow-soft lg:sticky lg:top-24">
      <div className="mb-4 flex items-center gap-2 text-sm font-semibold">
        <Search size={16} aria-hidden="true" />
        筛选
      </div>
      <div className="grid gap-4">
        <label className="grid gap-2">
          <span className="text-xs font-medium text-stone-500">时间范围</span>
          <div className="grid grid-cols-3 rounded-md border border-stone-200 bg-stone-100 p-1 lg:grid-cols-1">
            {RANGE_OPTIONS.map((option) => (
              <button
                className={filters.range === option.value ? "segmented is-active" : "segmented"}
                key={option.value}
                type="button"
                onClick={() => onChange((current) => ({ ...current, range: option.value }))}
              >
                {option.label}
              </button>
            ))}
          </div>
        </label>
        <label className="grid gap-2">
          <span className="text-xs font-medium text-stone-500">模型</span>
          <select
            className="h-10 rounded-md border border-stone-200 bg-white px-3 text-sm outline-none transition focus:border-sea focus:ring-4 focus:ring-teal-100"
            value={filters.model}
            onChange={(event) => onChange((current) => ({ ...current, model: event.target.value }))}
          >
            <option value="all">全部模型</option>
            {models.map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="mt-5 rounded-md border border-stone-200 bg-stone-50 p-3 text-xs leading-5 text-stone-500">
        当前数据来自 Cloudflare Worker 的 D1 聚合接口。
      </div>
    </aside>
  );
}

function SummaryGrid({ summary }) {
  const cards = [
    { label: "提交量", value: compact(summary.submissions), icon: Activity, tone: "teal", meta: "benchmark_submissions" },
    { label: "活跃用户", value: compact(summary.activeUsers), icon: Users, tone: "blue", meta: "近窗口去重" },
    { label: "平均准确率", value: percent(summary.averageAccuracy), icon: ShieldCheck, tone: "green", meta: "按提交加权" },
    { label: "平均耗时", value: `${summary.averageLatencySeconds}s`, icon: Clock3, tone: "amber", meta: "每题平均" },
    { label: "平均 TPS", value: summary.averageTps, icon: Gauge, tone: "rose", meta: "输出速度" },
    { label: "Token 总量", value: compact(summary.tokenTotal), icon: Sparkles, tone: "slate", meta: "输入+输出估算" },
  ];

  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
      {cards.map((card) => (
        <article className="metric-card" key={card.label}>
          <div className={`metric-icon tone-${card.tone}`}>
            <card.icon size={18} aria-hidden="true" />
          </div>
          <div className="mt-4 text-sm text-stone-500">{card.label}</div>
          <div className="mt-1 truncate text-2xl font-semibold tabular-nums">{card.value}</div>
          <div className="mt-3 truncate text-xs text-stone-400">{card.meta}</div>
        </article>
      ))}
    </section>
  );
}

function StatisticsPanel({ statistics }) {
  const cards = [
    {
      label: "准确率 95% CI",
      value: `${percent(statistics.accuracy.ci95Low)} - ${percent(statistics.accuracy.ci95High)}`,
      meta: `n=${statistics.accuracy.sampleSize.toLocaleString("zh-CN")}，误差 ${percent(statistics.accuracy.marginOfError)}`,
    },
    {
      label: "准确率标准差",
      value: percent(statistics.accuracy.stdDev),
      meta: "模型间离散度",
    },
    {
      label: "P95 耗时",
      value: `${statistics.latency.p95}s`,
      meta: `中位数 ${statistics.latency.median}s，标准差 ${statistics.latency.stdDev}s`,
    },
    {
      label: "回归 z 检验",
      value: `z=${statistics.regression.zScore}`,
      meta: `p=${formatPValue(statistics.regression.pValue)}，${verdictLabel(statistics.regression.verdict)}`,
    },
  ];

  return (
    <Panel title="统计置信度" icon={ShieldCheck} action="95% CI / z-test">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <div className="stat-card" key={card.label}>
            <span>{card.label}</span>
            <strong>{card.value}</strong>
            <em>{card.meta}</em>
          </div>
        ))}
      </div>
      <div className="mt-4 min-w-0 overflow-x-auto">
        <table className="data-table stats-table">
          <thead>
            <tr>
              <th>模型</th>
              <th>样本量</th>
              <th>准确率</th>
              <th>95% CI</th>
              <th>相对最佳</th>
              <th>判断</th>
            </tr>
          </thead>
          <tbody>
            {statistics.modelComparisons.map((item) => (
              <tr key={item.model}>
                <td>{item.model}</td>
                <td>{item.sampleSize.toLocaleString("zh-CN")}</td>
                <td>{percent(item.accuracy)}</td>
                <td>
                  {percent(item.ci95Low)} - {percent(item.ci95High)}
                </td>
                <td>{signedPercent(item.deltaVsBest)}</td>
                <td>
                  <StatusBadge status={modelVerdictStatus(item.verdict)} label={modelVerdictLabel(item.verdict)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function TestPanel({ coverage }) {
  return (
    <Panel title="测试矩阵" icon={Activity} action={percent(coverage.passRate)}>
      <div className="grid gap-2">
        {coverage.suites.map((suite) => (
          <div className="test-row" key={suite.label}>
            <div>
              <strong>{suite.label}</strong>
              <span>
                {suite.passed.toLocaleString("zh-CN")} / {suite.total.toLocaleString("zh-CN")}
              </span>
            </div>
            <Progress value={suite.passed / suite.total} />
            <StatusBadge status={suite.status === "pass" ? "healthy" : "watch"} />
          </div>
        ))}
      </div>
      <div className="mt-4 grid grid-cols-3 gap-2">
        <div className="mini-stat">
          <span>观察项</span>
          <strong>{coverage.watchCount}</strong>
        </div>
        <div className="mini-stat">
          <span>回退项</span>
          <strong>{coverage.regressionCount}</strong>
        </div>
        <div className="mini-stat">
          <span>波动题</span>
          <strong>{coverage.flakyQuestions}</strong>
        </div>
      </div>
    </Panel>
  );
}

function PairwisePanel({ tests }) {
  return (
    <Panel title="模型显著性" icon={BarChart3} action="Holm 校正">
      <div className="min-w-0 overflow-x-auto">
        <table className="data-table pairwise-table">
          <thead>
            <tr>
              <th>模型</th>
              <th>对照</th>
              <th>差值</th>
              <th>z</th>
              <th>p</th>
              <th>Holm p</th>
              <th>效应量 h</th>
              <th>判断</th>
            </tr>
          </thead>
          <tbody>
            {tests.map((item) => (
              <tr key={item.model}>
                <td>{item.model}</td>
                <td>{item.comparedTo}</td>
                <td>{signedPercent(item.delta)}</td>
                <td>{item.zScore}</td>
                <td>{formatPValue(item.pValue)}</td>
                <td>{formatPValue(item.adjustedPValue)}</td>
                <td>{item.effectSize}</td>
                <td>
                  <StatusBadge status={pairwiseStatus(item.verdict)} label={pairwiseLabel(item.verdict)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function PowerPanel({ statistics }) {
  const stability = statistics.trendStability;
  return (
    <Panel title="检验效能" icon={Gauge} action="power 80%">
      <div className="grid gap-2">
        <div className="power-row">
          <span>最小可检测差异</span>
          <strong>{percent(statistics.power.minimumDetectableEffect)}</strong>
        </div>
        <div className="power-row">
          <span>平均模型样本量</span>
          <strong>{statistics.power.averageModelSampleSize.toLocaleString("zh-CN")}</strong>
        </div>
        <div className="power-row">
          <span>最新准确率 z-score</span>
          <strong>{stability.latestZScore}</strong>
        </div>
      </div>
      <div className="mt-4 grid gap-2">
        {statistics.power.requiredSamples.map((item) => (
          <div className="sample-row" key={item.delta}>
            <span>检测 {percent(item.delta)} 差异</span>
            <strong>{item.perGroup.toLocaleString("zh-CN")} / 组</strong>
          </div>
        ))}
      </div>
      <div className="mt-4 rounded-md border border-stone-200 bg-stone-50 p-3 text-xs leading-5 text-stone-500">
        控制线：{percent(stability.lowerControlLimit)} - {percent(stability.upperControlLimit)}；异常点 {stability.anomalies.length} 个。
      </div>
    </Panel>
  );
}

function TimeOfDayPanel({ analysis }) {
  const worstHour = analysis.summary.worstHour;
  const worstSegment = analysis.summary.worstSegment;
  const [selectedHour, setSelectedHour] = useState(worstHour?.hour ?? 0);
  const selected = analysis.hourly.find((hour) => hour.hour === selectedHour) || analysis.hourly[0];

  useEffect(() => {
    setSelectedHour(worstHour?.hour ?? 0);
  }, [worstHour?.hour]);

  return (
    <Panel title="时段降智分析" icon={Clock3} action="jStat + Holm">
      <div className="grid gap-3 md:grid-cols-4">
        <div className="stat-card">
          <span>卡方总体检验</span>
          <strong>p={formatPValue(analysis.omnibus.pValue)}</strong>
          <em>{timeOmnibusLabel(analysis.omnibus.verdict)}，df={analysis.omnibus.degreesOfFreedom}</em>
        </div>
        <div className="stat-card">
          <span>最差小时</span>
          <strong>{worstHour ? worstHour.label : "无显著"}</strong>
          <em>{worstHour ? `${signedPercent(worstHour.deltaVsDay)}，Holm p=${formatPValue(worstHour.adjustedPValue)}` : "未检测到降智时段"}</em>
        </div>
        <div className="stat-card">
          <span>最差分段</span>
          <strong>{worstSegment ? worstSegment.label : "无"}</strong>
          <em>{worstSegment ? `${percent(worstSegment.accuracy)}，${signedPercent(worstSegment.deltaVsDay)}` : "样本不足"}</em>
        </div>
        <div className="stat-card">
          <span>受影响尝试</span>
          <strong>{analysis.summary.affectedAttempts.toLocaleString("zh-CN")}</strong>
          <em>Holm 校正后显著低于全天均值</em>
        </div>
      </div>

      <div className="mt-4 grid gap-5 xl:grid-cols-[minmax(0,1fr)_340px]">
        <div className="min-w-0">
          <div className="hour-grid" aria-label="24 小时准确率热力图">
            {analysis.hourly.map((hour) => (
              <button
                className={`hour-cell hour-${hour.verdict}${selected.hour === hour.hour ? " is-selected" : ""}`}
                key={hour.hour}
                title={`${hour.label} ${percent(hour.accuracy)}`}
                type="button"
                onClick={() => setSelectedHour(hour.hour)}
              >
                <span>{String(hour.hour).padStart(2, "0")}</span>
                <strong>{percent(hour.accuracy)}</strong>
                <em>{signedPercent(hour.deltaVsDay)}</em>
              </button>
            ))}
          </div>
        </div>

        <div className="grid gap-3">
          <div className="hour-detail-card">
            <span>选中小时</span>
            <strong>{selected.label}</strong>
            <dl>
              <div><dt>准确率</dt><dd>{percent(selected.accuracy)}</dd></div>
              <div><dt>95% CI</dt><dd>{percent(selected.ci95Low)} - {percent(selected.ci95High)}</dd></div>
              <div><dt>Holm p</dt><dd>{formatPValue(selected.adjustedPValue)}</dd></div>
              <div><dt>效应量 h</dt><dd>{selected.effectSize}</dd></div>
              <div><dt>风险分</dt><dd>{selected.riskScore}</dd></div>
              <div><dt>平均耗时</dt><dd>{selected.avgLatencySeconds}s</dd></div>
            </dl>
          </div>
          {analysis.segments.map((segment) => (
            <div className="time-segment-row" key={segment.label}>
              <div>
                <strong>{segment.label}</strong>
                <span>
                  {String(segment.startHour).padStart(2, "0")}:00-{String(segment.endHour + 1).padStart(2, "0")}:00
                </span>
              </div>
              <b>{percent(segment.accuracy)}</b>
              <StatusBadge status={timeStatus(segment.verdict)} label={timeLabel(segment.verdict)} />
            </div>
          ))}
        </div>
      </div>

      <div className="mt-4 grid gap-2 md:grid-cols-3">
        {(analysis.degradationWindows.length ? analysis.degradationWindows.slice(0, 3) : [{ label: "无显著窗口", attempts: 0, riskScore: 0, minDelta: 0 }]).map((window) => (
          <div className="time-window-card" key={window.label}>
            <span>{window.label}</span>
            <strong>{signedPercent(window.minDelta)}</strong>
            <em>risk {window.riskScore}，n={window.attempts.toLocaleString("zh-CN")}</em>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function ForecastPanel({ forecast }) {
  const chartData = useMemo(
    () =>
      forecast.accuracy.forecast.map((item, index) => ({
        step: `+${item.step}`,
        accuracyPct: Math.round(item.value * 1000) / 10,
        accuracyLowPct: Math.round(item.low * 1000) / 10,
        accuracyHighPct: Math.round(item.high * 1000) / 10,
        submissions: forecast.submissions.forecast[index]?.value ?? 0,
      })),
    [forecast],
  );

  return (
    <Panel title="趋势预测" icon={TrendingUp} action="OLS forecast">
      <div className="grid gap-3 md:grid-cols-4">
        <div className="stat-card">
          <span>准确率斜率</span>
          <strong>{signedPercentagePoints(forecast.accuracy.slope)}/日</strong>
          <em>p={formatPValue(forecast.accuracy.pValue)}，R2={forecast.accuracy.rSquared}</em>
        </div>
        <div className="stat-card">
          <span>准确率判断</span>
          <strong>{forecastLabel(forecast.accuracy.verdict)}</strong>
          <em>残差 SD {percent(forecast.accuracy.residualStdDev)}</em>
        </div>
        <div className="stat-card">
          <span>提交量斜率</span>
          <strong>{forecast.submissions.slope}/日</strong>
          <em>p={formatPValue(forecast.submissions.pValue)}，R2={forecast.submissions.rSquared}</em>
        </div>
        <div className="stat-card">
          <span>7 日预测</span>
          <strong>{Math.round(forecast.submissions.forecast.at(-1)?.value ?? 0).toLocaleString("zh-CN")}</strong>
          <em>第 7 天提交量点预测</em>
        </div>
      </div>
      <div className="chart-h-md mt-4">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 12, right: 10, left: -18, bottom: 0 }}>
            <CartesianGrid vertical={false} stroke="#e7e5e4" />
            <XAxis dataKey="step" tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="left" domain={[60, 100]} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <Tooltip content={<ChartTooltip />} />
            <Legend iconType="circle" />
            <Line yAxisId="left" type="monotone" dataKey="accuracyPct" name="准确率 %" stroke="#0f766e" strokeWidth={2.5} dot />
            <Line yAxisId="right" type="monotone" dataKey="submissions" name="提交量" stroke="#2563eb" strokeWidth={2.5} dot />
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div className="mt-3 min-w-0 overflow-x-auto">
        <table className="data-table forecast-table">
          <thead>
            <tr>
              <th>步长</th>
              <th>准确率</th>
              <th>95% 区间</th>
              <th>提交量</th>
            </tr>
          </thead>
          <tbody>
            {forecast.accuracy.forecast.map((item, index) => {
              const submissions = forecast.submissions.forecast[index];
              return (
                <tr key={item.step}>
                  <td>+{item.step} 天</td>
                  <td>{percent(item.value)}</td>
                  <td>
                    {percent(item.low)} - {percent(item.high)}
                  </td>
                  <td>{Math.round(submissions?.value ?? 0).toLocaleString("zh-CN")}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function CorrelationPanel({ correlations }) {
  return (
    <Panel title="相关性扫描" icon={BarChart3} action="Pearson r">
      <div className="grid gap-2">
        {correlations.map((item) => (
          <div className="correlation-row" key={item.metric}>
            <div>
              <strong>{item.metric}</strong>
              <span>
                n={item.sampleSize}，{correlationStrengthLabel(item.strength)}
              </span>
            </div>
            <b>{item.r}</b>
            <em>p={formatPValue(item.pValue)}</em>
            <StatusBadge status={correlationStatus(item.verdict, item.r)} label={correlationLabel(item.verdict)} />
          </div>
        ))}
      </div>
    </Panel>
  );
}

function ModelRankingPanel({ ranking }) {
  return (
    <Panel title="贝叶斯模型排名" icon={ShieldCheck} action="posterior">
      <div className="grid gap-2">
        {ranking.map((item, index) => (
          <div className="ranking-row" key={item.model}>
            <span>{index + 1}</span>
            <div>
              <strong>{item.model}</strong>
              <em>
                mean {percent(item.posteriorMean)}，loss {percent(item.expectedLoss)}
              </em>
            </div>
            <b>{percent(item.probabilityBest)}</b>
            <StatusBadge status={modelRankingStatus(item.verdict)} label={modelRankingLabel(item.verdict)} />
          </div>
        ))}
      </div>
      <div className="mt-4 rounded-md border border-stone-200 bg-stone-50 p-3 text-xs leading-5 text-stone-500">
        这里用 beta 后验正态近似估算“成为最佳”的概率，适合排序和发布候选判断，不替代离线评审。
      </div>
    </Panel>
  );
}

function RobustnessPanel({ robustness }) {
  const recentOutliers = robustness.recentOutliers.slice(0, 5);
  const questionOutliers = robustness.questionOutliers.slice(0, 5);

  return (
    <Panel title="鲁棒异常检测" icon={Gauge} action="MAD z-score">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="mini-stat">
          <span>准确率中位数</span>
          <strong>{percent(robustness.baselines.submissionAccuracyMedian)}</strong>
        </div>
        <div className="mini-stat">
          <span>耗时中位数</span>
          <strong>{robustness.baselines.submissionLatencyMedian}s</strong>
        </div>
        <div className="mini-stat">
          <span>失败率中位数</span>
          <strong>{percent(robustness.baselines.questionFailureMedian)}</strong>
        </div>
      </div>
      <div className="mt-4 grid gap-3">
        <div>
          <h3 className="mb-2 text-sm font-semibold text-stone-800">提交异常</h3>
          <div className="grid gap-2">
            {(recentOutliers.length ? recentOutliers : [{ id: "无异常提交", model: "MAD 阈值内", accuracyRobustZ: 0, latencyRobustZ: 0 }]).map((item) => (
              <div className="outlier-row" key={item.id}>
                <div>
                  <strong>{item.id}</strong>
                  <span>{item.model}</span>
                </div>
                <b>{item.accuracyRobustZ}</b>
                <em>{item.latencyRobustZ}</em>
              </div>
            ))}
          </div>
        </div>
        <div>
          <h3 className="mb-2 text-sm font-semibold text-stone-800">题目异常</h3>
          <div className="grid gap-2">
            {(questionOutliers.length ? questionOutliers : [{ questionId: "无异常题目", title: "MAD 阈值内", failureRate: 0, failureRobustZ: 0 }]).map((item) => (
              <div className="outlier-row" key={item.questionId}>
                <div>
                  <strong>{item.title}</strong>
                  <span>{item.questionId}</span>
                </div>
                <b>{percent(item.failureRate)}</b>
                <em>{item.failureRobustZ}</em>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Panel>
  );
}

function QuestionDiagnosticsPanel({ diagnostics }) {
  return (
    <Panel title="题目诊断优先级" icon={Activity} action="Wilson + z-score">
      <div className="min-w-0 overflow-x-auto">
        <table className="data-table diagnostic-table">
          <thead>
            <tr>
              <th>题目</th>
              <th>准确率</th>
              <th>95% CI</th>
              <th>难度 z</th>
              <th>优先级</th>
              <th>判断</th>
            </tr>
          </thead>
          <tbody>
            {diagnostics.map((item) => (
              <tr key={item.questionId}>
                <td>
                  <div className="min-w-[190px]">
                    <strong>{item.title}</strong>
                    <span>
                      {item.questionId}，n={item.attempts}
                    </span>
                  </div>
                </td>
                <td>{percent(item.accuracy)}</td>
                <td>
                  {percent(item.ci95Low)} - {percent(item.ci95High)}
                </td>
                <td>{item.difficultyZ}</td>
                <td>{item.priorityScore}</td>
                <td>
                  <StatusBadge status={questionDiagnosticStatus(item.verdict)} label={questionDiagnosticLabel(item.verdict)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function RiskBudgetPanel({ budget }) {
  return (
    <Panel title="质量风险预算" icon={ShieldCheck} action={riskBudgetLabel(budget.verdict)}>
      <div className="risk-gauge">
        <span>剩余预算</span>
        <strong>{signedPercent(budget.budgetRemaining)}</strong>
        <em>burn {budget.burnRate}x，目标 {percent(budget.targetAccuracy)}</em>
        <div>
          <i style={{ width: `${clampPercent(Math.max(0, budget.budgetRemaining))}%` }} />
        </div>
      </div>
      <div className="mt-4 grid gap-2">
        <div className="budget-row">
          <span>失败 / 允许</span>
          <strong>
            {budget.failures.toLocaleString("zh-CN")} / {budget.allowedFailures.toLocaleString("zh-CN")}
          </strong>
        </div>
        <div className="budget-row">
          <span>超额失败</span>
          <strong>{budget.excessFailures.toLocaleString("zh-CN")}</strong>
        </div>
        <div className="budget-row">
          <span>降智尝试占比</span>
          <strong>{percent(budget.degradedAttemptShare)}</strong>
        </div>
        <div className="budget-row">
          <span>审计题 / 异常负载</span>
          <strong>
            {budget.auditQuestions} / {budget.outlierLoad}
          </strong>
        </div>
      </div>
    </Panel>
  );
}

function DriftPanel({ drift }) {
  const chartData = drift.ewma.series.map((item, index) => ({
    date: item.date,
    ewmaPct: Math.round(item.value * 1000) / 10,
    cusum: drift.cusum.series[index]?.value ?? 0,
  }));

  return (
    <Panel title="窗口漂移监控" icon={TrendingUp} action="EWMA / CUSUM">
      <div className="grid gap-3 md:grid-cols-4">
        <div className="stat-card">
          <span>准确率漂移</span>
          <strong>{signedPercent(drift.window.delta)}</strong>
          <em>p={formatPValue(drift.window.pValue)}，z={drift.window.zScore}</em>
        </div>
        <div className="stat-card">
          <span>提交量漂移</span>
          <strong>{signedNumber(drift.volume.delta)}</strong>
          <em>p={formatPValue(drift.volume.pValue)}，t={drift.volume.tScore}</em>
        </div>
        <div className="stat-card">
          <span>EWMA 最新</span>
          <strong>{percent(drift.ewma.latest)}</strong>
          <em>{driftLabel(drift.ewma.verdict)}，lambda={drift.ewma.lambda}</em>
        </div>
        <div className="stat-card">
          <span>CUSUM 信号</span>
          <strong>{drift.cusum.signalScore}</strong>
          <em>{riskStatusLabel(drift.cusum.verdict)}，latest {drift.cusum.latest}</em>
        </div>
      </div>
      <div className="chart-h-md mt-4">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 12, right: 8, left: -18, bottom: 0 }}>
            <CartesianGrid vertical={false} stroke="#e7e5e4" />
            <XAxis dataKey="date" minTickGap={34} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="left" domain={[60, 100]} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <Tooltip content={<ChartTooltip />} />
            <Legend iconType="circle" />
            <Line yAxisId="left" type="monotone" dataKey="ewmaPct" name="EWMA %" stroke="#0f766e" strokeWidth={2.5} dot={false} />
            <Line yAxisId="right" type="monotone" dataKey="cusum" name="CUSUM" stroke="#be123c" strokeWidth={2.2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </Panel>
  );
}

function DistributionPanel({ shape }) {
  const rows = [
    { label: "日准确率", unit: "percent", ...shape.dailyAccuracy },
    { label: "日提交量", unit: "number", ...shape.dailySubmissions },
    { label: "近期耗时", unit: "seconds", ...shape.recentLatency },
    { label: "题目失败率", unit: "percent", ...shape.questionFailure },
    { label: "小时准确率", unit: "percent", ...shape.hourlyAccuracy },
  ];

  return (
    <Panel title="分布形态" icon={BarChart3} action="IQR / moments">
      <div className="min-w-0 overflow-x-auto">
        <table className="data-table distribution-table">
          <thead>
            <tr>
              <th>指标</th>
              <th>中位数</th>
              <th>IQR</th>
              <th>CV</th>
              <th>偏度</th>
              <th>峰度</th>
              <th>尾部风险</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((item) => (
              <tr key={item.label}>
                <td>
                  <strong>{item.label}</strong>
                  <span>
                    {formatDistributionValue(item.min, item.unit)} - {formatDistributionValue(item.max, item.unit)}
                  </span>
                </td>
                <td>{formatDistributionValue(item.median, item.unit)}</td>
                <td>{formatDistributionValue(item.iqr, item.unit)}</td>
                <td>{item.coefficientOfVariation}</td>
                <td>{item.skewness}</td>
                <td>{item.excessKurtosis}</td>
                <td>{percent(item.tailRisk)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function EfficiencyFrontierPanel({ frontier }) {
  return (
    <Panel title="效率前沿" icon={Gauge} action="Pareto frontier">
      <div className="grid gap-2">
        {frontier.map((item) => (
          <div className="frontier-row" key={item.model}>
            <div>
              <strong>{item.model}</strong>
              <span>
                {percent(item.accuracy)}，{item.avgTps} TPS，{item.avgTimeSeconds}s
              </span>
            </div>
            <b>{item.utilityScore}</b>
            <StatusBadge status={frontierStatus(item.verdict)} label={frontierLabel(item.verdict)} />
          </div>
        ))}
      </div>
      <div className="mt-4 rounded-md border border-stone-200 bg-stone-50 p-3 text-xs leading-5 text-stone-500">
        前沿模型不存在同时更准、更快、耗时更低的支配者；utility 用准确率、TPS、耗时加权用于排序。
      </div>
    </Panel>
  );
}

function TrendPanel({ trend }) {
  const chartData = useMemo(
    () =>
      trend.map((item) => ({
        ...item,
        accuracyPct: Math.round(item.accuracy * 1000) / 10,
      })),
    [trend],
  );

  return (
    <Panel title="趋势" icon={TrendingUp} action="提交量 / 准确率">
      <div className="chart-h-lg">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 12, right: 8, left: -18, bottom: 0 }}>
            <CartesianGrid vertical={false} stroke="#e7e5e4" />
            <XAxis dataKey="date" minTickGap={30} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis yAxisId="right" orientation="right" domain={[60, 100]} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <Tooltip content={<ChartTooltip />} />
            <Legend iconType="circle" />
            <Line yAxisId="left" type="monotone" dataKey="submissions" name="提交量" stroke="#0f766e" strokeWidth={2.5} dot={false} />
            <Line yAxisId="right" type="monotone" dataKey="accuracyPct" name="准确率 %" stroke="#b45309" strokeWidth={2.5} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </Panel>
  );
}

function ModelPanel({ models }) {
  return (
    <Panel title="模型对比" icon={BarChart3} action="准确率 / TPS">
      <div className="chart-h-md">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={models} layout="vertical" margin={{ top: 8, right: 14, left: 32, bottom: 8 }}>
            <CartesianGrid horizontal={false} stroke="#e7e5e4" />
            <XAxis type="number" domain={[0, 100]} tickLine={false} axisLine={false} tick={{ fill: "#78716c", fontSize: 12 }} />
            <YAxis dataKey="model" type="category" tickLine={false} axisLine={false} tick={{ fill: "#44403c", fontSize: 12 }} width={92} />
            <Tooltip content={<ChartTooltip percentKeys={["accuracyPct"]} />} />
            <Bar dataKey={(item) => Math.round(item.accuracy * 1000) / 10} name="准确率 %" radius={[0, 6, 6, 0]} fill="#0f766e" />
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div className="mt-4 grid gap-2">
        {models.map((model) => (
          <div className="model-row" key={model.model}>
            <span>{model.model}</span>
            <strong>{percent(model.accuracy)}</strong>
            <em>{model.avgTps} TPS</em>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function QualityPanel({ questions }) {
  return (
    <Panel title="题目质量" icon={ShieldCheck} action="低准确率优先">
      <div className="min-w-0 overflow-x-auto">
        <table className="data-table">
          <thead>
            <tr>
              <th>题目</th>
              <th>准确率</th>
              <th>尝试</th>
              <th>耗时</th>
              <th>失败率</th>
            </tr>
          </thead>
          <tbody>
            {questions.map((question) => (
              <tr key={question.questionId}>
                <td>
                  <div className="min-w-[180px]">
                    <strong>{question.title}</strong>
                    <span>{question.questionId}</span>
                  </div>
                </td>
                <td>
                  <Progress value={question.accuracy} />
                </td>
                <td>{question.attempts}</td>
                <td>{question.avgTimeSeconds}s</td>
                <td>{percent(question.failureRate)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function SegmentPanel({ segments }) {
  const total = segments.reduce((sum, item) => sum + item.count, 0);
  return (
    <Panel title="运行环境" icon={Gauge} action="系统分布">
      <div className="distribution-stack" aria-label="运行环境提交量分布">
        {segments.map((segment, index) => (
          <span
            key={segment.label}
            style={{
              width: `${(segment.count / total) * 100}%`,
              backgroundColor: SEGMENT_COLORS[index % SEGMENT_COLORS.length],
            }}
            title={`${segment.label}: ${segment.count}`}
          />
        ))}
      </div>
      <div className="grid gap-2">
        {segments.map((segment, index) => (
          <div className="segment-row" key={segment.label}>
            <span style={{ backgroundColor: SEGMENT_COLORS[index % SEGMENT_COLORS.length] }} />
            <strong>{segment.label}</strong>
            <em>{segment.count} 次</em>
            <b>{percent(segment.accuracy)}</b>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function RecentPanel({ submissions }) {
  return (
    <Panel title="最近提交" icon={Activity} action={`${submissions.length} 条`}>
      <div className="min-w-0 overflow-x-auto">
        <table className="data-table">
          <thead>
            <tr>
              <th>提交</th>
              <th>用户</th>
              <th>模型</th>
              <th>准确率</th>
              <th>题数</th>
              <th>平均耗时</th>
              <th>状态</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {submissions.map((submission) => (
              <tr key={submission.id}>
                <td className="font-mono text-xs text-stone-500">{submission.id}</td>
                <td>
                  <SubmissionUserCell user={submission.user} />
                </td>
                <td>{submission.model}</td>
                <td>{percent(submission.accuracy)}</td>
                <td>{submission.questionCount}</td>
                <td>{submission.avgTimeSeconds}s</td>
                <td>
                  <StatusBadge status={submission.status} />
                </td>
                <td>{relativeTime(submission.createdAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function SubmissionUserCell({ user }) {
  const displayUser = normalizeSubmissionUser(user);
  const avatar = displayUser.avatarUrl ? (
    <img src={displayUser.avatarUrl} alt="" loading="lazy" referrerPolicy="no-referrer" />
  ) : (
    <span aria-hidden="true">{displayUser.anonymous ? "匿名" : displayUser.displayName.slice(0, 1)}</span>
  );
  const name = displayUser.linuxdoUrl ? (
    <a href={displayUser.linuxdoUrl} target="_blank" rel="noreferrer">
      {displayUser.displayName}
    </a>
  ) : (
    <strong>{displayUser.displayName}</strong>
  );

  return (
    <div className={displayUser.anonymous ? "submission-user is-anonymous" : "submission-user"}>
      <div>{avatar}</div>
      {name}
    </div>
  );
}

function normalizeSubmissionUser(user) {
  if (!user || typeof user === "string") {
    return {
      anonymous: false,
      displayName: user || "Linux.do 用户",
      avatarUrl: "",
      linuxdoUrl: "",
    };
  }
  if (user.anonymous) {
    return {
      anonymous: true,
      displayName: "匿名",
      avatarUrl: "",
      linuxdoUrl: "",
    };
  }
  const username = user.username || "";
  return {
    anonymous: false,
    displayName: user.display_name || username || "Linux.do 用户",
    avatarUrl: user.avatar_url || "",
    linuxdoUrl: user.linuxdo_url || (username ? `https://linux.do/u/${encodeURIComponent(username)}/summary` : ""),
  };
}

function Panel({ title, icon: Icon, action, children }) {
  return (
    <section className="min-w-0 rounded-md border border-stone-200 bg-white p-4 shadow-soft sm:p-5">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <Icon size={18} aria-hidden="true" />
          <h2 className="truncate text-base font-semibold">{title}</h2>
        </div>
        <span className="shrink-0 rounded-md bg-stone-100 px-2.5 py-1 text-xs text-stone-500">{action}</span>
      </div>
      {children}
    </section>
  );
}

function Progress({ value }) {
  const pct = Math.round(value * 100);
  return (
    <div className="progress-cell">
      <span>{pct}%</span>
      <div>
        <i style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

function StatusBadge({ status, label }) {
  label = label || (status === "healthy" ? "稳定" : status === "watch" ? "观察" : "回退");
  return <span className={`status-badge status-${status}`}>{label}</span>;
}

function ChartTooltip({ active, payload, label }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-md border border-stone-200 bg-white px-3 py-2 text-xs shadow-soft">
      {label ? <div className="mb-1 font-semibold text-stone-700">{label}</div> : null}
      <div className="grid gap-1">
        {payload.map((item) => (
          <div className="flex items-center justify-between gap-5" key={`${item.name}-${item.value}`}>
            <span className="text-stone-500">{item.name}</span>
            <strong className="tabular-nums text-stone-900">{formatTooltipValue(item)}</strong>
          </div>
        ))}
      </div>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="grid min-h-screen place-items-center bg-[#f7f7f2]">
      <div className="rounded-md border border-stone-200 bg-white px-5 py-4 shadow-soft">加载 dashboard 数据...</div>
    </div>
  );
}

function ErrorState({ onRetry }) {
  return (
    <div className="grid min-h-screen place-items-center bg-[#f7f7f2] px-4">
      <div className="max-w-md rounded-md border border-rose-200 bg-white p-5 shadow-soft">
        <h1 className="text-lg font-semibold">Dashboard API 请求失败</h1>
        <p className="mt-2 text-sm text-stone-500">请确认本地 dev server 正在提供 /api/dashboard/overview。</p>
        <button className="command-button mt-4" type="button" onClick={onRetry}>
          <RefreshCcw size={17} aria-hidden="true" />
          重试
        </button>
      </div>
    </div>
  );
}

function formatTooltipValue(item) {
  if (String(item.name).includes("%")) return `${item.value}%`;
  return typeof item.value === "number" ? item.value.toLocaleString("zh-CN") : item.value;
}

function percent(value) {
  return `${Math.round(value * 1000) / 10}%`;
}

function signedPercent(value) {
  const sign = value > 0 ? "+" : "";
  return `${sign}${percent(value)}`;
}

function signedPercentagePoints(value) {
  const sign = value > 0 ? "+" : "";
  return `${sign}${Math.round(value * 10_000) / 100}pp`;
}

function formatPValue(value) {
  if (value < 0.0001) return "<0.0001";
  return String(value);
}

function verdictLabel(value) {
  if (value === "improved") return "显著提升";
  if (value === "regression") return "显著回退";
  return "未见显著差异";
}

function modelVerdictLabel(value) {
  if (value === "leader") return "最佳";
  if (value === "overlap") return "区间重叠";
  return "低于最佳";
}

function modelVerdictStatus(value) {
  if (value === "leader") return "healthy";
  if (value === "overlap") return "watch";
  return "regression";
}

function pairwiseLabel(value) {
  if (value === "leader") return "对照";
  if (value === "significant") return "显著";
  return "不显著";
}

function pairwiseStatus(value) {
  if (value === "leader") return "healthy";
  if (value === "significant") return "regression";
  return "watch";
}

function timeOmnibusLabel(value) {
  return value === "time_effect_detected" ? "存在显著时段效应" : "未见显著时段效应";
}

function timeLabel(value) {
  if (value === "degraded") return "降智";
  if (value === "elevated") return "偏高";
  return "正常";
}

function timeStatus(value) {
  if (value === "degraded") return "regression";
  if (value === "elevated") return "healthy";
  return "watch";
}

function forecastLabel(value) {
  if (value === "rising") return "上升";
  if (value === "falling") return "下降";
  if (value === "insufficient") return "样本不足";
  return "平稳";
}

function correlationStrengthLabel(value) {
  if (value === "strong") return "强相关";
  if (value === "moderate") return "中等相关";
  if (value === "weak") return "弱相关";
  if (value === "insufficient") return "样本不足";
  return "近似无关";
}

function correlationLabel(value) {
  return value === "significant" ? "显著" : "不显著";
}

function correlationStatus(value, r) {
  if (value !== "significant") return "watch";
  return r < 0 ? "regression" : "healthy";
}

function modelRankingLabel(value) {
  if (value === "ship") return "可发布";
  if (value === "candidate") return "候选";
  return "规避";
}

function modelRankingStatus(value) {
  if (value === "ship") return "healthy";
  if (value === "candidate") return "watch";
  return "regression";
}

function questionDiagnosticLabel(value) {
  if (value === "audit") return "审计";
  if (value === "watch") return "观察";
  return "正常";
}

function questionDiagnosticStatus(value) {
  if (value === "audit") return "regression";
  if (value === "watch") return "watch";
  return "healthy";
}

function riskBudgetLabel(value) {
  if (value === "over_budget") return "超预算";
  if (value === "watch") return "观察";
  return "健康";
}

function riskStatusLabel(value) {
  if (value === "alert") return "告警";
  if (value === "watch") return "观察";
  return "稳定";
}

function driftLabel(value) {
  if (value === "cooling") return "走低";
  if (value === "heating") return "走高";
  return "平稳";
}

function frontierLabel(value) {
  if (value === "frontier") return "前沿";
  if (value === "shadowed") return "被压制";
  return "被支配";
}

function frontierStatus(value) {
  if (value === "frontier") return "healthy";
  if (value === "shadowed") return "watch";
  return "regression";
}

function formatDistributionValue(value, unit) {
  if (unit === "percent") return percent(value);
  if (unit === "seconds") return `${value}s`;
  return value.toLocaleString("zh-CN");
}

function signedNumber(value) {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value}`;
}

function clampPercent(value) {
  return Math.round(clamp(value, 0, 1) * 100);
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function compact(value) {
  return Intl.NumberFormat("zh-CN", { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function formatDateTime(value) {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function relativeTime(value) {
  const minutes = Math.max(1, Math.round((Date.now() - new Date(value).getTime()) / 60_000));
  if (minutes < 60) return `${minutes} 分钟前`;
  return `${Math.round(minutes / 60)} 小时前`;
}

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
);
