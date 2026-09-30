"use client";

import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";

import { CaStatusBadge } from "@/components/shared/StatusBadge";
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
import { ApiError } from "@/lib/api/client";
import { formatDate, isoToDateInput } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import { useLocations } from "@/lib/queries/useAdmin";
import { useCorrectiveActions } from "@/lib/queries/useCorrectiveActions";
import { useSession } from "@/lib/queries/useSession";
import {
  CA_PRIORITIES,
  CA_STATUSES,
  caPriorityLabels,
  parseCaTrackerSearchParams,
} from "@/lib/schemas";
import { caStatusConfig } from "@/lib/tokens";
import type { CaTrackerFilters, CorrectiveAction } from "@/lib/types";
import { caUrgency, urgencyRowClass } from "@/lib/urgency";
import { cn } from "@/lib/utils";

function filtersToParams(filters: CaTrackerFilters) {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.assigneeId) params.set("assigneeId", filters.assigneeId);
  if (filters.priority) params.set("priority", filters.priority);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.dueFrom) params.set("dueFrom", filters.dueFrom);
  if (filters.dueTo) params.set("dueTo", filters.dueTo);
  if (filters.view === "kanban") params.set("view", "kanban");
  return params;
}

function inDueRange(item: CorrectiveAction, from?: string, to?: string) {
  const day = isoToDateInput(item.dueDate);
  if (from && day < from) return false;
  if (to && day > to) return false;
  return true;
}

export function CATrackerScreen() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const session = useSession();
  const locations = useLocations();
  const filters = useMemo(
    () => parseCaTrackerSearchParams(Object.fromEntries(searchParams.entries())),
    [searchParams],
  );
  const query = useCorrectiveActions({
    status: filters.status,
    assigneeId: filters.assigneeId,
    priority: filters.priority,
    locationId: filters.locationId,
  });

  const replaceFilters = useCallback(
    (patch: Partial<CaTrackerFilters>) => {
      const next = { ...filters, ...patch };
      const qs = filtersToParams(next).toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname);
    },
    [filters, pathname, router],
  );

  const items = useMemo(() => {
    return (query.data ?? []).filter((item) => inDueRange(item, filters.dueFrom, filters.dueTo));
  }, [query.data, filters.dueFrom, filters.dueTo]);

  const columns = useMemo<ColumnDef<CorrectiveAction>[]>(
    () => [
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <CaStatusBadge status={row.original.status} />,
      },
      { accessorKey: "description", header: "Tindakan" },
      {
        accessorKey: "priority",
        header: "Prioritas",
        cell: ({ row }) => caPriorityLabels[row.original.priority],
      },
      {
        accessorKey: "dueDate",
        header: "Tenggat",
        cell: ({ row }) => formatDate(row.original.dueDate),
      },
    ],
    [],
  );

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const forbidden = query.error instanceof ApiError && query.error.status === 403;
  const sites = locations.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="CA Tracker"
        actions={
          <div className="flex gap-2">
            <Button
              variant={filters.view === "table" ? "default" : "secondary"}
              className="min-h-11"
              onClick={() => replaceFilters({ view: "table" })}
            >
              Tabel
            </Button>
            <Button
              variant={filters.view === "kanban" ? "default" : "secondary"}
              className="min-h-11"
              onClick={() => replaceFilters({ view: "kanban" })}
            >
              Kanban
            </Button>
          </div>
        }
      />
      <div className="flex flex-col gap-4 md:flex-row md:flex-wrap">
        <Select
          value={filters.status ?? "ALL"}
          onValueChange={(value) =>
            replaceFilters({ status: !value || value === "ALL" ? undefined : (value as CaTrackerFilters["status"]) })
          }
        >
          <SelectTrigger className="h-9 w-full rounded-md md:w-52">
            <SelectValue placeholder="Semua status" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="ALL">Semua status</SelectItem>
              {CA_STATUSES.map((status) => (
                <SelectItem key={status} value={status}>
                  {caStatusConfig[status].label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Select
          value={filters.priority ?? "ALL"}
          onValueChange={(value) =>
            replaceFilters({
              priority: !value || value === "ALL" ? undefined : (value as CaTrackerFilters["priority"]),
            })
          }
        >
          <SelectTrigger className="h-9 w-full rounded-md md:w-52">
            <SelectValue placeholder="Semua prioritas" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="ALL">Semua prioritas</SelectItem>
              {CA_PRIORITIES.map((priority) => (
                <SelectItem key={priority} value={priority}>
                  {caPriorityLabels[priority]}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Select
          value={filters.locationId ?? "ALL"}
          onValueChange={(value) => replaceFilters({ locationId: !value || value === "ALL" ? undefined : value })}
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
        {session.data ? (
          <Select
            value={filters.assigneeId ?? "ALL"}
            onValueChange={(value) => replaceFilters({ assigneeId: !value || value === "ALL" ? undefined : value })}
          >
            <SelectTrigger className="h-9 w-full rounded-md md:w-52">
              <SelectValue placeholder="Semua assignee" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="ALL">Semua assignee</SelectItem>
                <SelectItem value={session.data.id}>Saya</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        ) : null}
        <Input
          type="date"
          className="h-9 rounded-md md:w-40"
          value={isoToDateInput(filters.dueFrom)}
          onChange={(event) => replaceFilters({ dueFrom: event.target.value || undefined })}
          aria-label="Tenggat dari"
        />
        <Input
          type="date"
          className="h-9 rounded-md md:w-40"
          value={isoToDateInput(filters.dueTo)}
          onChange={(event) => replaceFilters({ dueTo: event.target.value || undefined })}
          aria-label="Tenggat sampai"
        />
      </div>

      {query.isPending ? <LoadingBlock /> : null}
      {query.isError ? (
        <QueryError
          message={forbidden ? "Anda tidak memiliki akses ke halaman ini." : mapApiError(query.error, "Gagal memuat CA.")}
        />
      ) : null}
      {!query.isPending && items.length === 0 ? (
        <EmptyState title="Tidak ada CA sesuai filter" description="Ubah filter atau buat tindakan dari laporan insiden." />
      ) : null}

      {filters.view !== "kanban" && items.length > 0 ? (
        <div className="hidden overflow-hidden rounded-lg border border-border bg-white md:block">
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
                <TableRow
                  key={row.id}
                  className={cn("h-12 cursor-pointer hover:bg-canvas", urgencyRowClass(caUrgency(row.original)))}
                  onClick={() => router.push(`/incidents/${row.original.incidentId}?tab=ca`)}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="px-4 text-sm text-ink">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}

      {filters.view !== "kanban" && items.length > 0 ? (
        <div className="flex flex-col gap-3 md:hidden">
          {items.map((item) => (
            <Link
              key={item.id}
              href={`/incidents/${item.incidentId}?tab=ca`}
              className={cn("block rounded-lg border border-border bg-white p-4 shadow-card", urgencyRowClass(caUrgency(item)))}
            >
              <div className="flex justify-between gap-2">
                <p className="text-sm text-ink">{item.description}</p>
                <CaStatusBadge status={item.status} />
              </div>
              <p className="mt-2 text-xs text-subtle">Tenggat {formatDate(item.dueDate)}</p>
            </Link>
          ))}
        </div>
      ) : null}

      {filters.view === "kanban" && items.length > 0 ? (
        <div className="grid gap-4 lg:grid-cols-5">
          {CA_STATUSES.map((status) => {
            const column = items.filter((item) => item.status === status);
            return (
              <section key={status} className="rounded-lg border border-border bg-white p-3">
                <h2 className="text-sm font-medium text-ink">{caStatusConfig[status].label}</h2>
                <div className="mt-3 flex flex-col gap-2">
                  {column.length === 0 ? (
                    <p className="text-xs text-subtle">Kosong</p>
                  ) : (
                    column.map((item) => (
                      <Link
                        key={item.id}
                        href={`/incidents/${item.incidentId}?tab=ca`}
                        className={cn("rounded-md border border-border p-3 text-sm text-ink", urgencyRowClass(caUrgency(item)))}
                      >
                        {item.description}
                        <span className="mt-1 block text-xs text-subtle">{formatDate(item.dueDate)}</span>
                      </Link>
                    ))
                  )}
                </div>
              </section>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
