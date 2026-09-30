"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Sidebar } from "@/components/layout/Sidebar";
import { Topbar } from "@/components/layout/Topbar";
import { LoadingBlock } from "@/components/shared/EmptyState";
import { useSession } from "@/lib/queries/useSession";

export function PageShell({ children }: { children: React.ReactNode }) {
  const [navOpen, setNavOpen] = useState(false);

  return (
    <div className="flex min-h-screen bg-canvas">
      {navOpen ? (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-ink/40 md:hidden"
          aria-label="Tutup menu"
          onClick={() => setNavOpen(false)}
        />
      ) : null}
      <div
        className={
          navOpen
            ? "fixed inset-y-0 left-0 z-50 flex md:static md:z-auto"
            : "hidden md:flex"
        }
      >
        <Sidebar onNavigate={() => setNavOpen(false)} />
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar onOpenNav={() => setNavOpen(true)} />
        <main className="flex-1 p-4 md:p-6">
          <div className="mx-auto w-full max-w-[1280px]">{children}</div>
        </main>
      </div>
    </div>
  );
}

export function AuthGate({ children }: { children: React.ReactNode }) {
  const session = useSession();
  const router = useRouter();

  useEffect(() => {
    if (!session.isPending && !session.data) {
      router.replace("/login");
    }
  }, [router, session.data, session.isPending]);

  if (session.isPending || !session.data) {
    return (
      <div className="min-h-screen bg-canvas p-6">
        <LoadingBlock />
      </div>
    );
  }

  return <PageShell>{children}</PageShell>;
}
