"use client";

import { useQuery } from "@tanstack/react-query";

import { api, apiDownload, type ApiFileResult } from "@/lib/api/client";
import { dayEndRfc3339, dayStartRfc3339 } from "@/lib/datetime";
import type { ReportsFilterValues } from "@/lib/schemas";
import type { ArchiveReport, ReportJob } from "@/lib/types";

function pick(row: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    if (row[key] !== undefined && row[key] !== null) return row[key];
  }
  return null;
}

export function normalizeArchive(row: Record<string, unknown>): ArchiveReport {
  const stored = pick(row, "storedKey", "StoredKey");
  const err = pick(row, "errorMessage", "ErrorMessage", "error");
  return {
    id: String(pick(row, "id", "ID") ?? ""),
    type: String(pick(row, "type", "Type") ?? ""),
    status: String(pick(row, "status", "Status") ?? ""),
    storedKey: stored == null ? null : String(stored),
    createdAt: String(pick(row, "createdAt", "CreatedAt") ?? ""),
    errorMessage: err == null ? null : String(err),
  };
}

export function buildReportQuery(filters: ReportsFilterValues, format?: string) {
  const params = new URLSearchParams();
  if (format) params.set("format", format);
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", dayStartRfc3339(filters.from));
  if (filters.to) params.set("to", dayEndRfc3339(filters.to));
  return params.toString();
}

export function reportPath(filters: ReportsFilterValues) {
  if (filters.type === "investigation") {
    return `/reports/investigation/${filters.incidentId}`;
  }
  if (filters.type === "corrective-actions") return "/reports/corrective-actions";
  if (filters.type === "ltifr") return "/reports/ltifr";
  return "/reports/monthly";
}

export function useReportArchive(enabled: boolean) {
  return useQuery({
    queryKey: ["reports", "archive"],
    queryFn: async () => {
      const rows = await api<Record<string, unknown>[]>("/reports");
      return (rows ?? []).map((row) => normalizeArchive(row));
    },
    enabled,
  });
}

export function useReportJob(jobId: string | undefined) {
  return useQuery({
    queryKey: ["reports", "job", jobId],
    queryFn: () => api<ReportJob>(`/reports/jobs/${jobId}`),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status || status === "DONE" || status === "FAILED") return false;
      return 2000;
    },
  });
}

export async function requestReport(
  filters: ReportsFilterValues,
): Promise<ApiFileResult<ReportJob>> {
  const query = buildReportQuery(filters, filters.format);
  const path = `${reportPath(filters)}?${query}`;
  return apiDownload<ReportJob>(path);
}

export async function requestAuditCsv(query: string) {
  return apiDownload(`/audit-logs?${query}`);
}
