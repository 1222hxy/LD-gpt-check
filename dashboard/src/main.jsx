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
import React, { useMemo, useState } from "react";
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
import "./styles.css";

const queryClient = new QueryClient();
const RANGE_OPTIONS = [
  { label: "7 天", value: "7d" },
  { label: "30 天", value: "30d" },
  { label: "90 天", value: "90d" },
];
const SEGMENT_COLORS = ["#0f766e", "#b45309", "#2563eb", "#be123c"];

function App() {
  const [filters, setFilters] = useState({ range: "30d", model: "all" });
  const dashboard = useQuery({
    queryKey: ["dashboard-overview", filters],
    queryFn: () => fetchDashboardOverview(filters),
    refetchInterval: 60_000,
  });

  if (dashboard.isLoading) return <LoadingState />;
  if (dashboard.isError) return <ErrorState onRetry={dashboard.refetch} />;

  const data = dashboard.data;
  return (
    <main className="min-h-screen bg-[#f7f7f2] text-ink">
      <TopBar updatedAt={data.updatedAt} onRefresh={dashboard.refetch} isRefreshing={dashboard.isFetching} />
      <section className="mx-auto grid w-full max-w-[1440px] gap-5 px-4 py-5 sm:px-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:px-8">
        <Sidebar filters={filters} models={data.filters.models} onChange={setFilters} />
        <div className="grid min-w-0 gap-5">
          <SummaryGrid summary={data.summary} />
          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <StatisticsPanel statistics={data.statistics} />
            <TestPanel coverage={data.statistics.testCoverage} />
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

function TopBar({ updatedAt, onRefresh, isRefreshing }) {
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
          <button className="command-button" type="button">
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
        当前数据来自本地 mock API；生产环境接 Cloudflare Worker 的 D1 聚合接口。
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
                <td>{submission.user}</td>
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
