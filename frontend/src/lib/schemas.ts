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
