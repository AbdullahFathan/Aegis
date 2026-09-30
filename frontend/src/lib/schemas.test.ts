import { describe, expect, it } from "vitest";

import {
  auditFilterSchema,
  caCreateSchema,
  caDoneSchema,
  caTrackerFilterSchema,
  incidentFilterSchema,
  incidentFormSchema,
  locationSchema,
  loginSchema,
  parseReportsSearchParams,
  rcaSchema,
  reportsFilterSchema,
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

describe("rcaSchema", () => {
  it("rejects more than five why entries", () => {
    const result = rcaSchema.safeParse({
      timeline: "",
      humanFactor: "",
      environmentFactor: "",
      equipmentFactor: "",
      fiveWhys: Array.from({ length: 6 }, () => ({ why: "a", answer: "b" })),
      fishbone: {
        man: "",
        machine: "",
        method: "",
        material: "",
        environment: "",
        measurement: "",
      },
    });
    expect(result.success).toBe(false);
  });

  it("requires all fishbone keys", () => {
    const result = rcaSchema.safeParse({
      timeline: "a",
      humanFactor: "",
      environmentFactor: "",
      equipmentFactor: "",
      fiveWhys: [],
      fishbone: { man: "", machine: "" },
    });
    expect(result.success).toBe(false);
  });
});

describe("ca schemas", () => {
  it("requires description and due date", () => {
    const result = caCreateSchema.safeParse({
      description: "",
      actionType: "IMMEDIATE",
      priority: "HIGH",
      assigneeId: "",
      dueDate: "",
    });
    expect(result.success).toBe(false);
  });

  it("rejects Done without completion notes", () => {
    expect(caDoneSchema.safeParse({ completionNotes: "" }).success).toBe(false);
    expect(caDoneSchema.safeParse({ completionNotes: "Sudah diperbaiki" }).success).toBe(true);
  });

  it("accepts OVERDUE in tracker filters", () => {
    const result = caTrackerFilterSchema.safeParse({ status: "OVERDUE", view: "table" });
    expect(result.success).toBe(true);
  });
});

describe("reports and audit schemas", () => {
  it("accepts pdf and csv report formats", () => {
    expect(reportsFilterSchema.parse({ type: "monthly", format: "pdf" }).format).toBe("pdf");
    expect(reportsFilterSchema.parse({ type: "ltifr", format: "csv" }).format).toBe("csv");
    expect(reportsFilterSchema.safeParse({ format: "xlsx" }).success).toBe(false);
  });

  it("drops invalid report format from search params", () => {
    const parsed = parseReportsSearchParams({ format: "docx", type: "monthly" });
    expect(parsed.format).toBe("pdf");
  });

  it("rejects unknown audit actions", () => {
    expect(auditFilterSchema.safeParse({ action: "DELETED", page: 1 }).success).toBe(false);
    expect(auditFilterSchema.parse({ action: "STATUS_CHANGED" }).action).toBe("STATUS_CHANGED");
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
