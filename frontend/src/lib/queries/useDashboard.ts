"use client";

import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import { dayEndRfc3339, dayStartRfc3339 } from "@/lib/datetime";
import type {
  DashboardFilters,
  DashboardHeatCell,
  DashboardMonthBucket,
  DashboardSummary,
} from "@/lib/types";
import { dashboardQueryKey } from "@/lib/types";

export function buildDashboardQuery(filters: DashboardFilters) {
  const params = new URLSearchParams();
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", dayStartRfc3339(filters.from));
  if (filters.to) params.set("to", dayEndRfc3339(filters.to));
  return params.toString();
}

export function useDashboardSummary(filters: DashboardFilters, enabled: boolean) {
  const query = buildDashboardQuery(filters);
  return useQuery({
    queryKey: [...dashboardQueryKey(filters), "summary"],
    queryFn: () => api<DashboardSummary>(`/dashboard/summary${query ? `?${query}` : ""}`),
    enabled,
  });
}

export function useDashboardTrends(filters: DashboardFilters, enabled: boolean) {
  const query = buildDashboardQuery(filters);
  return useQuery({
    queryKey: [...dashboardQueryKey(filters), "trends"],
    queryFn: () => api<DashboardMonthBucket[]>(`/dashboard/trends${query ? `?${query}` : ""}`),
    enabled,
  });
}

export function useDashboardHeatmap(filters: DashboardFilters, enabled: boolean) {
  const query = buildDashboardQuery(filters);
  return useQuery({
    queryKey: [...dashboardQueryKey(filters), "heatmap"],
    queryFn: () => api<DashboardHeatCell[]>(`/dashboard/heatmap${query ? `?${query}` : ""}`),
    enabled,
  });
}
