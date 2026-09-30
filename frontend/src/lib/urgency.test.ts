import { describe, expect, it } from "vitest";

import type { Incident } from "@/lib/types";
import { caUrgency, incidentSlaUrgency, incidentUrgency } from "@/lib/urgency";

const baseIncident: Incident = {
  id: "1",
  incidentNumber: "INC-2026-09-0001",
  title: "Tergelincir",
  description: "x".repeat(50),
  category: "NEAR_MISS",
  severity: "LOW",
  escalationLevel: "L1",
  status: "PENDING_REVIEW",
  incidentDatetime: "2026-09-30T00:00:00.000Z",
  locationId: "loc",
  areaId: null,
  reporterId: "r1",
  hasVictim: false,
  victimName: null,
  victimPosition: null,
  injuryDescription: null,
  initialTreatment: null,
  witnesses: null,
  pendingReviewAt: "2026-09-29T00:00:00.000Z",
  closedAt: null,
  closedById: null,
  createdAt: "2026-09-29T00:00:00.000Z",
  updatedAt: "2026-09-29T00:00:00.000Z",
};

describe("urgencyLevel", () => {
  it("marks fatality as critical", () => {
    expect(incidentUrgency({ ...baseIncident, category: "FATALITY", status: "UNDER_INVESTIGATION" })).toBe(
      "critical",
    );
  });

  it("marks SLA overdue after 24 hours pending review", () => {
    const now = Date.parse("2026-09-30T01:00:00.000Z");
    expect(incidentSlaUrgency(baseIncident, now)).toBe("sla_overdue");
  });

  it("marks SLA soon within four hours of the deadline", () => {
    const now = Date.parse("2026-09-29T21:00:00.000Z");
    expect(incidentSlaUrgency(baseIncident, now)).toBe("sla_soon");
  });

  it("marks overdue CA", () => {
    expect(caUrgency({ status: "OVERDUE", dueDate: "2026-09-01" })).toBe("ca_overdue");
    expect(caUrgency({ status: "OPEN", dueDate: "2020-01-01" })).toBe("ca_overdue");
    expect(caUrgency({ status: "VERIFIED", dueDate: "2020-01-01" })).toBe("none");
  });
});
