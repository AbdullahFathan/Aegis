"use client";

import { useCallback, useRef, useState } from "react";
import { useDropzone } from "react-dropzone";

import { Button } from "@/components/ui/button";
import { getFileUrl, MAX_UPLOAD_BYTES, MAX_UPLOAD_FILES, validateUploadFiles } from "@/lib/files";
import type { IncidentFile } from "@/lib/types";

function isImage(mime: string) {
  return mime.startsWith("image/");
}

export function FileUploader({
  files,
  canUpload,
  uploading,
  onUpload,
  onRefreshFiles,
}: {
  files: IncidentFile[];
  canUpload: boolean;
  uploading?: boolean;
  onUpload: (files: File[]) => Promise<void>;
  onRefreshFiles: () => Promise<IncidentFile[] | undefined>;
}) {
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState<string | null>(null);
  const retried = useRef<Set<string>>(new Set());

  const onDrop = useCallback(
    async (accepted: File[]) => {
      setError(null);
      const issues = validateUploadFiles(files.length, accepted);
      if (issues.length > 0) {
        setError(issues[0] ?? null);
        return;
      }
      setProgress(`Mengunggah ${accepted.length} file…`);
      try {
        await onUpload(accepted);
        setProgress(null);
      } catch (cause) {
        setProgress(null);
        setError(cause instanceof Error ? cause.message : "Gagal mengunggah file.");
      }
    },
    [files.length, onUpload],
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    disabled: !canUpload || uploading,
    maxSize: MAX_UPLOAD_BYTES,
    maxFiles: MAX_UPLOAD_FILES,
    accept: {
      "image/jpeg": [".jpg", ".jpeg"],
      "image/png": [".png"],
      "image/webp": [".webp"],
      "application/pdf": [".pdf"],
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document": [".docx"],
    },
  });

  async function handleImageError(file: IncidentFile) {
    if (!file.url || retried.current.has(file.id)) return;
    retried.current.add(file.id);
    await getFileUrl(file.url, async () => {
      const next = await onRefreshFiles();
      return next?.find((item) => item.id === file.id)?.url ?? file.url ?? "";
    });
  }

  return (
    <div className="flex flex-col gap-3">
      {canUpload ? (
        <div
          {...getRootProps()}
          data-testid="evidence-dropzone"
          className="flex min-h-28 cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed border-border-strong bg-canvas px-4 py-6 text-center"
        >
          <input {...getInputProps()} data-testid="evidence-file" />
          <p className="text-sm text-ink">
            {isDragActive ? "Lepaskan file di sini" : "Seret file atau klik untuk memilih"}
          </p>
          <p className="mt-1 text-xs text-subtle">JPEG, PNG, WebP, PDF, DOCX. Maks. 5 file, 10MB per file.</p>
        </div>
      ) : null}
      {progress ? <p className="text-xs text-subtle">{progress}</p> : null}
      {error ? (
        <p role="alert" className="text-xs text-danger-500">
          {error}
        </p>
      ) : null}
      {files.length === 0 ? (
        <p className="text-sm text-subtle">Belum ada bukti terunggah.</p>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2">
          {files.map((file) => (
            <li key={file.id} className="rounded-lg border border-border bg-white p-3">
              {file.url && isImage(file.mimeType) ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={file.url}
                  alt={file.originalName}
                  className="mb-2 h-36 w-full rounded-md object-cover"
                  onError={() => {
                    void handleImageError(file);
                  }}
                />
              ) : null}
              <p className="truncate text-sm text-ink">{file.originalName}</p>
              <p className="text-xs text-subtle">{file.mimeType}</p>
              {file.url && !isImage(file.mimeType) ? (
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-2 min-h-11"
                  onClick={async () => {
                    const url = await getFileUrl(file.url ?? "", async () => {
                      const next = await onRefreshFiles();
                      return next?.find((item) => item.id === file.id)?.url ?? file.url ?? "";
                    });
                    window.open(url, "_blank", "noopener,noreferrer");
                  }}
                >
                  Unduh
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
