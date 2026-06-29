import { describe, expect, it } from "vitest";
import { parseFilters } from "./filters.js";

describe("filter parsing", () => {
  it("accepts valid range and model values", () => {
    expect(parseFilters("?range=90d&model=o4-mini&channel=official:openai", ["o4-mini"], [{ key: "official:openai" }])).toEqual({
      range: "90d",
      model: "o4-mini",
      channel: "official:openai",
    });
  });

  it("falls back on invalid values", () => {
    expect(parseFilters("?range=365d&model=unknown&channel=missing", ["gpt-5.5"], [{ key: "official:openai" }])).toEqual({
      range: "30d",
      model: "all",
      channel: "all",
    });
  });

  it("preserves requested model and channel until option lists are loaded", () => {
    expect(parseFilters("?range=90d&model=o4-mini&channel=bridge:krill")).toEqual({
      range: "90d",
      model: "o4-mini",
      channel: "bridge:krill",
    });
  });
});
