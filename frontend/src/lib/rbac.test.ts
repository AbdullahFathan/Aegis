import { describe, expect, it } from "vitest";

import { canSeeNavItem, type NavItem } from "@/lib/rbac";
import { ROLES } from "@/lib/schemas";

const items: NavItem[] = [
  "dashboard",
  "incidents",
  "corrective-actions",
  "reports",
  "locations",
  "users",
];

describe("canSeeNavItem", () => {
  it.each(ROLES)("shows operational menus to %s", (role) => {
    expect(canSeeNavItem(role, "dashboard")).toBe(true);
    expect(canSeeNavItem(role, "incidents")).toBe(true);
    expect(canSeeNavItem(role, "corrective-actions")).toBe(true);
  });

  it("hides admin menus from a reporter", () => {
    expect(canSeeNavItem("REPORTER", "users")).toBe(false);
    expect(canSeeNavItem("REPORTER", "locations")).toBe(false);
    expect(canSeeNavItem("REPORTER", "reports")).toBe(false);
  });

  it("shows reports to HSE roles and admins only", () => {
    expect(canSeeNavItem("HSE_OFFICER", "reports")).toBe(true);
    expect(canSeeNavItem("HSE_MANAGER", "reports")).toBe(true);
    expect(canSeeNavItem("SUPERVISOR", "reports")).toBe(false);
    expect(canSeeNavItem("ADMIN", "users")).toBe(true);
    expect(canSeeNavItem("SUPER_ADMIN", "locations")).toBe(true);
  });

  it("covers every role and menu pair", () => {
    const seen = ROLES.flatMap((role) => items.map((item) => canSeeNavItem(role, item)));
    expect(seen).toHaveLength(ROLES.length * items.length);
  });
});
