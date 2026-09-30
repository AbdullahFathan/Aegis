"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import { dayEndRfc3339, dayStartRfc3339, toRfc3339 } from "@/lib/datetime";
import type { IncidentFormValues } from "@/lib/schemas";
import type {
  Incident,
  IncidentFile,
  IncidentFilters,
  IncidentList,
  WorkflowLog,
} from "@/lib/types";
import { incidentsQueryKey } from "@/lib/types";

export function buildIncidentListQuery(filters: IncidentFilters) {
  const params = new URLSearchParams({
    page: String(filters.page),
    pageSize: String(filters.pageSize),
  });
  if (filters.status) params.set("status", filters.status);
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", dayStartRfc3339(filters.from));
  if (filters.to) params.set("to", dayEndRfc3339(filters.to));
  return params.toString();
}

export function toIncidentPayload(values: IncidentFormValues) {
  const witnesses = values.witnesses
    .filter((row) => row.name.trim())
    .map((row) => ({ name: row.name.trim(), position: row.position.trim() }));
  return {
    title: values.title,
    description: values.description,
    category: values.category,
    severity: values.severity,
    incidentDatetime: toRfc3339(values.incidentDatetime),
    locationId: values.locationId,
    areaId: values.areaId || undefined,
    hasVictim: values.hasVictim,
    victimName: values.hasVictim ? values.victimName : undefined,
    victimPosition: values.hasVictim ? values.victimPosition || undefined : undefined,
    injuryDescription: values.hasVictim ? values.injuryDescription : undefined,
    initialTreatment: values.hasVictim ? values.initialTreatment || undefined : undefined,
    witnesses,
  };
}

export function useIncidents(filters: IncidentFilters) {
  return useQuery({
    queryKey: incidentsQueryKey(filters),
    queryFn: () => api<IncidentList>(`/incidents?${buildIncidentListQuery(filters)}`),
  });
}

export function useIncident(id: string | undefined) {
  return useQuery({
    queryKey: ["incident", id],
    queryFn: () => api<Incident>(`/incidents/${id}`),
    enabled: Boolean(id),
  });
}

export function useIncidentTimeline(id: string | undefined) {
  return useQuery({
    queryKey: ["incident-timeline", id],
    queryFn: () => api<WorkflowLog[]>(`/incidents/${id}/timeline`),
    enabled: Boolean(id),
  });
}

export function useIncidentFiles(id: string | undefined) {
  return useQuery({
    queryKey: ["incident-files", id],
    queryFn: () => api<IncidentFile[]>(`/incidents/${id}/files`),
    enabled: Boolean(id),
  });
}

function invalidateIncident(queryClient: ReturnType<typeof useQueryClient>, id?: string) {
  void queryClient.invalidateQueries({ queryKey: ["incidents"] });
  if (id) {
    void queryClient.invalidateQueries({ queryKey: ["incident", id] });
    void queryClient.invalidateQueries({ queryKey: ["incident-timeline", id] });
    void queryClient.invalidateQueries({ queryKey: ["incident-files", id] });
  }
}

export function useCreateIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: IncidentFormValues) =>
      api<Incident>("/incidents", { method: "POST", body: toIncidentPayload(values) }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function usePatchIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; values: IncidentFormValues; reason?: string }) =>
      api<Incident>(`/incidents/${input.id}`, {
        method: "PATCH",
        body: {
          ...toIncidentPayload(input.values),
          ...(input.reason ? { reason: input.reason } : {}),
        },
      }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useSubmitIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api<Incident>(`/incidents/${id}/submit`, { method: "POST" }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useVerifyIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; comment?: string }) =>
      api<Incident>(`/incidents/${input.id}/verify`, {
        method: "POST",
        body: input.comment ? { comment: input.comment } : undefined,
      }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useRejectIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; comment: string }) =>
      api<Incident>(`/incidents/${input.id}/reject`, {
        method: "POST",
        body: { comment: input.comment },
      }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useCloseIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; comment?: string }) =>
      api<Incident>(`/incidents/${input.id}/close`, {
        method: "POST",
        body: input.comment ? { comment: input.comment } : undefined,
      }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useStartCorrectiveAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; comment?: string }) =>
      api<Incident>(`/incidents/${input.id}/start-corrective-action`, {
        method: "POST",
        body: input.comment ? { comment: input.comment } : undefined,
      }),
    onSuccess: (incident) => {
      invalidateIncident(queryClient, incident.id);
    },
  });
}

export function useUploadIncidentFiles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { id: string; files: File[] }) => {
      const body = new FormData();
      for (const file of input.files) {
        body.append("files", file);
      }
      return api<IncidentFile[]>(`/incidents/${input.id}/files`, { method: "POST", body });
    },
    onSuccess: (_files, input) => {
      invalidateIncident(queryClient, input.id);
    },
  });
}
