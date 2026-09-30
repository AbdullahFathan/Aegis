import { z } from "zod";

export const ROLES = [
  "SUPER_ADMIN",
  "ADMIN",
  "HSE_MANAGER",
  "HSE_OFFICER",
  "SUPERVISOR",
  "REPORTER",
] as const;

export const USER_STATUSES = ["ACTIVE", "INACTIVE"] as const;

export const LOCATION_TYPES = [
  "TAMBANG",
  "TELCO_SITE",
  "KANTOR",
  "GUDANG",
  "WORKSHOP",
  "LAINNYA",
] as const;

export type Role = (typeof ROLES)[number];
export type UserStatus = (typeof USER_STATUSES)[number];
export type LocationType = (typeof LOCATION_TYPES)[number];

export const roleLabels: Record<Role, string> = {
  SUPER_ADMIN: "Super Admin",
  ADMIN: "Admin",
  HSE_MANAGER: "HSE Manager",
  HSE_OFFICER: "HSE Officer",
  SUPERVISOR: "Supervisor",
  REPORTER: "Reporter",
};

export const statusLabels: Record<UserStatus, string> = {
  ACTIVE: "Aktif",
  INACTIVE: "Nonaktif",
};

export const locationTypeLabels: Record<LocationType, string> = {
  TAMBANG: "Tambang",
  TELCO_SITE: "Site telekomunikasi",
  KANTOR: "Kantor",
  GUDANG: "Gudang",
  WORKSHOP: "Workshop",
  LAINNYA: "Lainnya",
};

const required = (label: string) => z.string().trim().min(1, `${label} wajib diisi`);

export const loginSchema = z.object({
  email: required("Email"),
  password: required("Password"),
});

export const userFormSchema = z.object({
  name: required("Nama"),
  email: required("Email"),
  password: required("Password"),
  role: z.enum(ROLES, { error: "Role wajib dipilih" }),
  status: z.enum(USER_STATUSES, { error: "Status wajib dipilih" }),
});

export const userPatchSchema = z.object({
  name: required("Nama"),
  role: z.enum(ROLES, { error: "Role wajib dipilih" }),
  status: z.enum(USER_STATUSES, { error: "Status wajib dipilih" }),
});

export const regionSchema = z.object({
  name: required("Nama"),
  code: required("Kode"),
});

export const locationSchema = z.object({
  name: required("Nama"),
  code: required("Kode"),
  type: z.enum(LOCATION_TYPES, { error: "Tipe lokasi wajib dipilih" }),
  regionId: z.string(),
  supervisorId: required("Supervisor"),
  hseOfficerId: required("HSE Officer"),
});

export const areaSchema = z.object({
  name: required("Nama"),
  code: required("Kode"),
});

export type LoginValues = z.infer<typeof loginSchema>;
export type UserFormValues = z.infer<typeof userFormSchema>;
export type UserPatchValues = z.infer<typeof userPatchSchema>;
export type RegionValues = z.infer<typeof regionSchema>;
export type LocationValues = z.infer<typeof locationSchema>;
export type AreaValues = z.infer<typeof areaSchema>;

export const INCIDENT_CATEGORIES = [
  "NEAR_MISS",
  "FIRST_AID",
  "MEDICAL_TREATMENT",
  "LTI",
  "FATALITY",
  "PROPERTY_DAMAGE",
  "ENVIRONMENTAL",
] as const;

export const INCIDENT_SEVERITIES = ["LOW", "MEDIUM", "HIGH", "CRITICAL"] as const;

export const INCIDENT_STATUSES = [
  "DRAFT",
  "PENDING_REVIEW",
  "UNDER_INVESTIGATION",
  "CORRECTIVE_ACTION",
  "CLOSED",
  "REJECTED",
] as const;

export type IncidentCategoryValue = (typeof INCIDENT_CATEGORIES)[number];
export type IncidentSeverityValue = (typeof INCIDENT_SEVERITIES)[number];
export type IncidentStatusValue = (typeof INCIDENT_STATUSES)[number];

export const categoryLabels: Record<IncidentCategoryValue, string> = {
  NEAR_MISS: "Near-miss",
  FIRST_AID: "First aid",
  MEDICAL_TREATMENT: "Medical treatment",
  LTI: "Lost time injury (LTI)",
  FATALITY: "Fatality",
  PROPERTY_DAMAGE: "Kerusakan aset",
  ENVIRONMENTAL: "Insiden lingkungan",
};

export const categoryGuides: Record<IncidentCategoryValue, string> = {
  NEAR_MISS:
    "Kejadian yang hampir menimbulkan cedera atau kerugian, tetapi tidak sampai terjadi. Pelaporan near-miss menandai budaya keselamatan yang baik.",
  FIRST_AID:
    "Cedera ringan yang ditangani dengan pertolongan pertama di tempat, tanpa perawatan medis lanjutan.",
  MEDICAL_TREATMENT:
    "Korban memerlukan perawatan medis di luar first aid, tetapi tidak kehilangan hari kerja.",
  LTI: "Korban kehilangan hari kerja karena cedera. Eskalasi lebih tinggi ke HSE.",
  FATALITY: "Kematian terkait kerja. Notifikasi kritis ke Supervisor, HSE Officer, dan HSE Manager.",
  PROPERTY_DAMAGE: "Kerusakan peralatan, kendaraan, atau aset tanpa (atau terpisah dari) cedera manusia.",
  ENVIRONMENTAL: "Tumpahan, emisi, atau dampak lingkungan yang perlu dicatat dan ditindaklanjuti.",
};

export const severityLabels: Record<IncidentSeverityValue, string> = {
  LOW: "Rendah",
  MEDIUM: "Sedang",
  HIGH: "Tinggi",
  CRITICAL: "Kritis",
};

const optionalText = z.string();

export const incidentFilterSchema = z.object({
  status: z.enum(INCIDENT_STATUSES, { error: "Status tidak valid" }).optional(),
  category: z.enum(INCIDENT_CATEGORIES, { error: "Kategori tidak valid" }).optional(),
  locationId: z.string().min(1).optional(),
  from: z.string().optional(),
  to: z.string().optional(),
  page: z.coerce.number().int().min(1).default(1),
  pageSize: z.coerce.number().int().min(1).max(100).default(20),
});

export const incidentFormFields = z.object({
  title: required("Judul"),
  description: z.string().trim().min(50, "Deskripsi minimal 50 karakter"),
  category: z.enum(INCIDENT_CATEGORIES, { error: "Kategori wajib dipilih" }),
  severity: z.enum(INCIDENT_SEVERITIES, { error: "Tingkat keparahan wajib dipilih" }),
  incidentDatetime: required("Tanggal dan waktu"),
  locationId: required("Lokasi"),
  areaId: optionalText,
  hasVictim: z.boolean(),
  victimName: optionalText,
  victimPosition: optionalText,
  injuryDescription: optionalText,
  initialTreatment: optionalText,
  witnesses: z.array(
    z.object({
      name: optionalText,
      position: optionalText,
    }),
  ),
});

export const incidentFormSchema = incidentFormFields.superRefine((value, ctx) => {
    if (!value.hasVictim) return;
    if (!value.victimName.trim()) {
      ctx.addIssue({ code: "custom", path: ["victimName"], message: "Nama korban wajib diisi" });
    }
    if (!value.injuryDescription.trim()) {
      ctx.addIssue({
        code: "custom",
        path: ["injuryDescription"],
        message: "Jenis cedera wajib diisi",
      });
    }
  });

export type IncidentFilterValues = z.infer<typeof incidentFilterSchema>;
export type IncidentFormValues = z.infer<typeof incidentFormSchema>;

export function parseIncidentSearchParams(
  params: Record<string, string | string[] | undefined>,
): IncidentFilterValues {
  const pick = (key: string) => {
    const value = params[key];
    if (Array.isArray(value)) return value[0];
    return value;
  };
  const status = pick("status");
  const category = pick("category");
  const parsed = incidentFilterSchema.safeParse({
    status: status && (INCIDENT_STATUSES as readonly string[]).includes(status) ? status : undefined,
    category:
      category && (INCIDENT_CATEGORIES as readonly string[]).includes(category) ? category : undefined,
    locationId: pick("locationId") || undefined,
    from: pick("from") || undefined,
    to: pick("to") || undefined,
    page: pick("page") ?? 1,
    pageSize: pick("pageSize") ?? 20,
  });
  if (parsed.success) return parsed.data;
  return { page: 1, pageSize: 20 };
}

export const FISHBONE_KEYS = [
  "man",
  "machine",
  "method",
  "material",
  "environment",
  "measurement",
] as const;

export const fishboneLabels: Record<(typeof FISHBONE_KEYS)[number], string> = {
  man: "Man (manusia)",
  machine: "Machine (mesin/peralatan)",
  method: "Method (metode/prosedur)",
  material: "Material (bahan)",
  environment: "Environment (lingkungan)",
  measurement: "Measurement (pengukuran/SOP)",
};

export const emptyFishbone = {
  man: "",
  machine: "",
  method: "",
  material: "",
  environment: "",
  measurement: "",
};

export const rcaSchema = z.object({
  timeline: optionalText,
  humanFactor: optionalText,
  environmentFactor: optionalText,
  equipmentFactor: optionalText,
  fiveWhys: z
    .array(
      z.object({
        why: optionalText,
        answer: optionalText,
      }),
    )
    .max(5, "5-Why maksimal 5 entri"),
  fishbone: z.object({
    man: optionalText,
    machine: optionalText,
    method: optionalText,
    material: optionalText,
    environment: optionalText,
    measurement: optionalText,
  }),
});

export type RcaFormValues = z.infer<typeof rcaSchema>;

export function emptyRcaValues(): RcaFormValues {
  return {
    timeline: "",
    humanFactor: "",
    environmentFactor: "",
    equipmentFactor: "",
    fiveWhys: [{ why: "", answer: "" }],
    fishbone: { ...emptyFishbone },
  };
}

export const CA_ACTION_TYPES = ["IMMEDIATE", "SHORT_TERM", "LONG_TERM"] as const;
export const CA_PRIORITIES = ["LOW", "MEDIUM", "HIGH"] as const;
export const CA_STATUSES = ["OPEN", "IN_PROGRESS", "DONE", "OVERDUE", "VERIFIED"] as const;

export const caActionTypeLabels: Record<(typeof CA_ACTION_TYPES)[number], string> = {
  IMMEDIATE: "Segera",
  SHORT_TERM: "Jangka pendek",
  LONG_TERM: "Jangka panjang",
};

export const caPriorityLabels: Record<(typeof CA_PRIORITIES)[number], string> = {
  LOW: "Rendah",
  MEDIUM: "Sedang",
  HIGH: "Tinggi",
};

export const caCreateSchema = z.object({
  description: required("Deskripsi tindakan"),
  actionType: z.enum(CA_ACTION_TYPES, { error: "Tipe wajib dipilih" }),
  priority: z.enum(CA_PRIORITIES, { error: "Prioritas wajib dipilih" }),
  assigneeId: required("Penanggung jawab"),
  dueDate: required("Tenggat"),
});

export type CaCreateValues = z.infer<typeof caCreateSchema>;

export const caDoneSchema = z.object({
  completionNotes: required("Catatan penyelesaian"),
});

export const caTrackerFilterSchema = z.object({
  status: z.enum(CA_STATUSES, { error: "Status tidak valid" }).optional(),
  assigneeId: z.string().min(1).optional(),
  priority: z.enum(CA_PRIORITIES, { error: "Prioritas tidak valid" }).optional(),
  locationId: z.string().min(1).optional(),
  dueFrom: z.string().optional(),
  dueTo: z.string().optional(),
  view: z.enum(["table", "kanban"]).default("table"),
});

export type CaTrackerFilterValues = z.infer<typeof caTrackerFilterSchema>;

export function parseCaTrackerSearchParams(
  params: Record<string, string | string[] | undefined>,
): CaTrackerFilterValues {
  const pick = (key: string) => {
    const value = params[key];
    if (Array.isArray(value)) return value[0];
    return value;
  };
  const status = pick("status");
  const priority = pick("priority");
  const view = pick("view");
  const parsed = caTrackerFilterSchema.safeParse({
    status: status && (CA_STATUSES as readonly string[]).includes(status) ? status : undefined,
    assigneeId: pick("assigneeId") || undefined,
    priority: priority && (CA_PRIORITIES as readonly string[]).includes(priority) ? priority : undefined,
    locationId: pick("locationId") || undefined,
    dueFrom: pick("dueFrom") || undefined,
    dueTo: pick("dueTo") || undefined,
    view: view === "kanban" ? "kanban" : "table",
  });
  if (parsed.success) return parsed.data;
  return { view: "table" };
}

export const NOTIFICATION_TYPES = [
  "INCIDENT_SUBMITTED",
  "INCIDENT_ESCALATED",
  "INCIDENT_VERIFIED",
  "INCIDENT_REJECTED",
  "INCIDENT_CLOSED",
  "SLA_WARNING",
  "SLA_OVERDUE",
  "CA_ASSIGNED",
  "CA_DUE_SOON",
  "CA_OVERDUE",
  "REPORT_READY",
] as const;

export const notificationTypeLabels: Record<(typeof NOTIFICATION_TYPES)[number], string> = {
  INCIDENT_SUBMITTED: "Laporan baru disubmit",
  INCIDENT_ESCALATED: "Insiden dieskalasi",
  INCIDENT_VERIFIED: "Laporan diverifikasi",
  INCIDENT_REJECTED: "Laporan dikembalikan",
  INCIDENT_CLOSED: "Laporan ditutup",
  SLA_WARNING: "SLA hampir habis",
  SLA_OVERDUE: "SLA terlewat",
  CA_ASSIGNED: "Corrective action di-assign",
  CA_DUE_SOON: "CA mendekati tenggat",
  CA_OVERDUE: "CA overdue",
  REPORT_READY: "Laporan siap diunduh",
};
