"use client";

import { useState } from "react";

import { NotificationList } from "@/components/notifications/NotificationList";
import { PageHeader } from "@/components/shared/EmptyState";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { notificationTypeLabels, NOTIFICATION_TYPES } from "@/lib/schemas";
import { useNotificationPreferences, usePutNotificationPreference } from "@/lib/queries/useNotifications";
import type { NotificationType } from "@/lib/types";

export function NotificationsScreen() {
  const prefs = useNotificationPreferences();
  const putPref = usePutNotificationPreference();
  const [type, setType] = useState<string>("ALL");
  const [unreadOnly, setUnreadOnly] = useState(false);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Notifikasi" />
      <div className="flex flex-col gap-4 md:flex-row">
        <Select value={type} onValueChange={(value) => setType(value ?? "ALL")}>
          <SelectTrigger className="h-9 w-full rounded-md md:w-64">
            <SelectValue placeholder="Semua tipe" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="ALL">Semua tipe</SelectItem>
              {NOTIFICATION_TYPES.map((value) => (
                <SelectItem key={value} value={value}>
                  {notificationTypeLabels[value]}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <label className="flex items-center gap-2 text-sm text-ink">
          <Checkbox checked={unreadOnly} onCheckedChange={(value) => setUnreadOnly(Boolean(value))} />
          Hanya belum dibaca
        </label>
      </div>
      <NotificationList typeFilter={type} unreadOnly={unreadOnly} />

      <section className="rounded-lg border border-border bg-white p-4">
        <h2 className="text-base font-medium text-ink">Preferensi email</h2>
        <p className="mt-1 text-xs text-subtle">
          Email mati secara default. Centang tipe event yang ingin dikirim ke email Anda.
        </p>
        <ul className="mt-4 flex flex-col gap-3">
          {NOTIFICATION_TYPES.map((eventType) => (
            <li key={eventType} className="flex items-start gap-2">
              <Checkbox
                checked={Boolean(prefs.data?.[eventType])}
                onCheckedChange={(value) => {
                  void putPref.mutateAsync({
                    eventType: eventType as NotificationType,
                    emailEnabled: Boolean(value),
                  });
                }}
              />
              <div>
                <p className="text-sm text-ink">{notificationTypeLabels[eventType]}</p>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
