"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, ApiError } from "@/lib/api/client";
import type { RcaFormValues } from "@/lib/schemas";
import type { IncidentCategory, Rca, RcaTemplate, RcaTemplatePayload } from "@/lib/types";

function emptyRcaQuery(error: unknown) {
  return error instanceof ApiError && error.status === 404;
}

export function useRca(incidentId: string | undefined) {
  return useQuery({
    queryKey: ["incident-rca", incidentId],
    queryFn: async () => {
      try {
        return await api<Rca>(`/incidents/${incidentId}/rca`);
      } catch (error) {
        if (emptyRcaQuery(error)) return null;
        throw error;
      }
    },
    enabled: Boolean(incidentId),
  });
}

export function useUpsertRca(incidentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RcaFormValues & { completed?: boolean }) =>
      api<Rca>(`/incidents/${incidentId}/rca`, {
        method: "PUT",
        body: {
          timeline: input.timeline,
          humanFactor: input.humanFactor,
          environmentFactor: input.environmentFactor,
          equipmentFactor: input.equipmentFactor,
          fiveWhys: input.fiveWhys,
          fishbone: input.fishbone,
          completed: Boolean(input.completed),
        },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["incident-rca", incidentId] });
    },
  });
}

export function useRcaTemplate(category: IncidentCategory | undefined) {
  return useQuery({
    queryKey: ["rca-template", category],
    queryFn: async () => {
      try {
        return await api<RcaTemplate>(`/rca-templates/${category}`);
      } catch (error) {
        if (emptyRcaQuery(error)) return null;
        throw error;
      }
    },
    enabled: Boolean(category),
  });
}

export function useSaveRcaTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { category: IncidentCategory; payload: RcaTemplatePayload }) =>
      api<RcaTemplate>(`/rca-templates/${input.category}`, {
        method: "PUT",
        body: input.payload,
      }),
    onSuccess: (_data, input) => {
      void queryClient.invalidateQueries({ queryKey: ["rca-template", input.category] });
    },
  });
}
