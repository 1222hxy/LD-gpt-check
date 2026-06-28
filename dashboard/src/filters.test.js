import { describe, expect, it } from "vitest";
import { parseFilters } from "./filters.js";

describe("filter parsing", () => {
  it("accepts valid range and model values", () => {
    expect(parseFilters("?range=90d&model=o4-mini", ["o4-mini"])).toEqual({ range: "90d", model: "o4-mini" });
  });

  it("falls back on invalid values", () => {
    expect(parseFilters("?range=365d&model=unknown", ["gpt-5.5"])).toEqual({ range: "30d", model: "all" });
  });

  it("preserves requested model until the model list is loaded", () => {
    expect(parseFilters("?range=90d&model=o4-mini")).toEqual({ range: "90d", model: "o4-mini" });
  });
});
