"use client";

import { Clock, MapPin, UserRound } from "lucide-react";
import Link from "next/link";

import { IncidentCaPanel } from "@/components/corrective-actions/IncidentCaPanel";
import { FileUploader } from "@/components/incidents/FileUploader";
import { WorkflowActions } from "@/components/incidents/WorkflowActions";
import { RcaReadView } from "@/components/rca/RcaReadView";
import { EmptyState, LoadingBlock, QueryError } from "@/components/shared/EmptyState";
import { SeverityBadge, StatusBadge } from "@/components/shared/StatusBadge";
import { buttonVariants } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDateTime } from "@/lib/datetime";
import { mapIncidentError } from "@/lib/errors";
import { useLocations } from "@/lib/queries/useAdmin";
import {
  useIncident,
  useIncidentFiles,
  useIncidentTimeline,
  useUploadIncidentFiles,
} from "@/lib/queries/useIncidents";
import { useRca } from "@/lib/queries/useRca";
import { useSession } from "@/lib/queries/useSession";
import { canContinueDraft, canWriteRca } from "@/lib/rbac";
import { categoryLabels, severityLabels } from "@/lib/schemas";
import { statusConfig } from "@/lib/tokens";
import { incidentUrgency, urgencyRowClass } from "@/lib/urgency";
import { cn } from "@/lib/utils";
import type { IncidentStatus } from "@/lib/types";

const TABS = ["overview", "timeline", "files", "rca", "ca"] as const;

export function IncidentDetailScreen({
  incidentId,
  tab = "overview",
}: {
  incidentId: string;
  tab?: string;
}) {
  const session = useSession();
  const incident = useIncident(incidentId);
  const timeline = useIncidentTimeline(incidentId);
  const files = useIncidentFiles(incidentId);
  const rca = useRca(incidentId);
  const locations = useLocations();
  const uploadFiles = useUploadIncidentFiles();
  const initialTab = TABS.includes(tab as (typeof TABS)[number]) ? tab : "overview";

  if (incident.isPending) return <LoadingBlock />;
  if (incident.isError || !incident.data) {
    return (
      <QueryError message={mapIncidentError(incident.error, "Laporan tidak ditemukan.")} />
    );
  }

  const row = incident.data;
  const site = locations.data?.find((item) => item.id === row.locationId);
  const area = site?.areas.find((item) => item.id === row.areaId);
  const locationLabel = site ? `${site.code} — ${site.name}` : row.locationId;
  const pic =
    session.data?.id === row.reporterId
      ? session.data.name
      : site
        ? `Supervisor ${site.code}`
        : "PIC lokasi";
  const closed = row.status === "CLOSED";
  const canUpload =
    row.status === "DRAFT" ||
    row.status === "REJECTED" ||
    session.data?.role === "HSE_OFFICER" ||
    session.data?.role === "HSE_MANAGER" ||
    session.data?.role === "SUPER_ADMIN";
  const continueDraft =
    session.data &&
    canContinueDraft(session.data.role, row.status, session.data.id === row.reporterId);

  return (
    <div className="flex flex-col gap-6">
      <header
        className={cn(
          "rounded-lg border border-border bg-white p-4 shadow-card",
          urgencyRowClass(incidentUrgency(row)),
        )}
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <span className="font-mono text-xs text-subtle">{row.incidentNumber ?? "Draft"}</span>
          <SeverityBadge severity={row.severity} />
        </div>
        <h1 className="mt-2 text-2xl font-semibold text-ink">{row.title}</h1>
        <p className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-subtle">
          <span className="inline-flex items-center gap-1">
            <MapPin aria-hidden className="size-4" />
            {locationLabel}
            {area ? ` · ${area.name}` : ""}
          </span>
          <span className="inline-flex items-center gap-1">
            <Clock aria-hidden className="size-4" />
            {formatDateTime(row.incidentDatetime)}
          </span>
          <span className="inline-flex items-center gap-1">
            <UserRound aria-hidden className="size-4" />
            {pic}
          </span>
        </p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <StatusBadge status={row.status} />
          {closed ? (
            <span className="rounded-md border border-emerald-500 bg-[#E8F5E9] px-2 py-0.5 text-xs font-medium text-[#1B5E20]">
              Laporan Final — Tidak Dapat Diubah
            </span>
          ) : null}
        </div>
      </header>

      {session.data ? <WorkflowActions incident={row} role={session.data.role} /> : null}
      {continueDraft ? (
        <Link href={`/incidents/new?id=${row.id}`} className={cn(buttonVariants({ variant: "ghost" }), "min-h-11 w-fit")}>
          Lanjutkan pelaporan
        </Link>
      ) : null}

      <Tabs defaultValue={initialTab}>
        <TabsList variant="line">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="timeline">Timeline</TabsTrigger>
          <TabsTrigger value="files">Files</TabsTrigger>
          <TabsTrigger value="rca">RCA</TabsTrigger>
          <TabsTrigger value="ca">Corrective actions</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-4 flex flex-col gap-3">
          <p className="text-sm leading-relaxed text-ink">{row.description}</p>
          <p className="text-xs text-subtle">
            {categoryLabels[row.category]} · {severityLabels[row.severity]} · Eskalasi {row.escalationLevel}
          </p>
          {row.hasVictim ? (
            <div className="rounded-lg border border-border bg-white p-4">
              <h2 className="text-base font-medium text-ink">Korban</h2>
              <p className="mt-1 text-sm text-ink">{row.victimName}</p>
              {row.victimPosition ? <p className="text-xs text-subtle">{row.victimPosition}</p> : null}
              {row.injuryDescription ? (
                <p className="mt-2 text-sm leading-relaxed text-ink">{row.injuryDescription}</p>
              ) : null}
              {row.initialTreatment ? (
                <p className="mt-1 text-sm text-subtle">{row.initialTreatment}</p>
              ) : null}
            </div>
          ) : (
            <p className="text-sm text-subtle">Tidak ada korban.</p>
          )}
          {row.witnesses && row.witnesses.length > 0 ? (
            <div>
              <h2 className="text-base font-medium text-ink">Saksi</h2>
              <ul className="mt-2 flex flex-col gap-1 text-sm text-ink">
                {row.witnesses.map((witness, index) => (
                  <li key={`${witness.name}-${index}`}>
                    {witness.name}
                    {witness.position ? ` · ${witness.position}` : ""}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </TabsContent>
        <TabsContent value="timeline" className="mt-4">
          {timeline.isPending ? <LoadingBlock /> : null}
          {timeline.data?.length === 0 ? (
            <p className="text-sm text-subtle">Belum ada riwayat transisi.</p>
          ) : (
            <ol className="flex flex-col gap-3 border-l border-border pl-4">
              {timeline.data?.map((entry) => (
                <li key={entry.id} className="flex flex-col gap-1">
                  <p className="text-sm text-ink">
                    {statusConfig[entry.fromStatus].label} → {statusConfig[entry.toStatus].label}
                  </p>
                  <p className="text-xs text-subtle">
                    {entry.actorId === session.data?.id ? "Anda" : entry.actorId.slice(0, 8)} ·{" "}
                    {formatDateTime(entry.createdAt)}
                  </p>
                  {entry.comment ? <p className="text-sm leading-relaxed text-ink">{entry.comment}</p> : null}
                </li>
              ))}
            </ol>
          )}
        </TabsContent>
        <TabsContent value="files" className="mt-4">
          <FileUploader
            files={files.data ?? []}
            canUpload={Boolean(canUpload) && !closed}
            uploading={uploadFiles.isPending}
            onUpload={async (picked) => {
              await uploadFiles.mutateAsync({ id: row.id, files: picked });
            }}
            onRefreshFiles={async () => (await files.refetch()).data}
          />
        </TabsContent>
        <TabsContent value="rca" className="mt-4">
          {rca.isPending ? <LoadingBlock /> : null}
          {rca.data ? <RcaReadView rca={rca.data} /> : null}
          {!rca.isPending && !rca.data ? (
            <EmptyState title="Belum diisi" description="Hasil investigasi belum tersedia untuk laporan ini." />
          ) : null}
          {session.data && canWriteRca(session.data.role, row.status as IncidentStatus) ? (
            <Link
              href={`/incidents/${row.id}/rca`}
              className={cn(buttonVariants({ variant: "ghost" }), "mt-3 min-h-11 w-fit")}
            >
              Isi investigasi
            </Link>
          ) : null}
        </TabsContent>
        <TabsContent value="ca" className="mt-4">
          {session.data ? (
            <IncidentCaPanel
              incident={row}
              session={session.data}
              files={files.data ?? []}
              closed={closed}
              onRefreshFiles={async () => (await files.refetch()).data}
            />
          ) : null}
        </TabsContent>
      </Tabs>
    </div>
  );
}
