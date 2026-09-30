"use client";

import { Suspense } from "react";

import { CATrackerScreen } from "@/components/corrective-actions/CATrackerScreen";
import { LoadingBlock } from "@/components/shared/EmptyState";

export default function CorrectiveActionsPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <CATrackerScreen />
    </Suspense>
  );
}
