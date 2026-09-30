"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";

import { IncidentTable } from "@/components/incidents/IncidentTable";
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
import { isoToDateInput } from "@/lib/datetime";
import { mapIncidentError } from "@/lib/errors";
import { useLocations } from "@/lib/queries/useAdmin";
import { useIncidents } from "@/lib/queries/useIncidents";
import { useSession } from "@/lib/queries/useSession";
import {
  INCIDENT_CATEGORIES,
  INCIDENT_STATUSES,
  categoryLabels,
  parseIncidentSearchParams,
} from "@/lib/schemas";
import { statusConfig } from "@/lib/tokens";
import type { Incident, IncidentFilters } from "@/lib/types";
import { ApiError } from "@/lib/api/client";
import { cn } from "@/lib/utils";

function filtersToParams(filters: IncidentFilters) {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  if (filters.page > 1) params.set("page", String(filters.page));
  if (filters.pageSize !== 20) params.set("pageSize", String(filters.pageSize));
  return params;
}

export function IncidentsScreen() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const session = useSession();
  const locations = useLocations();
  const filters = useMemo(
    () => parseIncidentSearchParams(Object.fromEntries(searchParams.entries())),
    [searchParams],
  );
  const incidents = useIncidents(filters);

  const replaceFilters = useCallback(
    (patch: Partial<IncidentFilters>) => {
      const next: IncidentFilters = { ...filters, ...patch };
      if (
        patch.page === undefined &&
        (patch.status !== undefined ||
          patch.category !== undefined ||
          patch.locationId !== undefined ||
          patch.from !== undefined ||
          patch.to !== undefined)
      ) {
        next.page = 1;
      }
      const query = filtersToParams(next).toString();
      router.replace(query ? `${pathname}?${query}` : pathname);
    },
    [filters, pathname, router],
  );

  const locationName = useCallback(
    (locationId: string) => {
      const site = locations.data?.find((item) => item.id === locationId);
      return site ? `${site.code} — ${site.name}` : locationId;
    },
    [locations.data],
  );

  const picName = useCallback(
    (incident: Incident) => {
      if (session.data?.id === incident.reporterId) return session.data.name;
      const site = locations.data?.find((item) => item.id === incident.locationId);
      return site ? `Supervisor ${site.code}` : "PIC lokasi";
    },
    [locations.data, session.data],
  );

  const forbidden = incidents.error instanceof ApiError && incidents.error.status === 403;
  const total = incidents.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / filters.pageSize));
  const sites = locations.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Insiden"
        actions={
          <Link href="/incidents/new" className={cn(buttonVariants(), "min-h-11")}>
            Buat laporan
          </Link>
        }
      />

      <div className="flex flex-col gap-4 md:flex-row md:flex-wrap">
        <Select
          value={filters.status ?? "ALL"}
          onValueChange={(value) =>
            replaceFilters({
              status: !value || value === "ALL" ? undefined : (value as IncidentFilters["status"]),
            })
          }
        >
          <SelectTrigger className="h-9 w-full rounded-md md:w-52">
            <SelectValue placeholder="Semua status" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="ALL">Semua status</SelectItem>
              {INCIDENT_STATUSES.map((status) => (
                <SelectItem key={status} value={status}>
                  {statusConfig[status].label}
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
                !value || value === "ALL" ? undefined : (value as IncidentFilters["category"]),
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

      {incidents.isPending ? <LoadingBlock /> : null}
      {incidents.isError ? (
        <QueryError
          message={
            forbidden
              ? "Anda tidak memiliki akses ke halaman ini."
              : mapIncidentError(incidents.error, "Gagal memuat daftar insiden.")
          }
        />
      ) : null}
      {incidents.data && incidents.data.items.length === 0 ? (
        <EmptyState
          title="Belum ada laporan"
          description="Tidak ada insiden sesuai filter. Buat laporan baru dari lapangan."
          action={
            <Link href="/incidents/new" className={cn(buttonVariants(), "min-h-11")}>
              Buat laporan
            </Link>
          }
        />
      ) : null}
      {incidents.data && incidents.data.items.length > 0 ? (
        <IncidentTable items={incidents.data.items} locationName={locationName} picName={picName} />
      ) : null}
      {incidents.data ? (
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
