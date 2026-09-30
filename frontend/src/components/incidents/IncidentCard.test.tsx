import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { IncidentCard } from "@/components/incidents/IncidentCard";
import type { Incident } from "@/lib/types";

const incident: Incident = {
  id: "1",
  incidentNumber: "INC-2026-09-0042",
  title: "Tergelincir di crusher",
  description: "Deskripsi",
  category: "LTI",
  severity: "CRITICAL",
  escalationLevel: "L3",
  status: "CLOSED",
  incidentDatetime: "2026-09-01T08:00:00.000Z",
  locationId: "loc",
  areaId: null,
  reporterId: "u1",
  hasVictim: true,
  victimName: "Rudi",
  victimPosition: null,
  injuryDescription: null,
  initialTreatment: null,
  witnesses: [],
  pendingReviewAt: null,
  closedAt: "2026-09-02T08:00:00.000Z",
  closedById: "u2",
  createdAt: "2026-09-01T08:00:00.000Z",
  updatedAt: "2026-09-02T08:00:00.000Z",
};

describe("IncidentCard", () => {
  it("renders the incident code and badges", () => {
    render(
      <IncidentCard incident={incident} locationName="TMB-A — Tambang A" picName="Dewi" />,
    );
    expect(screen.getByText("INC-2026-09-0042")).toBeInTheDocument();
    expect(screen.getByText("Kritis")).toBeInTheDocument();
    expect(screen.getByText("Ditutup")).toBeInTheDocument();
    expect(screen.getByText("Tergelincir di crusher")).toBeInTheDocument();
  });
});
