"use client";

import { Clock, MapPin, UserRound } from "lucide-react";
import Link from "next/link";

import { SeverityBadge, StatusBadge } from "@/components/shared/StatusBadge";
import { formatDateTime } from "@/lib/datetime";
import type { Incident } from "@/lib/types";
import { cn } from "@/lib/utils";
import { incidentUrgency, urgencyRowClass } from "@/lib/urgency";

export function isUrgentIncident(incident: Incident) {
  return incidentUrgency(incident) !== "none";
}

export function IncidentCard({
  incident,
  locationName,
  picName,
}: {
  incident: Incident;
  locationName: string;
  picName: string;
}) {
  const urgent = isUrgentIncident(incident);
  return (
    <Link
      href={`/incidents/${incident.id}`}
      className={cn(
        "block rounded-lg border border-border bg-white p-4 shadow-card",
        urgent && urgencyRowClass(incidentUrgency(incident)),
      )}
    >
      <div className="mb-2 flex items-start justify-between gap-3">
        <span className="font-mono text-xs text-subtle">
          {incident.incidentNumber ?? "Draft"}
        </span>
        <SeverityBadge severity={incident.severity} />
      </div>
      <h3 className="text-base font-medium text-ink">{incident.title}</h3>
      <p className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-subtle">
        <span className="inline-flex items-center gap-1">
          <MapPin aria-hidden className="size-4" />
          {locationName}
        </span>
        <span className="inline-flex items-center gap-1">
          <Clock aria-hidden className="size-4" />
          {formatDateTime(incident.incidentDatetime)}
        </span>
      </p>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <StatusBadge status={incident.status} />
        <span className="inline-flex items-center gap-1 text-xs text-subtle">
          <UserRound aria-hidden className="size-4" />
          {picName}
        </span>
      </div>
    </Link>
  );
}
