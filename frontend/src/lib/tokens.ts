/**
 * Aegis Design Tokens — Warm Safety
 * Nilai sama dengan PRD §12 dan design guide.
 */

export const colors = {
  orange: { 50: "#FFF1E6", 500: "#E85D04", 700: "#B84A00" },
  navy: { 50: "#E8EAF6", 600: "#3949AB", 800: "#1A237E" },
  amber: { 400: "#F59E0B" },
  emerald: { 500: "#22C55E" },
  danger: { 500: "#EF4444" },
  canvas: "#F8FAFC",
  surface: "#FFFFFF",
  ink: "#1E293B",
  muted: "#475569", // caption AA on canvas/white; PRD listed #64748B
  border: "#E2E8F0",
  borderStrong: "#CBD5E1",
  successBg: "#E8F5E9",
  successText: "#1B5E20",
  warningBg: "#FFF8E1",
  warningText: "#7B5C00",
  dangerBg: "#FFEBEE",
  dangerText: "#B71C1C",
} as const;

export type SeverityLevel = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

export const severityConfig: Record<
  SeverityLevel,
  {
    label: string;
    badgeBg: string;
    badgeBorder: string;
    badgeText: string;
    icon: string;
  }
> = {
  CRITICAL: {
    label: "Kritis",
    badgeBg: colors.dangerBg,
    badgeBorder: colors.danger[500],
    badgeText: colors.dangerText,
    icon: "AlertTriangle",
  },
  HIGH: {
    label: "Tinggi",
    badgeBg: colors.orange[50],
    badgeBorder: colors.orange[500],
    badgeText: colors.orange[700],
    icon: "AlertCircle",
  },
  MEDIUM: {
    label: "Sedang",
    badgeBg: colors.warningBg,
    badgeBorder: colors.amber[400],
    badgeText: colors.warningText,
    icon: "AlertCircle",
  },
  LOW: {
    label: "Rendah",
    badgeBg: colors.successBg,
    badgeBorder: colors.emerald[500],
    badgeText: colors.successText,
    icon: "Info",
  },
};

/** Lifecycle laporan — selaras PRD, bukan OPEN/IN_REVIEW/RESOLVED */
export type IncidentStatus =
  | "DRAFT"
  | "PENDING_REVIEW"
  | "UNDER_INVESTIGATION"
  | "CORRECTIVE_ACTION"
  | "CLOSED"
  | "REJECTED";

export const statusConfig: Record<
  IncidentStatus,
  {
    label: string;
    badgeBg: string;
    badgeBorder: string;
    badgeText: string;
  }
> = {
  DRAFT: {
    label: "Draft",
    badgeBg: colors.canvas,
    badgeBorder: colors.borderStrong,
    badgeText: colors.muted,
  },
  PENDING_REVIEW: {
    label: "Menunggu review",
    badgeBg: colors.navy[50],
    badgeBorder: colors.navy[600],
    badgeText: colors.navy[800],
  },
  UNDER_INVESTIGATION: {
    label: "Investigasi",
    badgeBg: colors.warningBg,
    badgeBorder: colors.amber[400],
    badgeText: colors.warningText,
  },
  CORRECTIVE_ACTION: {
    label: "Tindakan perbaikan",
    badgeBg: colors.orange[50],
    badgeBorder: colors.orange[500],
    badgeText: colors.orange[700],
  },
  CLOSED: {
    label: "Ditutup",
    badgeBg: colors.successBg,
    badgeBorder: colors.emerald[500],
    badgeText: colors.successText,
  },
  REJECTED: {
    label: "Dikembalikan",
    badgeBg: colors.dangerBg,
    badgeBorder: colors.danger[500],
    badgeText: colors.dangerText,
  },
};

export type CorrectiveActionStatus =
  | "OPEN"
  | "IN_PROGRESS"
  | "DONE"
  | "OVERDUE"
  | "VERIFIED";

export const caStatusConfig: Record<
  CorrectiveActionStatus,
  {
    label: string;
    badgeBg: string;
    badgeBorder: string;
    badgeText: string;
  }
> = {
  OPEN: {
    label: "Terbuka",
    badgeBg: colors.navy[50],
    badgeBorder: colors.navy[600],
    badgeText: colors.navy[800],
  },
  IN_PROGRESS: {
    label: "Sedang dikerjakan",
    badgeBg: colors.warningBg,
    badgeBorder: colors.amber[400],
    badgeText: colors.warningText,
  },
  DONE: {
    label: "Selesai",
    badgeBg: colors.successBg,
    badgeBorder: colors.emerald[500],
    badgeText: colors.successText,
  },
  OVERDUE: {
    label: "Terlambat",
    badgeBg: colors.dangerBg,
    badgeBorder: colors.danger[500],
    badgeText: colors.dangerText,
  },
  VERIFIED: {
    label: "Terverifikasi",
    badgeBg: colors.successBg,
    badgeBorder: colors.emerald[500],
    badgeText: colors.successText,
  },
};

export const typography = {
  display: "text-2xl font-semibold text-ink",
  heading: "text-xl font-semibold text-ink",
  subheading: "text-base font-medium text-subtle",
  body: "text-sm text-ink",
  caption: "text-xs text-subtle",
  label: "text-sm font-medium text-ink",
  mono: "font-mono text-xs text-subtle",
} as const;

export const componentClasses = {
  btnPrimary:
    "h-9 px-4 text-sm rounded-md bg-orange-500 text-white hover:bg-orange-700 transition-colors inline-flex items-center gap-2 font-medium",
  btnSecondary:
    "h-9 px-4 text-sm rounded-md border border-navy-800 bg-transparent text-navy-800 hover:bg-navy-50 transition-colors inline-flex items-center gap-2",
  btnDanger:
    "h-9 px-4 text-sm rounded-md bg-danger-500 text-white hover:bg-[#B71C1C] transition-colors inline-flex items-center gap-2",
  btnGhost:
    "h-9 px-3 text-sm rounded-md bg-orange-50 text-orange-700 hover:bg-orange-50/80 transition-colors inline-flex items-center gap-2",
  input:
    "h-9 w-full rounded-md border border-border-strong px-3 text-sm text-ink placeholder:text-subtle focus:border-orange-500 focus:outline-none focus:ring-1 focus:ring-orange-500",
  label: "text-sm font-medium text-ink mb-1 block",
  helperText: "text-xs text-subtle mt-1",
  errorText: "text-xs text-danger-500 mt-1",
  card: "bg-white border border-border rounded-lg shadow-card",
  cardPadding: "p-4",
  cardHeader: "border-b border-border pb-3 mb-4",
  badge:
    "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium border",
  tableWrapper: "border border-border rounded-lg overflow-hidden",
  tableHead: "bg-canvas text-xs font-medium text-subtle uppercase tracking-wide",
  tableRow: "border-b border-border hover:bg-canvas transition-colors",
  tableCell: "px-4 py-3 text-sm text-ink",
} as const;

export const layout = {
  pageBackground: "min-h-screen bg-canvas",
  sidebarWidth: "w-60",
  topbarHeight: "h-14",
  contentPadding: "p-6",
  contentMaxWidth: "max-w-[1280px] mx-auto",
} as const;
