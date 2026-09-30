import type { AppNotification } from "@/lib/types";

export function sortNotificationsUnreadFirst(items: AppNotification[]) {
  return [...items].sort((a, b) => {
    if (a.isRead !== b.isRead) return Number(a.isRead) - Number(b.isRead);
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });
}

export function notificationHref(item: AppNotification) {
  if (item.referenceType === "corrective_action") {
    return `/corrective-actions`;
  }
  if (item.referenceId) {
    return `/incidents/${item.referenceId}`;
  }
  return "/notifications";
}
