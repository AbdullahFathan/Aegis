"use client";

import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import { dayEndRfc3339, dayStartRfc3339 } from "@/lib/datetime";
import type { AuditFilterValues } from "@/lib/schemas";
import type { AuditLog, AuditLogList } from "@/lib/types";
import { auditLogsQueryKey } from "@/lib/types";

function pick(row: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    if (row[key] !== undefined && row[key] !== null) return row[key];
  }
  return null;
}

export function normalizeAuditLog(row: Record<string, unknown>): AuditLog {
  const userId = pick(row, "userId", "UserID", "userID");
  return {
    id: String(pick(row, "id", "ID") ?? ""),
    userId: userId == null ? "" : String(userId),
    userRole: String(pick(row, "userRole", "UserRole") ?? ""),
    ipAddress: String(pick(row, "ipAddress", "IPAddress") ?? ""),
    entityType: String(pick(row, "entityType", "EntityType") ?? ""),
    entityId: String(pick(row, "entityId", "EntityID") ?? ""),
    action: String(pick(row, "action", "Action") ?? ""),
    createdAt: String(pick(row, "createdAt", "CreatedAt") ?? ""),
  };
}

export function buildAuditQuery(filters: AuditFilterValues, format?: "csv") {
  const params = new URLSearchParams({
    page: String(filters.page),
    pageSize: String(filters.pageSize),
  });
  if (filters.userId) params.set("userId", filters.userId);
  if (filters.entityType) params.set("entityType", filters.entityType);
  if (filters.action) params.set("action", filters.action);
  if (filters.from) params.set("from", dayStartRfc3339(filters.from));
  if (filters.to) params.set("to", dayEndRfc3339(filters.to));
  if (format) params.set("format", format);
  return params.toString();
}

export function useAuditLogs(filters: AuditFilterValues, enabled: boolean) {
  const query = buildAuditQuery(filters);
  return useQuery({
    queryKey: auditLogsQueryKey(filters),
    queryFn: async () => {
      const data = await api<{
        items?: Record<string, unknown>[];
        total?: number;
        page?: number;
        pageSize?: number;
      }>(`/audit-logs?${query}`);
      const items = (data.items ?? []).map((row) => normalizeAuditLog(row));
      return {
        items,
        total: data.total ?? items.length,
        page: data.page ?? filters.page,
        pageSize: data.pageSize ?? filters.pageSize,
      } satisfies AuditLogList;
    },
    enabled,
  });
}
