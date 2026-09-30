"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api/client";
import type { AppNotification, NotificationType } from "@/lib/types";

export const notificationsQueryKey = ["notifications"] as const;
export const notificationPrefsQueryKey = ["notification-preferences"] as const;

export function useNotifications() {
  return useQuery({
    queryKey: notificationsQueryKey,
    queryFn: () => api<AppNotification[]>("/notifications"),
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api<void>(`/notifications/${id}/read`, { method: "PATCH" }),
    onSuccess: (_data, id) => {
      queryClient.setQueryData(notificationsQueryKey, (current: AppNotification[] | undefined) =>
        current?.map((item) => (item.id === id ? { ...item, isRead: true } : item)),
      );
    },
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const items = queryClient.getQueryData<AppNotification[]>(notificationsQueryKey) ?? [];
      const unread = items.filter((item) => !item.isRead);
      for (const item of unread) {
        await api<void>(`/notifications/${item.id}/read`, { method: "PATCH" });
      }
    },
    onSuccess: () => {
      queryClient.setQueryData(notificationsQueryKey, (current: AppNotification[] | undefined) =>
        current?.map((item) => ({ ...item, isRead: true })),
      );
    },
  });
}

export type EmailPreferenceMap = Partial<Record<NotificationType, boolean>>;

export function useNotificationPreferences() {
  return useQuery({
    queryKey: notificationPrefsQueryKey,
    queryFn: async (): Promise<EmailPreferenceMap> => ({}),
    staleTime: Infinity,
  });
}

export function usePutNotificationPreference() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { eventType: NotificationType; emailEnabled: boolean }) =>
      api<{ eventType: NotificationType; emailEnabled: boolean }>("/notifications/preferences", {
        method: "PUT",
        body: input,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(notificationPrefsQueryKey, (current: EmailPreferenceMap | undefined) => ({
        ...(current ?? {}),
        [data.eventType]: data.emailEnabled,
      }));
    },
  });
}
