import { describe, expect, it } from "vitest";

import {
  heatmapIntensity,
  mapDonutFromHeatmap,
  mapHeatmapMatrix,
  mapPipelineBuckets,
  mapTrendSeries,
  nearMissRateFromHeatmap,
} from "@/lib/dashboard";
import { categoryLabels } from "@/lib/schemas";

describe("dashboard mappers", () => {
  it("maps trend buckets to labeled series", () => {
    const series = mapTrendSeries([
      { year: 2026, month: 8, count: 2 },
      { year: 2026, month: 9, count: 5 },
    ]);
    expect(series[0]?.label).toContain("2026");
    expect(series[1]?.count).toBe(5);
  });

  it("maps pipeline to PRD lifecycle labels", () => {
    const buckets = mapPipelineBuckets({
      PENDING_REVIEW: 1,
      UNDER_INVESTIGATION: 2,
      CORRECTIVE_ACTION: 3,
      CLOSED: 4,
    });
    expect(buckets.map((row) => row.key)).toEqual([
      "PENDING_REVIEW",
      "UNDER_INVESTIGATION",
      "CORRECTIVE_ACTION",
      "CLOSED",
    ]);
    expect(buckets.find((row) => row.key === "CLOSED")?.label).toBe("Ditutup");
    expect(buckets.every((row) => !/open|resolved/i.test(row.label))).toBe(true);
  });

  it("aggregates heatmap cells into donut slices", () => {
    const slices = mapDonutFromHeatmap(
      [
        { locationId: "a", locationCode: "A", category: "NEAR_MISS", count: 3 },
        { locationId: "b", locationCode: "B", category: "NEAR_MISS", count: 1 },
        { locationId: "a", locationCode: "A", category: "LTI", count: 2 },
      ],
      categoryLabels,
    );
    expect(slices.find((row) => row.category === "NEAR_MISS")?.count).toBe(4);
    expect(slices.find((row) => row.category === "LTI")?.count).toBe(2);
    expect(slices.find((row) => row.category === "FATALITY")).toBeUndefined();
  });

  it("builds a heatmap matrix and intensity", () => {
    const matrix = mapHeatmapMatrix([
      { locationId: "a", locationCode: "SITE-A", category: "LTI", count: 4 },
      { locationId: "a", locationCode: "SITE-A", category: "NEAR_MISS", count: 1 },
    ]);
    expect(matrix.locations).toEqual([{ id: "a", code: "SITE-A" }]);
    expect(matrix.max).toBe(4);
    expect(matrix.values.a?.LTI).toBe(4);
    expect(heatmapIntensity(4, 4)).toBe(1);
    expect(heatmapIntensity(0, 4)).toBe(0);
  });

  it("computes near-miss rate or null when empty", () => {
    expect(nearMissRateFromHeatmap([])).toBeNull();
    expect(
      nearMissRateFromHeatmap([
        { locationId: "a", locationCode: "A", category: "NEAR_MISS", count: 1 },
        { locationId: "a", locationCode: "A", category: "LTI", count: 1 },
      ]),
    ).toBe(50);
  });
});
