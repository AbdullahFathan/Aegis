"use client";

import { useSearchParams } from "next/navigation";
import { Suspense } from "react";

import { IncidentForm } from "@/components/forms/IncidentForm";
import { PageHeader, LoadingBlock } from "@/components/shared/EmptyState";

function NewIncidentForm() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") ?? undefined;
  return <IncidentForm incidentId={id} />;
}

export default function NewIncidentPage() {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Laporan insiden baru" />
      <Suspense fallback={<LoadingBlock />}>
        <NewIncidentForm />
      </Suspense>
    </div>
  );
}
