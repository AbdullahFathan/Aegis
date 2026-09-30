"use client";

import { Suspense } from "react";

import { ReportsScreen } from "@/components/reports/ReportsScreen";
import { LoadingBlock } from "@/components/shared/EmptyState";

export default function ReportsPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <ReportsScreen />
    </Suspense>
  );
}
