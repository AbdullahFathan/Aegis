import type { Role } from "@/lib/schemas";

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
