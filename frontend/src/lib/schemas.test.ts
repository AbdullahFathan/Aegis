import { describe, expect, it } from "vitest";

import {
  incidentFilterSchema,
  incidentFormSchema,
  locationSchema,
  loginSchema,
  userFormSchema,
} from "@/lib/schemas";

describe("loginSchema", () => {
  it("requires email and password", () => {
    const result = loginSchema.safeParse({ email: "", password: "" });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((issue) => issue.path[0]);
      expect(fields).toContain("email");
      expect(fields).toContain("password");
    }
  });
});

describe("userFormSchema", () => {
  it("rejects a missing email", () => {
    const result = userFormSchema.safeParse({
      name: "Rudi",
      email: "",
      password: "rahasia",
      role: "REPORTER",
      status: "ACTIVE",
    });
    expect(result.success).toBe(false);
  });
});

const validIncident = {
  title: "Tergelincir di pit",
  description: "Kronologi kejadian dilaporkan dengan cukup detail untuk memenuhi lima puluh karakter.",
  category: "NEAR_MISS" as const,
  severity: "LOW" as const,
  incidentDatetime: "2026-09-30T08:00",
  locationId: "loc-1",
  areaId: "",
  hasVictim: false,
  victimName: "",
  victimPosition: "",
  injuryDescription: "",
  initialTreatment: "",
  witnesses: [{ name: "", position: "" }],
};

describe("incidentFormSchema", () => {
  it("rejects a missing location", () => {
    const result = incidentFormSchema.safeParse({ ...validIncident, locationId: "" });
    expect(result.success).toBe(false);
  });

  it("rejects a description shorter than 50 characters", () => {
    const result = incidentFormSchema.safeParse({ ...validIncident, description: "x".repeat(49) });
    expect(result.success).toBe(false);
  });

  it("requires victim fields when hasVictim is true", () => {
    const result = incidentFormSchema.safeParse({
      ...validIncident,
      hasVictim: true,
      victimName: "",
      injuryDescription: "",
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.issues.map((issue) => issue.path[0]);
      expect(paths).toContain("victimName");
      expect(paths).toContain("injuryDescription");
    }
  });
});

describe("incidentFilterSchema", () => {
  it("rejects a non-PRD status", () => {
    const result = incidentFilterSchema.safeParse({
      status: "IN_REVIEW",
      page: 1,
      pageSize: 20,
    });
    expect(result.success).toBe(false);
  });
});

describe("locationSchema", () => {
  it("requires a code and a known location type", () => {
    const result = locationSchema.safeParse({
      name: "Pit North",
      code: "",
      type: "LAIN",
      regionId: "",
      supervisorId: "sup",
      hseOfficerId: "hse",
    });
    expect(result.success).toBe(false);
  });
});
