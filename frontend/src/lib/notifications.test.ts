import { describe, expect, it } from "vitest";

import { sortNotificationsUnreadFirst } from "@/lib/notifications";
import type { AppNotification } from "@/lib/types";

function item(partial: Partial<AppNotification>): AppNotification {
  return {
    id: "1",
    type: "CA_ASSIGNED",
    title: "CA",
    body: "body",
    priority: "MEDIUM",
    isRead: false,
    referenceType: "corrective_action",
    referenceId: "ca-1",
    createdAt: "2026-09-30T10:00:00.000Z",
    ...partial,
  };
}

describe("sortNotificationsUnreadFirst", () => {
  it("places unread items before read items", () => {
    const sorted = sortNotificationsUnreadFirst([
      item({ id: "read-old", isRead: true, createdAt: "2026-09-30T12:00:00.000Z" }),
      item({ id: "unread-new", isRead: false, createdAt: "2026-09-30T11:00:00.000Z" }),
      item({ id: "unread-old", isRead: false, createdAt: "2026-09-30T09:00:00.000Z" }),
    ]);
    expect(sorted.map((row) => row.id)).toEqual(["unread-new", "unread-old", "read-old"]);
  });
});
