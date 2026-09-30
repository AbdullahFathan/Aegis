"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { EmptyState, LoadingBlock, PageHeader, QueryError } from "@/components/shared/EmptyState";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api, ApiError, triggerBrowserDownload } from "@/lib/api/client";
import { formatDateTime, isoToDateInput } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import { getFileUrl } from "@/lib/files";
import { useLocations } from "@/lib/queries/useAdmin";
import { requestReport, useReportArchive, useReportJob } from "@/lib/queries/useReports";
import { useSession } from "@/lib/queries/useSession";
import { canExportReports } from "@/lib/rbac";
import {
  INCIDENT_CATEGORIES,
  REPORT_FORMATS,
  REPORT_TYPES,
  categoryLabels,
  parseReportsSearchParams,
  reportTypeLabels,
  type ReportsFilterValues,
} from "@/lib/schemas";
import type { ReportJob } from "@/lib/types";
import { cn } from "@/lib/utils";

function filtersToParams(filters: ReportsFilterValues) {
  const params = new URLSearchParams();
  params.set("type", filters.type);
  params.set("format", filters.format);
  params.set("tab", filters.tab);
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  if (filters.incidentId) params.set("incidentId", filters.incidentId);
  return params;
}

async function openSigned(url: string) {
  const href = await getFileUrl(url, async () => url);
  window.open(href, "_blank", "noopener,noreferrer");
}

export function ReportsScreen() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const session = useSession();
  const allowed = canExportReports(session.data?.role ?? "");
  const locations = useLocations();
  const filters = useMemo(
    () => parseReportsSearchParams(Object.fromEntries(searchParams.entries())),
    [searchParams],
  );
  const archive = useReportArchive(allowed && filters.tab === "archive");
  const [jobId, setJobId] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const openedJob = useRef<string | null>(null);
  const job = useReportJob(jobId);

  const replaceFilters = useCallback(
    (patch: Partial<ReportsFilterValues>) => {
      const next = { ...filters, ...patch };
      router.replace(`${pathname}?${filtersToParams(next).toString()}`);
    },
    [filters, pathname, router],
  );

  useEffect(() => {
    if (!jobId || job.data?.status !== "DONE" || !job.data.downloadUrl) return;
    if (openedJob.current === jobId) return;
    openedJob.current = jobId;
    void openSigned(job.data.downloadUrl);
  }, [job.data, jobId]);

  const jobMessage =
    job.data?.status === "FAILED"
      ? job.data.error || "Generate PDF gagal."
      : job.data?.status === "DONE"
        ? "PDF siap diunduh."
        : jobId
          ? "PDF sedang digenerate. Tunggu hingga selesai."
          : message;

  const sites = locations.data ?? [];

  async function download() {
    setMessage(null);
    if (filters.type === "investigation" && !filters.incidentId?.trim()) {
      setMessage("ID insiden wajib diisi untuk laporan investigasi.");
      return;
    }
    setBusy(true);
    openedJob.current = null;
    setJobId(undefined);
    try {
      const result = await requestReport(filters);
      if (result.kind === "file") {
        triggerBrowserDownload(result.blob, result.filename);
        setMessage("Unduhan CSV dimulai.");
        return;
      }
      const id = result.data.jobId ?? result.data.id;
      if (!id) {
        setMessage("Server tidak mengembalikan job PDF.");
        return;
      }
      setJobId(id);
      setMessage("PDF sedang digenerate. Tunggu hingga selesai.");
    } catch (error) {
      setMessage(mapApiError(error, "Gagal generate laporan."));
    } finally {
      setBusy(false);
    }
  }

  if (!allowed && session.data) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader title="Laporan" />
        <QueryError message="Anda tidak memiliki akses ke halaman ini." />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Laporan" />
      <div className="flex gap-2">
        <button
          type="button"
          className={cn(
            buttonVariants({ variant: filters.tab === "generate" ? "default" : "secondary" }),
            "min-h-11",
          )}
          onClick={() => replaceFilters({ tab: "generate" })}
        >
          Generate
        </button>
        <button
          type="button"
          className={cn(
            buttonVariants({ variant: filters.tab === "archive" ? "default" : "secondary" }),
            "min-h-11",
          )}
          onClick={() => replaceFilters({ tab: "archive" })}
        >
          Arsip
        </button>
      </div>

      {filters.tab === "generate" ? (
        <div className="flex flex-col gap-4 rounded-lg border border-border bg-white p-4 shadow-card">
          <Select
            value={filters.type}
            onValueChange={(value) =>
              replaceFilters({ type: (value as ReportsFilterValues["type"]) || "monthly" })
            }
          >
            <SelectTrigger className="h-9 w-full rounded-md md:w-72">
              <SelectValue placeholder="Tipe laporan" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {REPORT_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>
                    {reportTypeLabels[type]}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Select
            value={filters.format}
            onValueChange={(value) => replaceFilters({ format: value === "csv" ? "csv" : "pdf" })}
          >
            <SelectTrigger className="h-9 w-full rounded-md md:w-40">
              <SelectValue placeholder="Format" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {REPORT_FORMATS.map((format) => (
                  <SelectItem key={format} value={format}>
                    {format.toUpperCase()}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Select
            value={filters.category ?? "ALL"}
            onValueChange={(value) =>
              replaceFilters({
                category:
                  !value || value === "ALL" ? undefined : (value as ReportsFilterValues["category"]),
              })
            }
          >
            <SelectTrigger className="h-9 w-full rounded-md md:w-52">
              <SelectValue placeholder="Semua kategori" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="ALL">Semua kategori</SelectItem>
                {INCIDENT_CATEGORIES.map((category) => (
                  <SelectItem key={category} value={category}>
                    {categoryLabels[category]}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Select
            value={filters.locationId ?? "ALL"}
            onValueChange={(value) =>
              replaceFilters({ locationId: !value || value === "ALL" ? undefined : value })
            }
          >
            <SelectTrigger className="h-9 w-full rounded-md md:w-52">
              <SelectValue placeholder="Semua lokasi" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="ALL">Semua lokasi</SelectItem>
                {sites.map((site) => (
                  <SelectItem key={site.id} value={site.id}>
                    {site.code} — {site.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <div className="flex flex-col gap-4 md:flex-row">
            <Input
              type="date"
              className="h-9 rounded-md md:w-40"
              value={isoToDateInput(filters.from)}
              onChange={(event) => replaceFilters({ from: event.target.value || undefined })}
              aria-label="Dari tanggal"
            />
            <Input
              type="date"
              className="h-9 rounded-md md:w-40"
              value={isoToDateInput(filters.to)}
              onChange={(event) => replaceFilters({ to: event.target.value || undefined })}
              aria-label="Sampai tanggal"
            />
          </div>
          {filters.type === "investigation" ? (
            <Input
              className="h-9 rounded-md md:w-80"
              placeholder="ID insiden"
              value={filters.incidentId ?? ""}
              onChange={(event) => replaceFilters({ incidentId: event.target.value || undefined })}
            />
          ) : null}

          <div className="rounded-md border border-border bg-canvas p-3 text-sm text-ink">
            <p className="font-medium">Preview</p>
            <p className="mt-1 text-subtle">
              {reportTypeLabels[filters.type]} · {filters.format.toUpperCase()}
              {filters.from || filters.to
                ? ` · ${filters.from ?? "…"} s.d. ${filters.to ?? "…"}`
                : " · periode default API"}
            </p>
          </div>

          <Button className="min-h-11 w-fit" disabled={busy || Boolean(jobId && job.data?.status === "PENDING")} onClick={() => void download()}>
            {busy || (jobId && job.data?.status === "PENDING") ? "Memproses…" : "Unduh"}
          </Button>
          {jobMessage ? <p className="text-sm text-subtle">{jobMessage}</p> : null}
          {job.isError ? (
            <QueryError
              message={
                job.error instanceof ApiError
                  ? mapApiError(job.error, "Gagal memuat status job.")
                  : "Gagal memuat status job."
              }
            />
          ) : null}
        </div>
      ) : archive.isPending ? (
        <LoadingBlock />
      ) : archive.isError ? (
        <QueryError message={mapApiError(archive.error, "Gagal memuat arsip laporan.")} />
      ) : !archive.data?.length ? (
        <EmptyState title="Arsip kosong" description="Belum ada laporan yang digenerate." />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border bg-white">
          <table className="w-full text-sm">
            <thead className="bg-canvas text-xs font-medium tracking-wide text-subtle uppercase">
              <tr>
                <th className="px-4 py-2 text-left">Tipe</th>
                <th className="px-4 py-2 text-left">Status</th>
                <th className="px-4 py-2 text-left">Dibuat</th>
                <th className="px-4 py-2 text-left">File</th>
                <th className="px-4 py-2 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {archive.data.map((row) => (
                <tr key={row.id} className="border-t border-border">
                  <td className="px-4 py-3 font-mono text-xs">{row.type}</td>
                  <td className="px-4 py-3">{row.status}</td>
                  <td className="px-4 py-3 font-mono text-xs">{formatDateTime(row.createdAt)}</td>
                  <td className="px-4 py-3 font-mono text-xs text-subtle">{row.storedKey ?? "—"}</td>
                  <td className="px-4 py-3 text-right">
                    {row.status === "DONE" ? (
                      <ArchiveDownload id={row.id} />
                    ) : (
                      <span className="text-xs text-subtle">{row.errorMessage ?? "Belum siap"}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function ArchiveDownload({ id }: { id: string }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    setPending(true);
    setError(null);
    try {
      const row = await api<ReportJob>(`/reports/jobs/${id}`);
      if (!row.downloadUrl) {
        setError("URL unduhan tidak tersedia.");
        return;
      }
      await openSigned(row.downloadUrl);
    } catch (err) {
      setError(mapApiError(err, "Gagal mengunduh arsip."));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <Button variant="ghost" size="sm" disabled={pending} onClick={() => void handleClick()}>
        {pending ? "Memuat…" : "Unduh"}
      </Button>
      {error ? <span className="text-xs text-danger-500">{error}</span> : null}
    </div>
  );
}
