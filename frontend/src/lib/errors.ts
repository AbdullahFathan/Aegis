import { ApiError } from "@/lib/api/client";

export function mapLoginError(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 401) return "Kredensial salah";
    if (error.status === 429) {
      return "Terlalu banyak percobaan. Coba lagi dalam satu menit.";
    }
    return error.message;
  }
  return "Tidak dapat menghubungi server";
}

export function mapApiError(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    if (error.status === 403) return "Anda tidak memiliki akses ke halaman ini.";
    if (error.status === 409) return "Data sudah digunakan.";
    if (error.status === 422) return "Periksa kembali isian formulir.";
    return error.message || fallback;
  }
  return fallback;
}

export function mapIncidentError(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    if (error.status === 409) {
      return error.message || "Laporan sudah disubmit dan tidak dapat dikirim ulang.";
    }
    if (error.status === 422 && /verified/i.test(error.message)) {
      return "Semua corrective action harus terverifikasi sebelum laporan ditutup.";
    }
    if (error.status === 403) return "Anda tidak memiliki akses untuk aksi ini.";
    return error.message || fallback;
  }
  return fallback;
}
