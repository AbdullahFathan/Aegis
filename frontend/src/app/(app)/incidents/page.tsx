"use client";

import { Suspense } from "react";

import { IncidentsScreen } from "@/components/incidents/IncidentsScreen";
import { LoadingBlock } from "@/components/shared/EmptyState";

export default function IncidentsPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <IncidentsScreen />
    </Suspense>
  );
}
