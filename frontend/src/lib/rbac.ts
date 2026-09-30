import type { Role } from "@/lib/schemas";
import type { IncidentStatus } from "@/lib/types";

export type NavItem =
  | "dashboard"
  | "incidents"
  | "corrective-actions"
  | "reports"
  | "locations"
  | "users";

const ALL_ROLES: Role[] = [
  "SUPER_ADMIN",
  "ADMIN",
  "HSE_MANAGER",
  "HSE_OFFICER",
  "SUPERVISOR",
  "REPORTER",
];

const REPORT_ROLES: Role[] = ["SUPER_ADMIN", "ADMIN", "HSE_MANAGER", "HSE_OFFICER"];
const ADMIN_ROLES: Role[] = ["SUPER_ADMIN", "ADMIN"];

const visibility: Record<NavItem, readonly Role[]> = {
  dashboard: ALL_ROLES,
  incidents: ALL_ROLES,
  "corrective-actions": ALL_ROLES,
  reports: REPORT_ROLES,
  locations: ADMIN_ROLES,
  users: ADMIN_ROLES,
};

export function canSeeNavItem(role: string, item: NavItem) {
  return visibility[item].includes(role as Role);
}

export function isAdminRole(role: string) {
  return ADMIN_ROLES.includes(role as Role);
}

export type WorkflowAction = "verify" | "reject" | "close" | "startCA";

export function workflowActionsFor(role: string, status: IncidentStatus): WorkflowAction[] {
  const actions: WorkflowAction[] = [];
  if (status === "PENDING_REVIEW") {
    if (role === "SUPERVISOR" || role === "HSE_MANAGER" || role === "SUPER_ADMIN") {
      actions.push("verify");
    }
    if (
      role === "SUPERVISOR" ||
      role === "HSE_OFFICER" ||
      role === "HSE_MANAGER" ||
      role === "SUPER_ADMIN"
    ) {
      actions.push("reject");
    }
  }
  if (status === "UNDER_INVESTIGATION") {
    if (role === "HSE_OFFICER" || role === "HSE_MANAGER" || role === "SUPER_ADMIN") {
      actions.push("startCA");
    }
    if (role === "HSE_MANAGER" || role === "SUPER_ADMIN") {
      actions.push("close");
    }
  }
  if (status === "CORRECTIVE_ACTION" && (role === "HSE_MANAGER" || role === "SUPER_ADMIN")) {
    actions.push("close");
  }
  return actions;
}

export function canContinueDraft(role: string, status: IncidentStatus, isOwner: boolean) {
  if (status !== "DRAFT" && status !== "REJECTED") return false;
  if (isOwner) return true;
  return role === "HSE_OFFICER" || role === "HSE_MANAGER" || role === "SUPER_ADMIN";
}
