"use client";

import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import type { PipelineBucket } from "@/lib/dashboard";
import { colors } from "@/lib/tokens";

export function PipelineChart({ data }: { data: PipelineBucket[] }) {
  return (
    <div className="rounded-lg border border-border bg-white p-4 shadow-card">
      <h2 className="mb-3 text-base font-medium text-ink">Pipeline status</h2>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid stroke={colors.border} vertical={false} />
            <XAxis dataKey="label" tick={{ fontSize: 11, fill: colors.muted }} />
            <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: colors.muted }} width={32} />
            <Tooltip />
            <Bar dataKey="count" name="Laporan" fill={colors.navy[800]} radius={[6, 6, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
