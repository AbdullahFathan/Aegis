"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, refreshAccessToken, setAccessToken, setSessionClearedHandler } from "@/lib/api/client";
import type { User } from "@/lib/types";

export const sessionQueryKey = ["session"] as const;

async function loadSession(): Promise<User | null> {
  const refreshed = await refreshAccessToken();
  if (!refreshed) return null;
  try {
    return await api<User>("/auth/me");
  } catch {
    setAccessToken(null);
    return null;
  }
}

export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: loadSession,
    staleTime: Infinity,
    retry: false,
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { email: string; password: string }) =>
      api<{ accessToken: string; user: User }>("/auth/login", {
        method: "POST",
        body,
        auth: false,
      }),
    onSuccess: (data) => {
      setAccessToken(data.accessToken);
      queryClient.setQueryData(sessionQueryKey, data.user);
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api<{ loggedOut: boolean }>("/auth/logout", { method: "POST" }),
    onSettled: () => {
      setAccessToken(null);
      queryClient.setQueryData(sessionQueryKey, null);
    },
  });
}

export function registerSessionClear(queryClient: ReturnType<typeof useQueryClient>) {
  setSessionClearedHandler(() => {
    queryClient.setQueryData(sessionQueryKey, null);
  });
}
