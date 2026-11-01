import { describe, expect, it } from "vitest";
import { formatDate, formatNumber, freshness } from "./format";

describe("format", () => {
  it("formats numbers in fr-FR", () => {
    expect(formatNumber(1300).replace(/\s/g, " ")).toBe("1 300");
    expect(formatNumber(undefined)).toBe("—");
  });
  it("formats a unix timestamp or dash", () => {
    expect(formatDate(0)).toBe("—");
    expect(formatDate(1700000000)).toContain("2023");
  });
  it("describes freshness in text", () => {
    const now = 1_000_000;
    expect(freshness(now - 30, now)).toBe("à l'instant");
    expect(freshness(now - 300, now)).toBe("il y a 5 min");
    expect(freshness(now - 7200, now)).toBe("il y a 2 h");
    expect(freshness(now - 172800, now)).toBe("il y a 2 j");
    expect(freshness(undefined, now)).toBe("");
  });
});
