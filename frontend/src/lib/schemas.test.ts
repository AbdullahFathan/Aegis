import { describe, expect, it } from "vitest";

import { locationSchema, loginSchema, userFormSchema } from "@/lib/schemas";

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
