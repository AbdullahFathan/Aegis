import { describe, expect, it } from "vitest";

import {
  canSeeNavItem,
  canUpdateCAStatus,
  canVerifyCA,
  canWriteCA,
  canWriteRca,
  workflowActionsFor,
  type NavItem,
} from "@/lib/rbac";
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

describe("workflowActionsFor", () => {
  it("lets a supervisor verify and reject pending review", () => {
    expect(workflowActionsFor("SUPERVISOR", "PENDING_REVIEW")).toEqual(["verify", "reject"]);
  });

  it("does not let a reporter verify", () => {
    expect(workflowActionsFor("REPORTER", "PENDING_REVIEW")).toEqual([]);
  });

  it("lets an HSE officer reject but not verify", () => {
    expect(workflowActionsFor("HSE_OFFICER", "PENDING_REVIEW")).toEqual(["reject"]);
  });

  it("lets HSE officer start CA during investigation", () => {
    expect(workflowActionsFor("HSE_OFFICER", "UNDER_INVESTIGATION")).toEqual(["startCA"]);
  });

  it("lets HSE manager close from investigation or CA", () => {
    expect(workflowActionsFor("HSE_MANAGER", "UNDER_INVESTIGATION")).toEqual(["startCA", "close"]);
    expect(workflowActionsFor("HSE_MANAGER", "CORRECTIVE_ACTION")).toEqual(["close"]);
  });

  it("hides close from admin and reporter", () => {
    expect(workflowActionsFor("ADMIN", "UNDER_INVESTIGATION")).toEqual([]);
    expect(workflowActionsFor("REPORTER", "CORRECTIVE_ACTION")).toEqual([]);
  });
});

describe("CA and RCA permissions", () => {
  it("lets HSE officer verify CA and write RCA before close", () => {
    expect(canVerifyCA("HSE_OFFICER")).toBe(true);
    expect(canWriteCA("HSE_OFFICER")).toBe(true);
    expect(canWriteRca("HSE_OFFICER", "UNDER_INVESTIGATION")).toBe(true);
  });

  it("blocks RCA after close and blocks reporter verify", () => {
    expect(canWriteRca("HSE_MANAGER", "CLOSED")).toBe(false);
    expect(canVerifyCA("REPORTER")).toBe(false);
    expect(canWriteCA("SUPERVISOR")).toBe(false);
  });

  it("lets the assignee update CA status", () => {
    expect(canUpdateCAStatus("REPORTER", "u1", "u1")).toBe(true);
    expect(canUpdateCAStatus("REPORTER", "u1", "u2")).toBe(false);
    expect(canUpdateCAStatus("HSE_OFFICER", "u1", "u2")).toBe(true);
  });
});
