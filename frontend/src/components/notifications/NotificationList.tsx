"use client";

import Link from "next/link";

import { EmptyState, LoadingBlock, QueryError } from "@/components/shared/EmptyState";
import { Button } from "@/components/ui/button";
import { notificationHref, sortNotificationsUnreadFirst } from "@/lib/notifications";
import { formatDateTime } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotifications,
} from "@/lib/queries/useNotifications";
import { cn } from "@/lib/utils";
import type { NotificationPriority } from "@/lib/types";

const priorityClass: Record<NotificationPriority, string> = {
  INFO: "border-l-[3px] border-l-navy-600 bg-navy-50",
  MEDIUM: "border-l-[3px] border-l-amber-400 bg-[#FFF8E1]",
  HIGH: "border-l-[3px] border-l-orange-500 bg-orange-50",
  CRITICAL: "border-l-[3px] border-l-danger-500 bg-[#FFEBEE]",
};

export function NotificationList({
  compact,
  onNavigate,
  typeFilter,
  unreadOnly,
  showMarkAll,
}: {
  compact?: boolean;
  onNavigate?: () => void;
  typeFilter?: string;
  unreadOnly?: boolean;
  showMarkAll?: boolean;
}) {
  const notifications = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();
  let items = sortNotificationsUnreadFirst(notifications.data ?? []);
  if (typeFilter && typeFilter !== "ALL") {
    items = items.filter((item) => item.type === typeFilter);
  }
  if (unreadOnly) items = items.filter((item) => !item.isRead);
  const visible = compact ? items.slice(0, 8) : items;

  if (notifications.isPending) {
    return compact ? <p className="text-sm text-subtle">Memuat notifikasi…</p> : <LoadingBlock />;
  }

  if (notifications.isError) {
    return (
      <QueryError message={mapApiError(notifications.error, "Gagal memuat notifikasi.")} />
    );
  }

  if (visible.length === 0) {
    if (compact) {
      return <p className="text-sm text-subtle">Tidak ada notifikasi.</p>;
    }
    return (
      <EmptyState
        title="Tidak ada notifikasi"
        description={
          unreadOnly || (typeFilter && typeFilter !== "ALL")
            ? "Tidak ada notifikasi sesuai filter. Ubah filter atau lihat semua notifikasi."
            : "Belum ada pemberitahuan untuk akun ini."
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {showMarkAll !== false && !compact ? (
        <Button
          variant="ghost"
          className="min-h-11 w-fit"
          disabled={markAll.isPending || items.every((item) => item.isRead)}
          onClick={() => void markAll.mutateAsync()}
        >
          Tandai semua dibaca
        </Button>
      ) : null}
      <ul className="flex flex-col gap-2">
        {visible.map((item) => (
          <li key={item.id}>
            <Link
              href={notificationHref(item)}
              className={cn(
                "block min-h-11 rounded-md border border-border p-3",
                priorityClass[item.priority],
                !item.isRead && "font-medium",
              )}
              onClick={() => {
                if (!item.isRead) void markRead.mutateAsync(item.id);
                onNavigate?.();
              }}
            >
              <p className="text-sm text-ink">{item.title}</p>
              <p className="mt-1 text-xs text-subtle">{item.body}</p>
              <p className="mt-1 text-xs text-subtle">{formatDateTime(item.createdAt)}</p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
