"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { Sidebar } from "@/components/layout/Sidebar";
import { Topbar } from "@/components/layout/Topbar";
import { LoadingBlock } from "@/components/shared/EmptyState";
import { useSession } from "@/lib/queries/useSession";

export function PageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-canvas">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="flex-1 p-6">
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
