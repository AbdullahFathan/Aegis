export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;
export const MAX_UPLOAD_FILES = 5;

export const ALLOWED_UPLOAD_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
  "application/pdf",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
]);

const ALLOWED_EXTENSIONS = [".jpg", ".jpeg", ".png", ".webp", ".pdf", ".docx"];

export function fileLooksAllowed(file: File) {
  if (ALLOWED_UPLOAD_TYPES.has(file.type)) return true;
  const name = file.name.toLowerCase();
  return ALLOWED_EXTENSIONS.some((ext) => name.endsWith(ext));
}

export function validateUploadFiles(existingCount: number, files: File[]) {
  const errors: string[] = [];
  if (existingCount + files.length > MAX_UPLOAD_FILES) {
    errors.push("Maksimal 5 file per sesi unggah.");
  }
  for (const file of files) {
    if (file.size > MAX_UPLOAD_BYTES) {
      errors.push(`${file.name} melebihi 10MB.`);
    }
    if (!fileLooksAllowed(file)) {
      errors.push(`${file.name} harus JPEG, PNG, WebP, PDF, atau DOCX.`);
    }
  }
  return errors;
}

/** If the signed URL is expired (403), refresh once via the provided callback. */
export async function getFileUrl(url: string, refresh: () => Promise<string>) {
  const response = await fetch(url, { method: "HEAD" });
  if (response.status === 403) {
    return refresh();
  }
  return url;
}
