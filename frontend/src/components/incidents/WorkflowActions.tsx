"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { mapIncidentError } from "@/lib/errors";
import {
  useCloseIncident,
  useRejectIncident,
  useStartCorrectiveAction,
  useVerifyIncident,
} from "@/lib/queries/useIncidents";
import { workflowActionsFor, type WorkflowAction } from "@/lib/rbac";
import type { Incident, IncidentStatus } from "@/lib/types";

export function WorkflowActions({
  incident,
  role,
}: {
  incident: Incident;
  role: string;
}) {
  const actions = workflowActionsFor(role, incident.status as IncidentStatus);
  const verify = useVerifyIncident();
  const reject = useRejectIncident();
  const close = useCloseIncident();
  const startCa = useStartCorrectiveAction();
  const [rejectOpen, setRejectOpen] = useState(false);
  const [confirm, setConfirm] = useState<Exclude<WorkflowAction, "reject"> | null>(null);
  const [comment, setComment] = useState("");
  const [error, setError] = useState<string | null>(null);

  if (actions.length === 0) return null;

  const pending = verify.isPending || reject.isPending || close.isPending || startCa.isPending;

  async function run(action: Exclude<WorkflowAction, "reject">, note?: string) {
    setError(null);
    try {
      if (action === "verify") await verify.mutateAsync({ id: incident.id, comment: note });
      if (action === "close") await close.mutateAsync({ id: incident.id, comment: note });
      if (action === "startCA") await startCa.mutateAsync({ id: incident.id, comment: note });
      setConfirm(null);
      setComment("");
    } catch (cause) {
      setError(mapIncidentError(cause, "Aksi gagal dijalankan."));
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        {actions.includes("verify") ? (
          <Button className="min-h-11" disabled={pending} onClick={() => setConfirm("verify")}>
            Verifikasi
          </Button>
        ) : null}
        {actions.includes("reject") ? (
          <Button
            variant="destructive"
            className="min-h-11"
            disabled={pending}
            onClick={() => setRejectOpen(true)}
          >
            Kembalikan
          </Button>
        ) : null}
        {actions.includes("startCA") ? (
          <Button
            variant="secondary"
            className="min-h-11"
            disabled={pending}
            onClick={() => setConfirm("startCA")}
          >
            Mulai tindakan perbaikan
          </Button>
        ) : null}
        {actions.includes("close") ? (
          <Button
            variant="destructive"
            className="min-h-11"
            disabled={pending}
            onClick={() => setConfirm("close")}
          >
            Tutup laporan
          </Button>
        ) : null}
      </div>
      {error ? (
        <p role="alert" className="text-xs text-danger-500">
          {error}
        </p>
      ) : null}

      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Kembalikan laporan</DialogTitle>
            <DialogDescription>Catatan wajib agar pelapor tahu apa yang perlu diperbaiki.</DialogDescription>
          </DialogHeader>
          <Textarea
            className="min-h-24 rounded-md"
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            placeholder="Alasan pengembalian"
          />
          <DialogFooter className="border-border bg-white">
            <Button
              variant="destructive"
              className="min-h-11"
              disabled={pending || !comment.trim()}
              onClick={async () => {
                setError(null);
                try {
                  await reject.mutateAsync({ id: incident.id, comment: comment.trim() });
                  setRejectOpen(false);
                  setComment("");
                } catch (cause) {
                  setError(mapIncidentError(cause, "Gagal mengembalikan laporan."));
                }
              }}
            >
              Kembalikan
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={confirm !== null} onOpenChange={(open) => !open && setConfirm(null)}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {confirm === "verify"
                ? "Verifikasi laporan?"
                : confirm === "close"
                  ? "Tutup laporan?"
                  : "Pindah ke tindakan perbaikan?"}
            </DialogTitle>
            <DialogDescription>
              {confirm === "close"
                ? "Laporan tertutup bersifat final. Semua corrective action harus sudah terverifikasi."
                : "Status hanya berubah lewat aksi ini, bukan ubahan langsung."}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            className="min-h-20 rounded-md"
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            placeholder="Catatan (opsional)"
          />
          <DialogFooter className="border-border bg-white">
            <Button
              className="min-h-11"
              disabled={pending || !confirm}
              onClick={() => confirm && void run(confirm, comment.trim() || undefined)}
            >
              Konfirmasi
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
