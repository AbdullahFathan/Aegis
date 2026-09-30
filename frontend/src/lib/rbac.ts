import type { Role } from "@/lib/schemas";
import type { IncidentStatus } from "@/lib/types";

export type NavItem =
  | "dashboard"
  | "incidents"
  | "corrective-actions"
  | "reports"
  | "locations"
  | "users"
  | "audit-logs";

export type DashboardWidget =
  | "kpi"
  | "charts"
  | "officerQueue"
  | "supervisorQueue"
  | "reporterBrief"
  | "quickActions";

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
  "audit-logs": ADMIN_ROLES,
};

const DASHBOARD_API_ROLES: Role[] = ["SUPER_ADMIN", "ADMIN", "HSE_MANAGER"];

export function canSeeNavItem(role: string, item: NavItem) {
  return visibility[item].includes(role as Role);
}

export function isAdminRole(role: string) {
  return ADMIN_ROLES.includes(role as Role);
}

export function canReadDashboardApi(role: string) {
  return DASHBOARD_API_ROLES.includes(role as Role);
}

export function canExportReports(role: string) {
  return REPORT_ROLES.includes(role as Role);
}

export function canReadAuditLogs(role: string) {
  return ADMIN_ROLES.includes(role as Role);
}

export function dashboardWidgetsFor(role: string): DashboardWidget[] {
  if (canReadDashboardApi(role)) return ["kpi", "charts"];
  if (role === "HSE_OFFICER") return ["officerQueue", "quickActions"];
  if (role === "SUPERVISOR") return ["supervisorQueue"];
  if (role === "REPORTER") return ["reporterBrief"];
  return [];
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

const HSE_WRITE_ROLES: Role[] = ["SUPER_ADMIN", "HSE_MANAGER", "HSE_OFFICER"];

export function canWriteRca(role: string, incidentStatus: IncidentStatus) {
  if (incidentStatus === "CLOSED") return false;
  return HSE_WRITE_ROLES.includes(role as Role);
}

export function canWriteCA(role: string) {
  return HSE_WRITE_ROLES.includes(role as Role);
}

export function canVerifyCA(role: string) {
  return HSE_WRITE_ROLES.includes(role as Role);
}

export function canUpdateCAStatus(role: string, userId: string, assigneeId: string) {
  if (userId === assigneeId) return true;
  return canWriteCA(role);
}
