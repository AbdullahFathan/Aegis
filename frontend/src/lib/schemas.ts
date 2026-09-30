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
