import { cn } from "@/lib/utils";

type Tone = "incident" | "warning" | "ca" | "success";

const tones: Record<Tone, string> = {
  incident: "bg-orange-50 text-orange-700",
  warning: "bg-[#FFF8E1] text-[#7B5C00]",
  ca: "bg-navy-50 text-navy-800",
  success: "bg-[#E8F5E9] text-[#1B5E20]",
};

export function KPICard({
  label,
  value,
  hint,
  tone,
  trend,
  invertTrend,
}: {
  label: string;
  value: string;
  hint?: string;
  tone: Tone;
  trend?: "up" | "down" | "flat" | string;
  invertTrend?: boolean;
}) {
  const upIsBad = invertTrend ?? tone === "incident";
  const trendLabel =
    trend === "up" ? "↑" : trend === "down" ? "↓" : trend === "flat" ? "→" : null;
  const trendClass =
    trend === "flat" || !trend
      ? "text-subtle"
      : (trend === "up") === upIsBad
        ? "text-danger-500"
        : "text-emerald-500";

  return (
    <div className={cn("rounded-lg border border-border p-3", tones[tone])}>
      <p className="text-xs font-medium">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
      {hint || trendLabel ? (
        <p className={cn("mt-1 text-xs", trendLabel ? trendClass : "opacity-80")}>
          {trendLabel ? `${trendLabel} ` : null}
          {hint}
        </p>
      ) : null}
    </div>
  );
}
