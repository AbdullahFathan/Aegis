"use client";

import { CAForm } from "@/components/forms/CAForm";
import { CAItem } from "@/components/corrective-actions/CAItem";
import { EmptyState, LoadingBlock, QueryError } from "@/components/shared/EmptyState";
import { mapApiError } from "@/lib/errors";
import { assigneeOptions } from "@/lib/assignees";
import { useLocations, useUsers } from "@/lib/queries/useAdmin";
import { useIncidentCorrectiveActions } from "@/lib/queries/useCorrectiveActions";
import { canWriteCA, isAdminRole } from "@/lib/rbac";
import type { Incident, IncidentFile, User } from "@/lib/types";

export function IncidentCaPanel({
  incident,
  session,
  files,
  closed,
  onRefreshFiles,
}: {
  incident: Incident;
  session: User;
  files: IncidentFile[];
  closed: boolean;
  onRefreshFiles: () => Promise<IncidentFile[] | undefined>;
}) {
  const cas = useIncidentCorrectiveActions(incident.id);
  const locations = useLocations();
  const users = useUsers({ page: 1, pageSize: 100 }, { enabled: isAdminRole(session.role) });
  const site = locations.data?.find((item) => item.id === incident.locationId);
  const assignees = assigneeOptions({
    session,
    site,
    reporterId: incident.reporterId,
    users: isAdminRole(session.role) ? users.data?.items : undefined,
  });

  if (cas.isPending) return <LoadingBlock />;
  if (cas.isError) {
    return <QueryError message={mapApiError(cas.error, "Gagal memuat corrective action.")} />;
  }

  const items = cas.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      {items.length === 0 ? (
        <EmptyState title="Belum diisi" description="Belum ada corrective action untuk laporan ini." />
      ) : (
        items.map((item) => (
          <CAItem
            key={item.id}
            item={item}
            role={session.role}
            userId={session.id}
            files={files}
            canUploadEvidence
            closed={closed}
            onRefreshFiles={onRefreshFiles}
          />
        ))
      )}
      {canWriteCA(session.role) && !closed ? (
        <CAForm incidentId={incident.id} assignees={assignees} />
      ) : null}
    </div>
  );
}
