"use client";

import { useState } from "react";

import { FileUploader } from "@/components/incidents/FileUploader";
import { CaStatusBadge } from "@/components/shared/StatusBadge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { formatDate, formatDateTime } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import { useUploadIncidentFiles } from "@/lib/queries/useIncidents";
import { usePatchCorrectiveAction, useVerifyCorrectiveAction } from "@/lib/queries/useCorrectiveActions";
import { canUpdateCAStatus, canVerifyCA } from "@/lib/rbac";
import { caDoneSchema } from "@/lib/schemas";
import { cn } from "@/lib/utils";
import { caUrgency, urgencyRowClass } from "@/lib/urgency";
import type { CaStatus, CorrectiveAction, IncidentFile } from "@/lib/types";

const NEXT: Partial<Record<CaStatus, CaStatus>> = {
  OPEN: "IN_PROGRESS",
  IN_PROGRESS: "DONE",
  OVERDUE: "DONE",
};

export function CAItem({
  item,
  role,
  userId,
  files,
  canUploadEvidence,
  closed,
  onRefreshFiles,
}: {
  item: CorrectiveAction;
  role: string;
  userId: string;
  files: IncidentFile[];
  canUploadEvidence: boolean;
  closed: boolean;
  onRefreshFiles: () => Promise<IncidentFile[] | undefined>;
}) {
  const patch = usePatchCorrectiveAction();
  const verify = useVerifyCorrectiveAction();
  const upload = useUploadIncidentFiles();
  const [notes, setNotes] = useState(item.completionNotes ?? "");
  const [error, setError] = useState<string | null>(null);
  const canUpdate = canUpdateCAStatus(role, userId, item.assigneeId) && !closed;
  const next = NEXT[item.status];
  const showDoneFields = item.status === "IN_PROGRESS" || item.status === "OVERDUE" || item.status === "DONE";
  const evidence = files.filter((file) => file.correctiveActionId === item.id);

  async function run(status: CaStatus) {
    setError(null);
    if (status === "DONE") {
      const parsed = caDoneSchema.safeParse({ completionNotes: notes });
      if (!parsed.success) {
        setError(parsed.error.issues[0]?.message ?? "Catatan penyelesaian wajib diisi.");
        return;
      }
    }
    try {
      await patch.mutateAsync({
        id: item.id,
        incidentId: item.incidentId,
        status,
        ...(status === "DONE" ? { completionNotes: notes.trim() } : {}),
      });
    } catch (cause) {
      setError(mapApiError(cause, "Gagal memperbarui status."));
    }
  }

  return (
    <article className={cn("rounded-lg border border-border bg-white p-4", urgencyRowClass(caUrgency(item)))}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="text-sm leading-relaxed text-ink">{item.description}</p>
        <CaStatusBadge status={item.status} />
      </div>
      <p className="mt-2 text-xs text-subtle">
        Tenggat {formatDate(item.dueDate)}
        {item.completedAt ? ` · Selesai ${formatDateTime(item.completedAt)}` : ""}
      </p>
      {showDoneFields && canUpdate ? (
        <div className="mt-3">
          <label className="text-sm font-medium text-ink" htmlFor={`notes-${item.id}`}>
            Catatan penyelesaian
          </label>
          <Textarea
            id={`notes-${item.id}`}
            className="mt-1 min-h-20 rounded-md"
            value={notes}
            onChange={(event) => setNotes(event.target.value)}
            disabled={item.status === "DONE" || item.status === "VERIFIED"}
          />
        </div>
      ) : item.completionNotes ? (
        <p className="mt-2 text-sm leading-relaxed text-ink">{item.completionNotes}</p>
      ) : null}
      {error ? (
        <p role="alert" className="mt-2 text-xs text-danger-500">
          {error}
        </p>
      ) : null}
      <div className="relative z-10 mt-3 flex flex-wrap gap-2">
        {canUpdate && next ? (
          <Button
            className="min-h-11"
            disabled={patch.isPending}
            onClick={() => void run(next)}
          >
            {next === "IN_PROGRESS" ? "Mulai dikerjakan" : "Tandai selesai"}
          </Button>
        ) : null}
        {canVerifyCA(role) && item.status === "DONE" && !closed ? (
          <Button
            variant="ghost"
            className="min-h-11"
            disabled={verify.isPending}
            onClick={async () => {
              setError(null);
              try {
                await verify.mutateAsync({ id: item.id });
              } catch (cause) {
                setError(mapApiError(cause, "Gagal memverifikasi."));
              }
            }}
          >
            Verifikasi
          </Button>
        ) : null}
      </div>
      {canUploadEvidence && item.status !== "OPEN" ? (
        <div className="mt-3">
          <p className="mb-2 text-xs text-subtle">Bukti penyelesaian</p>
          <FileUploader
            files={evidence}
            canUpload={canUpdate && item.status !== "VERIFIED"}
            uploading={upload.isPending}
            onUpload={async (picked) => {
              await upload.mutateAsync({
                id: item.incidentId,
                files: picked,
                correctiveActionId: item.id,
              });
            }}
            onRefreshFiles={async () => (await onRefreshFiles())?.filter((file) => file.correctiveActionId === item.id)}
          />
        </div>
      ) : evidence.length > 0 ? (
        <FileUploader files={evidence} canUpload={false} onUpload={async () => undefined} onRefreshFiles={onRefreshFiles} />
      ) : null}
    </article>
  );
}
