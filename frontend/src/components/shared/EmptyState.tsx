import type { ReactNode } from "react";

import { Skeleton } from "@/components/ui/skeleton";

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-start gap-3 rounded-lg border border-border bg-white p-6 shadow-card">
      <div className="flex flex-col gap-1">
        <h2 className="text-base font-medium text-ink">{title}</h2>
        <p className="text-sm leading-relaxed text-subtle">{description}</p>
      </div>
      {action}
    </div>
  );
}

export function PlaceholderPage({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-ink">{title}</h1>
      <EmptyState title="Belum tersedia" description={description} />
    </div>
  );
}

export function QueryError({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="rounded-lg border border-danger-500 bg-[#FFEBEE] px-4 py-3 text-sm text-[#B71C1C]"
    >
      {message}
    </div>
  );
}

export function LoadingBlock() {
  return (
    <div className="flex flex-col gap-3" aria-busy="true" aria-label="Memuat">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
    </div>
  );
}

export function PageHeader({
  title,
  actions,
}: {
  title: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <h1 className="text-2xl font-semibold text-ink">{title}</h1>
      {actions}
    </div>
  );
}
