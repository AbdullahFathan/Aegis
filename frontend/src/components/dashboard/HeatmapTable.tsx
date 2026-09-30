import { heatmapIntensity, type HeatmapMatrix } from "@/lib/dashboard";
import { categoryLabels, type IncidentCategoryValue } from "@/lib/schemas";

export function HeatmapTable({ matrix }: { matrix: HeatmapMatrix }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border bg-white shadow-card">
      <div className="border-b border-border p-4">
        <h2 className="text-base font-medium text-ink">Heatmap site × kategori</h2>
      </div>
      {matrix.locations.length === 0 ? (
        <p className="p-4 text-sm text-subtle">Tidak ada data heatmap pada filter ini.</p>
      ) : (
        <table className="w-full text-sm">
          <thead className="bg-canvas text-xs font-medium tracking-wide text-subtle uppercase">
            <tr>
              <th className="px-3 py-2 text-left">Site</th>
              {matrix.categories.map((category) => (
                <th key={category} className="px-2 py-2 text-center">
                  {categoryLabels[category as IncidentCategoryValue] ?? category}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {matrix.locations.map((location) => (
              <tr key={location.id} className="border-t border-border">
                <td className="px-3 py-2 font-mono text-xs text-ink">{location.code}</td>
                {matrix.categories.map((category) => {
                  const count = matrix.values[location.id]?.[category] ?? 0;
                  const intensity = heatmapIntensity(count, matrix.max);
                  return (
                    <td key={category} className="px-2 py-2 text-center">
                      <span
                        className="inline-flex min-w-8 items-center justify-center rounded-md px-2 py-1 text-xs text-navy-800"
                        style={{ backgroundColor: `rgba(232, 93, 4, ${intensity * 0.55})` }}
                      >
                        {count || "—"}
                      </span>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
