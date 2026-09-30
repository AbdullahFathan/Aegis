"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";

import { CategoryDonut } from "@/components/dashboard/CategoryDonut";
import { HeatmapTable } from "@/components/dashboard/HeatmapTable";
import { KPICard } from "@/components/dashboard/KPICard";
import { PipelineChart } from "@/components/dashboard/PipelineChart";
import { TrendChart } from "@/components/dashboard/TrendChart";
import { IncidentCard } from "@/components/incidents/IncidentCard";
import { EmptyState, LoadingBlock, PageHeader, QueryError } from "@/components/shared/EmptyState";
import { CaStatusBadge } from "@/components/shared/StatusBadge";
import { buttonVariants } from "@/components/ui/button";
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
  casDueWithinDays,
  isStaleQueueIncident,
  mapDonutFromHeatmap,
  mapHeatmapMatrix,
  mapPipelineBuckets,
  mapTrendSeries,
  nearMissRateFromHeatmap,
} from "@/lib/dashboard";
import { isoToDateInput } from "@/lib/datetime";
import { mapApiError } from "@/lib/errors";
import { formatKpiNumber, formatPercent, formatRate } from "@/lib/format";
import { ApiError } from "@/lib/api/client";
import { useLocations } from "@/lib/queries/useAdmin";
import { useCorrectiveActions } from "@/lib/queries/useCorrectiveActions";
import {
  useDashboardHeatmap,
  useDashboardSummary,
  useDashboardTrends,
} from "@/lib/queries/useDashboard";
import { useIncidents } from "@/lib/queries/useIncidents";
import { useSession } from "@/lib/queries/useSession";
import { canReadDashboardApi, dashboardWidgetsFor } from "@/lib/rbac";
import {
  INCIDENT_CATEGORIES,
  categoryLabels,
  parseDashboardSearchParams,
  type DashboardFilterValues,
} from "@/lib/schemas";
import type { Incident } from "@/lib/types";
import { cn } from "@/lib/utils";
import { caUrgency, urgencyRowClass } from "@/lib/urgency";

function filtersToParams(filters: DashboardFilterValues) {
  const params = new URLSearchParams();
  if (filters.category) params.set("category", filters.category);
  if (filters.locationId) params.set("locationId", filters.locationId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  return params;
}

export function DashboardScreen() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const session = useSession();
  const role = session.data?.role ?? "";
  const widgets = dashboardWidgetsFor(role);
  const canCharts = canReadDashboardApi(role) && widgets.includes("charts");
  const locations = useLocations();
  const filters = useMemo(
    () => parseDashboardSearchParams(Object.fromEntries(searchParams.entries())),
    [searchParams],
  );

  const summary = useDashboardSummary(filters, canCharts);
  const trends = useDashboardTrends(filters, canCharts);
  const heatmap = useDashboardHeatmap(filters, canCharts);

  const needIncidentQueue =
    widgets.includes("officerQueue") ||
    widgets.includes("supervisorQueue") ||
    widgets.includes("reporterBrief");
  const officerIncidents = useIncidents({ page: 1, pageSize: 50 }, { enabled: needIncidentQueue });
  const officerCas = useCorrectiveActions({}, { enabled: widgets.includes("officerQueue") });
  const reporterCas = useCorrectiveActions(
    session.data?.id ? { assigneeId: session.data.id } : {},
    { enabled: widgets.includes("reporterBrief") && Boolean(session.data?.id) },
  );

  const replaceFilters = useCallback(
    (patch: Partial<DashboardFilterValues>) => {
      const next = { ...filters, ...patch };
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

  const donut = useMemo(
    () => mapDonutFromHeatmap(heatmap.data ?? [], categoryLabels),
    [heatmap.data],
  );
  const nearMiss = useMemo(() => nearMissRateFromHeatmap(heatmap.data ?? []), [heatmap.data]);
  const matrix = useMemo(() => mapHeatmapMatrix(heatmap.data ?? []), [heatmap.data]);
  const pipeline = useMemo(() => mapPipelineBuckets(summary.data?.pipeline), [summary.data]);
  const trendSeries = useMemo(() => mapTrendSeries(trends.data ?? []), [trends.data]);

  const queueItems = officerIncidents.data?.items ?? [];
  const officerQueue = queueItems.filter(
    (item) => item.status === "PENDING_REVIEW" || item.status === "UNDER_INVESTIGATION",
  );
  const supervisorQueue = queueItems.filter((item) => item.status === "PENDING_REVIEW");
  const dueCas = casDueWithinDays(officerCas.data ?? []);
  const myCas = reporterCas.data ?? [];

  const chartError = summary.error || trends.error || heatmap.error;
  const forbidden = chartError instanceof ApiError && chartError.status === 403;
  const sites = locations.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Dashboard" />

      {canCharts ? (
        <div className="flex flex-col gap-4 md:flex-row md:flex-wrap">
          <Select
            value={filters.category ?? "ALL"}
            onValueChange={(value) =>
              replaceFilters({
                category: !value || value === "ALL" ? undefined : (value as DashboardFilterValues["category"]),
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
      ) : null}

      {widgets.includes("kpi") && canCharts ? (
        summary.isPending ? (
          <LoadingBlock />
        ) : summary.isError ? (
          <QueryError
            message={
              forbidden
                ? "Anda tidak memiliki akses ke ringkasan dashboard."
                : mapApiError(summary.error, "Gagal memuat KPI dashboard.")
            }
          />
        ) : summary.data ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <KPICard
              tone="incident"
              label="Insiden periode ini"
              value={formatKpiNumber(summary.data.thisMonthCount)}
              trend={summary.data.trend}
              invertTrend
              hint={`vs ${formatKpiNumber(summary.data.lastMonthCount)} periode lalu`}
            />
            <KPICard
              tone="warning"
              label="LTIFR"
              value={formatRate(summary.data.ltifr)}
              hint={summary.data.workHours ? `${formatKpiNumber(summary.data.workHours)} jam kerja` : "Jam kerja belum diisi"}
            />
            <KPICard
              tone="ca"
              label="CA overdue"
              value={formatKpiNumber(summary.data.caOverdueCount)}
            />
            {nearMiss !== null ? (
              <KPICard tone="success" label="Near-miss rate" value={formatPercent(nearMiss)} />
            ) : (
              <KPICard
                tone="success"
                label="TRIFR"
                value={formatRate(summary.data.trifr)}
              />
            )}
          </div>
        ) : null
      ) : null}

      {widgets.includes("charts") && canCharts ? (
        heatmap.isPending || trends.isPending ? (
          <LoadingBlock />
        ) : chartError && !summary.isError ? (
          <QueryError message={mapApiError(chartError, "Gagal memuat grafik dashboard.")} />
        ) : (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <TrendChart data={trendSeries} />
            <CategoryDonut data={donut} />
            <PipelineChart data={pipeline} />
            <HeatmapTable matrix={matrix} />
          </div>
        )
      ) : null}

      {widgets.includes("quickActions") ? (
        <div className="flex flex-wrap gap-2">
          <Link href="/incidents/new" className={cn(buttonVariants(), "min-h-11")}>
            Buat laporan
          </Link>
          <Link href="/corrective-actions" className={cn(buttonVariants({ variant: "secondary" }), "min-h-11")}>
            CA aktif
          </Link>
        </div>
      ) : null}

      {widgets.includes("officerQueue") ? (
        <QueueSection
          title="Perlu tindakan"
          empty="Tidak ada laporan pending atau investigasi."
          loading={officerIncidents.isPending}
          error={officerIncidents.error}
          incidents={officerQueue}
          locationName={locationName}
          picName={picName}
          stale
        />
      ) : null}

      {widgets.includes("officerQueue") && officerQueue.length > 0 ? (
        <div className="rounded-lg border border-border bg-white p-4 shadow-card">
          <h2 className="text-base font-medium text-ink">Ringkasan per site</h2>
          <ul className="mt-2 flex flex-col gap-1 text-sm text-ink">
            {Object.entries(
              officerQueue.reduce<Record<string, number>>((acc, item) => {
                acc[item.locationId] = (acc[item.locationId] ?? 0) + 1;
                return acc;
              }, {}),
            ).map(([locationId, count]) => (
              <li key={locationId} className="flex justify-between gap-3">
                <span>{locationName(locationId)}</span>
                <span className="font-medium">{count}</span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {widgets.includes("officerQueue") ? (
        <div className="flex flex-col gap-3">
          <h2 className="text-base font-medium text-ink">CA jatuh tempo 7 hari</h2>
          {officerCas.isPending ? <LoadingBlock /> : null}
          {officerCas.isError ? (
            <QueryError message={mapApiError(officerCas.error, "Gagal memuat corrective action.")} />
          ) : null}
          {!officerCas.isPending && dueCas.length === 0 ? (
            <EmptyState title="Tidak ada CA mendesak" description="Tidak ada CA yang jatuh tempo dalam 7 hari." />
          ) : null}
          {dueCas.map((item) => (
            <Link
              key={item.id}
              href={`/incidents/${item.incidentId}`}
              className={cn(
                "rounded-lg border border-border bg-white p-4 shadow-card",
                urgencyRowClass(caUrgency(item)),
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <p className="text-sm font-medium text-ink">{item.description}</p>
                <CaStatusBadge status={item.status} />
              </div>
              <p className="mt-1 text-xs text-subtle">Tenggat {item.dueDate.slice(0, 10)}</p>
            </Link>
          ))}
        </div>
      ) : null}

      {widgets.includes("supervisorQueue") ? (
        <QueueSection
          title="Menunggu review"
          empty="Tidak ada laporan yang perlu diverifikasi."
          loading={officerIncidents.isPending}
          error={officerIncidents.error}
          incidents={supervisorQueue}
          locationName={locationName}
          picName={picName}
        />
      ) : null}

      {widgets.includes("reporterBrief") ? (
        <>
          <QueueSection
            title="Laporan Anda"
            empty="Anda belum memiliki laporan."
            loading={officerIncidents.isPending}
            error={officerIncidents.error}
            incidents={queueItems}
            locationName={locationName}
            picName={picName}
          />
          <div className="flex flex-col gap-3">
            <h2 className="text-base font-medium text-ink">CA yang di-assign ke Anda</h2>
            {reporterCas.isPending ? <LoadingBlock /> : null}
            {!reporterCas.isPending && myCas.length === 0 ? (
              <EmptyState title="Tidak ada CA" description="Tidak ada corrective action yang di-assign ke Anda." />
            ) : null}
            {myCas.map((item) => (
              <Link
                key={item.id}
                href={`/incidents/${item.incidentId}`}
                className="rounded-lg border border-border bg-white p-4 shadow-card"
              >
                <div className="flex items-start justify-between gap-3">
                  <p className="text-sm font-medium text-ink">{item.description}</p>
                  <CaStatusBadge status={item.status} />
                </div>
              </Link>
            ))}
          </div>
        </>
      ) : null}
    </div>
  );
}

function QueueSection({
  title,
  empty,
  loading,
  error,
  incidents,
  locationName,
  picName,
  stale,
}: {
  title: string;
  empty: string;
  loading: boolean;
  error: unknown;
  incidents: Incident[];
  locationName: (id: string) => string;
  picName: (incident: Incident) => string;
  stale?: boolean;
}) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="text-base font-medium text-ink">{title}</h2>
      {loading ? <LoadingBlock /> : null}
      {error ? <QueryError message={mapApiError(error, "Gagal memuat antrian.")} /> : null}
      {!loading && incidents.length === 0 ? <EmptyState title="Kosong" description={empty} /> : null}
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {incidents.map((incident) => (
          <div key={incident.id} className="flex flex-col gap-1">
            {stale && isStaleQueueIncident(incident) ? (
              <p className="text-xs font-medium text-danger-500">Lebih dari 3 hari tanpa kemajuan</p>
            ) : null}
            <IncidentCard
              incident={incident}
              locationName={locationName(incident.locationId)}
              picName={picName(incident)}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
