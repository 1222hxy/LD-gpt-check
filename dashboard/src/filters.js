export const DEFAULT_FILTERS = { range: "30d", model: "all" };
export const VALID_RANGES = new Set(["7d", "30d", "90d"]);

export function parseFilters(search, models = []) {
  const params = new URLSearchParams(search || "");
  const range = VALID_RANGES.has(params.get("range")) ? params.get("range") : DEFAULT_FILTERS.range;
  const requestedModel = params.get("model") || DEFAULT_FILTERS.model;
  const model =
    requestedModel === "all" || models.includes(requestedModel) || (models.length === 0 && requestedModel)
      ? requestedModel
      : DEFAULT_FILTERS.model;

  return { range, model };
}

export function writeFiltersToUrl(filters) {
  if (typeof window === "undefined") return;
  const url = new URL(window.location.href);
  url.searchParams.set("range", filters.range);
  url.searchParams.set("model", filters.model);
  window.history.replaceState({}, "", url);
}
