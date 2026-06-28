export function buildDashboardExport(data, filters) {
  return {
    exportedAt: new Date().toISOString(),
    filters,
    updatedAt: data.updatedAt,
    summary: data.summary,
    statistics: data.statistics,
  };
}

export function buildHourlyCsv(hourly) {
  const headers = [
    "hour",
    "attempts",
    "accuracy",
    "delta_vs_day",
    "adjusted_p_value",
    "effect_size",
    "risk_score",
    "verdict",
  ];
  const rows = hourly.map((item) => [
    item.label,
    item.attempts,
    item.accuracy,
    item.deltaVsDay,
    item.adjustedPValue,
    item.effectSize,
    item.riskScore,
    item.verdict,
  ]);

  return [headers, ...rows].map((row) => row.map(csvCell).join(",")).join("\n");
}

export function downloadDashboardExport(data, filters) {
  const payload = buildDashboardExport(data, filters);
  downloadText(`ld-gpt-check-dashboard-${filters.range}-${filters.model}.json`, JSON.stringify(payload, null, 2), "application/json");
  downloadText(`ld-gpt-check-hourly-${filters.range}-${filters.model}.csv`, buildHourlyCsv(data.statistics.timeOfDay.hourly), "text/csv");
}

function downloadText(filename, text, type) {
  const blob = new Blob([text], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function csvCell(value) {
  const text = String(value ?? "");
  if (!/[",\n]/.test(text)) return text;
  return `"${text.replace(/"/g, '""')}"`;
}
