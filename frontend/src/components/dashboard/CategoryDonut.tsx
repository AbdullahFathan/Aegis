"use client";

import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import type { DonutSlice } from "@/lib/dashboard";
import { colors } from "@/lib/tokens";

const SLICE_COLORS = [
  colors.emerald[500],
  colors.navy[600],
  colors.amber[400],
  colors.orange[500],
  colors.danger[500],
  colors.navy[800],
  colors.orange[700],
];

export function CategoryDonut({ data }: { data: DonutSlice[] }) {
  return (
    <div className="rounded-lg border border-border bg-white p-4 shadow-card">
      <h2 className="mb-3 text-base font-medium text-ink">Distribusi kategori</h2>
      {data.length === 0 ? (
        <p className="text-sm text-subtle">Tidak ada data kategori pada filter ini.</p>
      ) : (
        <div className="flex flex-col gap-3 md:flex-row md:items-center">
          <div className="h-56 w-full md:w-56">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={data} dataKey="count" nameKey="label" innerRadius={48} outerRadius={80} stroke={colors.surface}>
                  {data.map((slice, index) => (
                    <Cell key={slice.category} fill={SLICE_COLORS[index % SLICE_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
          <ul className="flex flex-1 flex-col gap-1 text-sm text-ink">
            {data.map((slice, index) => (
              <li key={slice.category} className="flex items-center justify-between gap-3">
                <span className="flex items-center gap-2">
                  <span
                    className="size-2.5 rounded-sm"
                    style={{ background: SLICE_COLORS[index % SLICE_COLORS.length] }}
                  />
                  {slice.label}
                </span>
                <span className="font-medium">{slice.count}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
