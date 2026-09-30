"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import type { AreaValues, LocationValues, RegionValues } from "@/lib/schemas";
import type { Area, Region, Site, User, UserFilters, UserList } from "@/lib/types";
import { usersQueryKey } from "@/lib/types";

export function useUsers(filters: UserFilters, options?: { enabled?: boolean }) {
  const params = new URLSearchParams({
    page: String(filters.page),
    pageSize: String(filters.pageSize),
  });
  if (filters.role) params.set("role", filters.role);
  if (filters.status) params.set("status", filters.status);

  return useQuery({
    queryKey: usersQueryKey(filters),
    queryFn: () => api<UserList>(`/users?${params.toString()}`),
    enabled: options?.enabled ?? true,
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      name: string;
      email: string;
      password: string;
      role: string;
      status: string;
    }) => api<User>("/users", { method: "POST", body }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function usePatchUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; name: string; role: string; status: string }) =>
      api<User>(`/users/${input.id}`, {
        method: "PATCH",
        body: { name: input.name, role: input.role, status: input.status },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function useRegions() {
  return useQuery({
    queryKey: ["regions"],
    queryFn: () => api<Region[]>("/regions"),
  });
}

export function useLocations() {
  return useQuery({
    queryKey: ["locations"],
    queryFn: () => api<Site[]>("/locations"),
  });
}

export function useCreateRegion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: RegionValues) => api<Region>("/regions", { method: "POST", body }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["regions"] });
    },
  });
}

export function useCreateLocation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: LocationValues) =>
      api<Site>("/locations", {
        method: "POST",
        body: {
          name: values.name,
          code: values.code,
          type: values.type,
          regionId: values.regionId || null,
          supervisorId: values.supervisorId,
          hseOfficerId: values.hseOfficerId,
        },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["locations"] });
    },
  });
}

export function useDeactivateLocation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api<Site>(`/locations/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["locations"] });
    },
  });
}

export function useCreateArea() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AreaValues & { locationId: string }) =>
      api<Area>(`/locations/${input.locationId}/areas`, {
        method: "POST",
        body: { name: input.name, code: input.code },
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["locations"] });
    },
  });
}
