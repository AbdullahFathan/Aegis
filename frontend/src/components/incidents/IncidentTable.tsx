"use client";

import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useRouter } from "next/navigation";

import { IncidentCard } from "@/components/incidents/IncidentCard";
import { SeverityBadge, StatusBadge } from "@/components/shared/StatusBadge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDateTime } from "@/lib/datetime";
import type { Incident } from "@/lib/types";
import { cn } from "@/lib/utils";
import { incidentUrgency, urgencyRowClass } from "@/lib/urgency";

export function IncidentTable({
  items,
  locationName,
  picName,
}: {
  items: Incident[];
  locationName: (locationId: string) => string;
  picName: (incident: Incident) => string;
}) {
  const router = useRouter();
  const columns = useMemo<ColumnDef<Incident>[]>(
    () => [
      {
        accessorKey: "incidentNumber",
        header: "Nomor",
        cell: ({ row }) => (
          <span className="font-mono text-xs">
            {row.original.incidentNumber ?? "Draft"}
          </span>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "severity",
        header: "Keparahan",
        cell: ({ row }) => <SeverityBadge severity={row.original.severity} />,
      },
      { accessorKey: "title", header: "Judul" },
      {
        id: "location",
        header: "Lokasi",
        cell: ({ row }) => locationName(row.original.locationId),
      },
      {
        accessorKey: "incidentDatetime",
        header: "Waktu",
        cell: ({ row }) => formatDateTime(row.original.incidentDatetime),
      },
    ],
    [locationName],
  );

  // TanStack Table returns unstable function identities; the React Compiler skips this call.
  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data: items,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <>
      <div className="flex flex-col gap-3 md:hidden">
        {items.map((incident) => (
          <IncidentCard
            key={incident.id}
            incident={incident}
            locationName={locationName(incident.locationId)}
            picName={picName(incident)}
          />
        ))}
      </div>
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
                className={cn(
                  "h-12 cursor-pointer hover:bg-canvas",
                  urgencyRowClass(incidentUrgency(row.original)),
                )}
                onClick={() => router.push(`/incidents/${row.original.id}`)}
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
    </>
  );
}
