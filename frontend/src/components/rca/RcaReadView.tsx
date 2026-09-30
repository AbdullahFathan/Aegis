"use client";

import { fishboneLabels, FISHBONE_KEYS } from "@/lib/schemas";
import type { Rca } from "@/lib/types";

export function RcaReadView({ rca }: { rca: Rca }) {
  const fishbone = rca.fishbone;
  return (
    <div className="flex flex-col gap-6">
      <section>
        <h2 className="text-base font-medium text-ink">Rekonstruksi kejadian</h2>
        <p className="mt-2 text-sm leading-relaxed text-ink">{rca.timeline || "—"}</p>
        <dl className="mt-3 grid gap-3 md:grid-cols-3">
          <div>
            <dt className="text-xs text-subtle">Faktor manusia</dt>
            <dd className="text-sm leading-relaxed text-ink">{rca.humanFactor || "—"}</dd>
          </div>
          <div>
            <dt className="text-xs text-subtle">Faktor lingkungan</dt>
            <dd className="text-sm leading-relaxed text-ink">{rca.environmentFactor || "—"}</dd>
          </div>
          <div>
            <dt className="text-xs text-subtle">Faktor peralatan</dt>
            <dd className="text-sm leading-relaxed text-ink">{rca.equipmentFactor || "—"}</dd>
          </div>
        </dl>
      </section>
      <section>
        <h2 className="text-base font-medium text-ink">5-Why</h2>
        <ol className="mt-2 flex flex-col gap-2">
          {(rca.fiveWhys ?? []).length === 0 ? (
            <p className="text-sm text-subtle">Belum diisi.</p>
          ) : (
            (rca.fiveWhys ?? []).map((entry, index) => (
              <li key={`${entry.why}-${index}`} className="text-sm leading-relaxed text-ink">
                <span className="text-xs text-subtle">Mengapa {index + 1}</span>
                <p>
                  {entry.why}
                  {entry.answer ? ` — ${entry.answer}` : ""}
                </p>
              </li>
            ))
          )}
        </ol>
      </section>
      <section>
        <h2 className="text-base font-medium text-ink">Fishbone</h2>
        <dl className="mt-2 grid gap-3 md:grid-cols-2">
          {FISHBONE_KEYS.map((key) => (
            <div key={key}>
              <dt className="text-xs text-subtle">{fishboneLabels[key]}</dt>
              <dd className="text-sm leading-relaxed text-ink">{fishbone?.[key] || "—"}</dd>
            </div>
          ))}
        </dl>
      </section>
    </div>
  );
}
