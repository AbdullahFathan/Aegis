"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { LoadingBlock } from "@/components/shared/EmptyState";
import { useSession } from "@/lib/queries/useSession";

export function HomeRedirect() {
  const session = useSession();
  const router = useRouter();

  useEffect(() => {
    if (session.isPending) return;
    router.replace(session.data ? "/dashboard" : "/login");
  }, [router, session.data, session.isPending]);

  return (
    <div className="min-h-screen bg-canvas p-6">
      <LoadingBlock />
    </div>
  );
}
