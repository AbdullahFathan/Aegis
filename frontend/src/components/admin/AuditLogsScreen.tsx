"use client";

import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from "@tanstack/react-table";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo, useState } from "react";

import { EmptyState, LoadingBlock, PageHeader, QueryError } from "@/components/shared/EmptyState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, triggerBrowserDownload } from "@/lib/api/client";
import { formatDateTime, isoToDateInput } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import { buildAuditQuery, useAuditLogs } from "@/lib/queries/useAuditLogs";
import { requestAuditCsv } from "@/lib/queries/useReports";
import { useSession } from "@/lib/queries/useSession";
import { canReadAuditLogs } from "@/lib/rbac";
import {
  AUDIT_ACTIONS,
  auditActionLabels,
  parseAuditSearchParams,
  type AuditFilterValues,
} from "@/lib/schemas";
import type { AuditLog } from "@/lib/types";

function filtersToParams(filters: AuditFilterValues) {
  const params = new URLSearchParams();
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  if (filters.userId) params.set("userId", filters.userId);
  if (filters.entityType) params.set("entityType", filters.entityType);
  if (filters.action) params.set("action", filters.action);
  if (filters.page > 1) params.set("page", String(filters.page));
  if (filters.pageSize !== 50) params.set("pageSize", String(filters.pageSize));
  return params;
}

export function AuditLogsScreen() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const session = useSession();
  const allowed = canReadAuditLogs(session.data?.role ?? "");
  const filters = useMemo(
    () => parseAuditSearchParams(Object.fromEntries(searchParams.entries())),
    [searchParams],
  );
  const logs = useAuditLogs(filters, allowed);
  const [exportError, setExportError] = useState<string | null>(null);

  const replaceFilters = useCallback(
    (patch: Partial<AuditFilterValues>) => {
      const next = { ...filters, ...patch };
      if (
        patch.page === undefined &&
        (patch.from !== undefined ||
          patch.to !== undefined ||
          patch.userId !== undefined ||
          patch.entityType !== undefined ||
          patch.action !== undefined)
      ) {
        next.page = 1;
      }
      const query = filtersToParams(next).toString();
      router.replace(query ? `${pathname}?${query}` : pathname);
    },
    [filters, pathname, router],
  );

  const columns = useMemo<ColumnDef<AuditLog>[]>(
    () => [
      {
        accessorKey: "createdAt",
        header: "Waktu",
        cell: ({ row }) => (
          <span className="font-mono text-xs">{formatDateTime(row.original.createdAt)}</span>
        ),
      },
      { accessorKey: "userRole", header: "Role" },
      { accessorKey: "action", header: "Aksi" },
      { accessorKey: "entityType", header: "Entitas" },
      {
        accessorKey: "entityId",
        header: "ID",
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.entityId}</span>,
      },
      { accessorKey: "ipAddress", header: "IP" },
    ],
    [],
  );

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data: logs.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const forbidden = logs.error instanceof ApiError && logs.error.status === 403;
  const total = logs.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / filters.pageSize));

  async function exportCsv() {
    setExportError(null);
    try {
      const result = await requestAuditCsv(buildAuditQuery(filters, "csv"));
      if (result.kind === "file") {
        triggerBrowserDownload(result.blob, result.filename);
        return;
      }
      setExportError("Server tidak mengembalikan CSV.");
    } catch (error) {
      setExportError(mapApiError(error, "Gagal mengekspor audit log."));
    }
  }

  if (!allowed && session.data) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader title="Audit log" />
        <QueryError message="Anda tidak memiliki akses ke halaman ini." />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Audit log"
        actions={
          <Button variant="secondary" className="min-h-11" onClick={() => void exportCsv()}>
            Export CSV
          </Button>
        }
      />
      <p className="text-xs text-subtle">Catatan bersifat tetap — tidak dapat diubah atau dihapus.</p>
      <div className="flex flex-col gap-4 md:flex-row md:flex-wrap">
        <div className="flex min-w-0 flex-col gap-1">
          <label className="text-sm font-medium text-ink" htmlFor="audit-from">
            Dari tanggal
          </label>
          <Input
            id="audit-from"
            type="date"
            className="h-9 rounded-md md:w-40"
            value={isoToDateInput(filters.from)}
            onChange={(event) => replaceFilters({ from: event.target.value || undefined })}
          />
        </div>
        <div className="flex min-w-0 flex-col gap-1">
          <label className="text-sm font-medium text-ink" htmlFor="audit-to">
            Sampai tanggal
          </label>
          <Input
            id="audit-to"
            type="date"
            className="h-9 rounded-md md:w-40"
            value={isoToDateInput(filters.to)}
            onChange={(event) => replaceFilters({ to: event.target.value || undefined })}
          />
        </div>
        <div className="flex min-w-0 flex-col gap-1">
          <label className="text-sm font-medium text-ink" htmlFor="audit-user">
            User ID
          </label>
          <Input
            id="audit-user"
            className="h-9 rounded-md md:w-56"
            value={filters.userId ?? ""}
            onChange={(event) => replaceFilters({ userId: event.target.value || undefined })}
          />
        </div>
        <div className="flex min-w-0 flex-col gap-1">
          <label className="text-sm font-medium text-ink" htmlFor="audit-entity">
            Tipe entitas
          </label>
          <Input
            id="audit-entity"
            className="h-9 rounded-md md:w-40"
            value={filters.entityType ?? ""}
            onChange={(event) => replaceFilters({ entityType: event.target.value || undefined })}
          />
        </div>
        <Select
          value={filters.action ?? "ALL"}
          onValueChange={(value) =>
            replaceFilters({
              action: !value || value === "ALL" ? undefined : (value as AuditFilterValues["action"]),
            })
          }
        >
          <SelectTrigger className="h-9 w-full rounded-md md:w-52">
            <SelectValue placeholder="Semua aksi" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="ALL">Semua aksi</SelectItem>
              {AUDIT_ACTIONS.map((action) => (
                <SelectItem key={action} value={action}>
                  {auditActionLabels[action]}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
      {exportError ? <QueryError message={exportError} /> : null}
      {logs.isPending ? <LoadingBlock /> : null}
      {logs.isError ? (
        <QueryError
          message={
            forbidden
              ? "Anda tidak memiliki akses ke halaman ini."
              : mapApiError(logs.error, "Gagal memuat audit log.")
          }
        />
      ) : null}
      {logs.data && logs.data.items.length === 0 ? (
        <EmptyState
          title="Tidak ada entri"
          description="Tidak ada audit log sesuai filter. Ubah rentang tanggal atau kriteria lain."
        />
      ) : null}
      {logs.data && logs.data.items.length > 0 ? (
        <div className="overflow-hidden rounded-lg border border-border bg-white">
          <Table>
            <TableHeader className="bg-canvas">
              {table.getHeaderGroups().map((group) => (
                <TableRow key={group.id} className="hover:bg-transparent">
                  {group.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className="h-10 px-4 text-xs font-medium tracking-wide text-subtle uppercase"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} className="h-12">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="px-4 text-sm">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
      {logs.data ? (
        <div className="flex items-center justify-between text-sm text-subtle">
          <span>
            Halaman {filters.page} dari {pageCount}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              className="min-h-11"
              disabled={filters.page <= 1}
              onClick={() => replaceFilters({ page: filters.page - 1 })}
            >
              Sebelumnya
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="min-h-11"
              disabled={filters.page >= pageCount}
              onClick={() => replaceFilters({ page: filters.page + 1 })}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
