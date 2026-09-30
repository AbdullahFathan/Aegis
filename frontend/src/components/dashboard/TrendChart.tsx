"use client";

import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { colors } from "@/lib/tokens";
import type { TrendPoint } from "@/lib/dashboard";

export function TrendChart({ data }: { data: TrendPoint[] }) {
  return (
    <div className="rounded-lg border border-border bg-white p-4 shadow-card">
      <h2 className="mb-3 text-base font-medium text-ink">Tren insiden 12 bulan</h2>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid stroke={colors.border} vertical={false} />
            <XAxis dataKey="label" tick={{ fontSize: 11, fill: colors.muted }} />
            <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: colors.muted }} width={32} />
            <Tooltip />
            <Line type="monotone" dataKey="count" name="Insiden" stroke={colors.orange[500]} strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
