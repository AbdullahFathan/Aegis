"use client";

import { Suspense } from "react";

import { AuditLogsScreen } from "@/components/admin/AuditLogsScreen";
import { LoadingBlock } from "@/components/shared/EmptyState";

export default function AuditLogsPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <AuditLogsScreen />
    </Suspense>
  );
}
