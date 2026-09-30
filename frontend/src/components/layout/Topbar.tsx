"use client";

import { Bell } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { NotificationList } from "@/components/notifications/NotificationList";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverHeader, PopoverTitle, PopoverTrigger } from "@/components/ui/popover";
import { roleLabels, type Role } from "@/lib/schemas";
import { useLogout, useSession } from "@/lib/queries/useSession";
import { useNotifications } from "@/lib/queries/useNotifications";

export function Topbar() {
  const session = useSession();
  const logout = useLogout();
  const router = useRouter();
  const notifications = useNotifications();
  const user = session.data;
  const role = user && user.role in roleLabels ? roleLabels[user.role as Role] : user?.role;
  const unread = (notifications.data ?? []).filter((item) => !item.isRead).length;

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-white px-6">
      <p className="text-sm text-subtle">Pelaporan insiden K3</p>
      <div className="flex items-center gap-3">
        <Popover>
          <PopoverTrigger
            render={
              <Button variant="ghost" size="icon" className="relative min-h-11 min-w-11" aria-label="Notifikasi" />
            }
          >
            <Bell className="size-4" />
            {unread > 0 ? (
              <span className="absolute top-1 right-1 inline-flex min-w-4 justify-center rounded-md bg-danger-500 px-1 text-[10px] text-white">
                {unread}
              </span>
            ) : null}
          </PopoverTrigger>
          <PopoverContent align="end" className="w-80 max-h-96 overflow-y-auto p-3">
            <PopoverHeader>
              <PopoverTitle>Notifikasi</PopoverTitle>
            </PopoverHeader>
            <NotificationList compact />
            <Link href="/notifications" className="mt-2 inline-block text-xs text-navy-600">
              Semua notifikasi
            </Link>
          </PopoverContent>
        </Popover>
        {user ? (
          <div className="text-right">
            <p className="text-sm font-medium text-ink">{user.name}</p>
            <p className="text-xs text-subtle">{role}</p>
          </div>
        ) : null}
        <Button
          variant="outline"
          size="sm"
          disabled={logout.isPending}
          onClick={() => {
            logout.mutate(undefined, {
              onSettled: () => router.replace("/login"),
            });
          }}
        >
          Keluar
        </Button>
      </div>
    </header>
  );
}
