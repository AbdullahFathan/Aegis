import { INCIDENT_CATEGORIES, type IncidentCategoryValue } from "@/lib/schemas";
import { statusConfig } from "@/lib/tokens";
import type {
  CorrectiveAction,
  DashboardHeatCell,
  DashboardMonthBucket,
  DashboardPipeline,
  Incident,
} from "@/lib/types";

export type TrendPoint = { label: string; count: number; year: number; month: number };

const MONTH_LABELS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

export function mapTrendSeries(rows: DashboardMonthBucket[]): TrendPoint[] {
  return rows.map((row) => ({
    year: row.year,
    month: row.month,
    count: row.count,
    label: `${MONTH_LABELS[Math.max(0, row.month - 1)] ?? row.month} ${row.year}`,
  }));
}

export const PIPELINE_ORDER = [
  "PENDING_REVIEW",
  "UNDER_INVESTIGATION",
  "CORRECTIVE_ACTION",
  "CLOSED",
] as const;

export type PipelineBucket = {
  key: (typeof PIPELINE_ORDER)[number];
  label: string;
  count: number;
};

export function mapPipelineBuckets(pipeline: DashboardPipeline | undefined): PipelineBucket[] {
  const empty: DashboardPipeline = {
    PENDING_REVIEW: 0,
    UNDER_INVESTIGATION: 0,
    CORRECTIVE_ACTION: 0,
    CLOSED: 0,
  };
  const src = pipeline ?? empty;
  return PIPELINE_ORDER.map((key) => ({
    key,
    label: statusConfig[key].label,
    count: src[key] ?? 0,
  }));
}

export type DonutSlice = { category: IncidentCategoryValue; label: string; count: number };

export function mapDonutFromHeatmap(
  cells: DashboardHeatCell[],
  categoryLabels: Record<IncidentCategoryValue, string>,
): DonutSlice[] {
  const counts = new Map<IncidentCategoryValue, number>();
  for (const cat of INCIDENT_CATEGORIES) counts.set(cat, 0);
  for (const cell of cells) {
    const cat = cell.category as IncidentCategoryValue;
    if (!counts.has(cat)) continue;
    counts.set(cat, (counts.get(cat) ?? 0) + cell.count);
  }
  return INCIDENT_CATEGORIES.map((category) => ({
    category,
    label: categoryLabels[category],
    count: counts.get(category) ?? 0,
  })).filter((row) => row.count > 0);
}

export function nearMissRateFromHeatmap(cells: DashboardHeatCell[]): number | null {
  let near = 0;
  let total = 0;
  for (const cell of cells) {
    total += cell.count;
    if (cell.category === "NEAR_MISS") near += cell.count;
  }
  if (total === 0) return null;
  return (near / total) * 100;
}

export type HeatmapMatrix = {
  locations: { id: string; code: string }[];
  categories: IncidentCategoryValue[];
  max: number;
  values: Record<string, Record<string, number>>;
};

export function mapHeatmapMatrix(cells: DashboardHeatCell[]): HeatmapMatrix {
  const locations: { id: string; code: string }[] = [];
  const seen = new Set<string>();
  const values: Record<string, Record<string, number>> = {};
  let max = 0;
  for (const cell of cells) {
    if (!seen.has(cell.locationId)) {
      seen.add(cell.locationId);
      locations.push({ id: cell.locationId, code: cell.locationCode });
    }
    values[cell.locationId] ??= {};
    values[cell.locationId][cell.category] = cell.count;
    if (cell.count > max) max = cell.count;
  }
  return { locations, categories: [...INCIDENT_CATEGORIES], max, values };
}

export function heatmapIntensity(count: number, max: number) {
  if (max <= 0 || count <= 0) return 0;
  return Math.min(1, count / max);
}

const THREE_DAYS_MS = 3 * 24 * 60 * 60 * 1000;

export function isStaleQueueIncident(incident: Incident, now = Date.now()) {
  if (incident.status !== "PENDING_REVIEW" && incident.status !== "UNDER_INVESTIGATION") {
    return false;
  }
  const ref = incident.updatedAt || incident.pendingReviewAt || incident.createdAt;
  const ts = new Date(ref).getTime();
  if (Number.isNaN(ts)) return false;
  return now - ts > THREE_DAYS_MS;
}

export function casDueWithinDays(items: CorrectiveAction[], days = 7, now = Date.now()) {
  const startOfToday = new Date(now);
  startOfToday.setHours(0, 0, 0, 0);
  const end = now + days * 24 * 60 * 60 * 1000;
  return items.filter((item) => {
    if (item.status === "DONE" || item.status === "VERIFIED") return false;
    if (item.status === "OVERDUE") return true;
    const due = new Date(item.dueDate).getTime();
    if (Number.isNaN(due)) return false;
    return due >= startOfToday.getTime() && due <= end;
  });
}
