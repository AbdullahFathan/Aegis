import { describe, expect, it } from "vitest";

import { colors, severityConfig, statusConfig } from "@/lib/tokens";

const lifecycle = [
  "DRAFT",
  "PENDING_REVIEW",
  "UNDER_INVESTIGATION",
  "CORRECTIVE_ACTION",
  "CLOSED",
  "REJECTED",
];

describe("tokens", () => {
  it("uses Warm Safety muted text with AA contrast on canvas", () => {
    expect(colors.muted).toBe("#475569");
  });

  it("limits incident status keys to the PRD lifecycle", () => {
    expect(Object.keys(statusConfig).sort()).toEqual([...lifecycle].sort());
    expect(Object.keys(statusConfig)).not.toContain("IN_REVIEW");
    expect(Object.keys(statusConfig)).not.toContain("OPEN");
    expect(Object.keys(statusConfig)).not.toContain("RESOLVED");
  });

  it("maps critical severity to the danger icon", () => {
    expect(severityConfig.CRITICAL.icon).toBe("AlertTriangle");
    expect(severityConfig.CRITICAL.badgeText).toBe(colors.dangerText);
    expect(statusConfig.PENDING_REVIEW.badgeBg).toBe(colors.navy[50]);
  });
});
