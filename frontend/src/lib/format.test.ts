import { describe, expect, it } from "vitest";

import { formatKpiNumber, formatPercent, formatRate } from "@/lib/format";

describe("kpi formatters", () => {
  it("formats integers with id-ID grouping", () => {
    expect(formatKpiNumber(1200)).toMatch(/1.?200/);
  });

  it("formats rates and treats missing as em dash", () => {
    expect(formatRate(null)).toBe("—");
    expect(formatRate(2)).toBe("2");
  });

  it("formats percents", () => {
    expect(formatPercent(50)).toContain("50");
  });
});
