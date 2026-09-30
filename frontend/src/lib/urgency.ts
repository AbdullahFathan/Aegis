import type { CaStatus, Incident } from "@/lib/types";

export type UrgencyLevel = "critical" | "sla_overdue" | "sla_soon" | "ca_overdue" | "none";

const SLA_HOURS = 24;
const SLA_WARN_HOURS = 4;

export function incidentSlaUrgency(incident: Incident, now = Date.now()): UrgencyLevel {
  if (incident.status !== "PENDING_REVIEW" || !incident.pendingReviewAt) return "none";
  const started = new Date(incident.pendingReviewAt).getTime();
  if (Number.isNaN(started)) return "none";
  const deadline = started + SLA_HOURS * 60 * 60 * 1000;
  if (now >= deadline) return "sla_overdue";
  if (now >= deadline - SLA_WARN_HOURS * 60 * 60 * 1000) return "sla_soon";
  return "none";
}

export function incidentUrgency(incident: Incident, now = Date.now()): UrgencyLevel {
  if (
    incident.severity === "CRITICAL" ||
    incident.category === "LTI" ||
    incident.category === "FATALITY"
  ) {
    return "critical";
  }
  return incidentSlaUrgency(incident, now);
}

export function caUrgency(
  item: { status: CaStatus; dueDate: string },
  now = Date.now(),
): UrgencyLevel {
  if (item.status === "OVERDUE") return "ca_overdue";
  if (item.status === "DONE" || item.status === "VERIFIED") return "none";
  const due = new Date(item.dueDate).getTime();
  if (Number.isNaN(due)) return "none";
  const startOfToday = new Date(now);
  startOfToday.setHours(0, 0, 0, 0);
  if (due < startOfToday.getTime()) return "ca_overdue";
  return "none";
}

export function urgencyRowClass(level: UrgencyLevel) {
  if (level === "none") return "";
  if (level === "sla_soon") {
    return "border-l-[3px] border-l-amber-400 bg-[#FFF8E1]";
  }
  return "border-l-[3px] border-l-danger-500 bg-[#FFEBEE]";
}
