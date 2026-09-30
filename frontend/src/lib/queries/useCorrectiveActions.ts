"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import type { CaCreateValues } from "@/lib/schemas";
import type { CaTrackerFilters, CorrectiveAction } from "@/lib/types";
import { caTrackerQueryKey } from "@/lib/types";

export function buildCaTrackerQuery(filters: CaTrackerFilters) {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.assigneeId) params.set("assigneeId", filters.assigneeId);
  if (filters.priority) params.set("priority", filters.priority);
  if (filters.locationId) params.set("locationId", filters.locationId);
  return params.toString();
}

export function useIncidentCorrectiveActions(incidentId: string | undefined) {
  return useQuery({
    queryKey: ["incident-cas", incidentId],
    queryFn: () => api<CorrectiveAction[]>(`/incidents/${incidentId}/corrective-actions`),
    enabled: Boolean(incidentId),
  });
}

export function useCorrectiveActions(filters: CaTrackerFilters, options?: { enabled?: boolean }) {
  const query = buildCaTrackerQuery(filters);
  return useQuery({
    queryKey: caTrackerQueryKey(filters),
    queryFn: () => api<CorrectiveAction[]>(`/corrective-actions${query ? `?${query}` : ""}`),
    enabled: options?.enabled ?? true,
  });
}

function invalidateCa(queryClient: ReturnType<typeof useQueryClient>, incidentId?: string) {
  void queryClient.invalidateQueries({ queryKey: ["corrective-actions"] });
  if (incidentId) {
    void queryClient.invalidateQueries({ queryKey: ["incident-cas", incidentId] });
    void queryClient.invalidateQueries({ queryKey: ["incident-files", incidentId] });
  }
}

export function useCreateCorrectiveAction(incidentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: CaCreateValues) =>
      api<CorrectiveAction>(`/incidents/${incidentId}/corrective-actions`, {
        method: "POST",
        body: values,
      }),
    onSuccess: () => {
      invalidateCa(queryClient, incidentId);
    },
  });
}

export function usePatchCorrectiveAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      id: string;
      incidentId: string;
      status?: string;
      completionNotes?: string;
      description?: string;
      dueDate?: string;
      assigneeId?: string;
      actionType?: string;
      priority?: string;
    }) => {
      return api<CorrectiveAction>(`/corrective-actions/${input.id}`, {
        method: "PATCH",
        body: {
          description: input.description,
          actionType: input.actionType,
          priority: input.priority,
          assigneeId: input.assigneeId,
          dueDate: input.dueDate,
          status: input.status,
          completionNotes: input.completionNotes,
        },
      });
    },
    onSuccess: (row) => {
      invalidateCa(queryClient, row.incidentId);
    },
  });
}

export function useVerifyCorrectiveAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string }) =>
      api<CorrectiveAction>(`/corrective-actions/${input.id}/verify`, { method: "POST" }),
    onSuccess: (row) => {
      invalidateCa(queryClient, row.incidentId);
    },
  });
}
