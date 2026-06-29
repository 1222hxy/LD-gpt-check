export const DEFAULT_FILTERS = { range: "30d", model: "all", channel: "all" };
export const VALID_RANGES = new Set(["7d", "30d", "90d"]);

export function parseFilters(search, models = [], channels = []) {
  const params = new URLSearchParams(search || "");
  const range = VALID_RANGES.has(params.get("range")) ? params.get("range") : DEFAULT_FILTERS.range;
  const requestedModel = params.get("model") || DEFAULT_FILTERS.model;
  const requestedChannel = params.get("channel") || DEFAULT_FILTERS.channel;
  const model =
    requestedModel === "all" || models.includes(requestedModel) || (models.length === 0 && requestedModel)
      ? requestedModel
      : DEFAULT_FILTERS.model;
  const channelKeys = channels.map((item) => item.key);
  const channel =
    requestedChannel === "all" || channelKeys.includes(requestedChannel) || (channels.length === 0 && requestedChannel)
      ? requestedChannel
      : DEFAULT_FILTERS.channel;

  return { range, model, channel };
}

export function writeFiltersToUrl(filters) {
  if (typeof window === "undefined") return;
  const url = new URL(window.location.href);
  url.searchParams.set("range", filters.range);
  url.searchParams.set("model", filters.model);
  url.searchParams.set("channel", filters.channel);
  window.history.replaceState({}, "", url);
}
