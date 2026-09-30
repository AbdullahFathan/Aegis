"use client";

import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { roleLabels, type Role } from "@/lib/schemas";
import { useLogout, useSession } from "@/lib/queries/useSession";

export function Topbar() {
  const session = useSession();
  const logout = useLogout();
  const router = useRouter();
  const user = session.data;
  const role = user && user.role in roleLabels ? roleLabels[user.role as Role] : user?.role;

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-white px-6">
      <p className="text-sm text-subtle">Pelaporan insiden K3</p>
      <div className="flex items-center gap-3">
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
