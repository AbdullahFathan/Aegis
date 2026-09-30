export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

type Envelope<T> = {
  success: boolean;
  data?: T;
  error?: { code?: string; message?: string };
};

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  auth?: boolean;
  retry?: boolean;
};

let accessToken: string | null = null;
let refreshPromise: Promise<boolean> | null = null;
let onSessionCleared: () => void = () => {};

export function getAccessToken() {
  return accessToken;
}

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function setSessionClearedHandler(handler: () => void) {
  onSessionCleared = handler;
}

export function resetAuthState() {
  accessToken = null;
  refreshPromise = null;
  onSessionCleared = () => {};
}

export function apiBaseUrl() {
  return process.env.NEXT_PUBLIC_API_BASE_URL || "/api";
}

function clearSession() {
  accessToken = null;
  onSessionCleared();
}

export async function refreshAccessToken() {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const response = await fetch(`${apiBaseUrl()}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
      const body = (await response.json().catch(() => null)) as Envelope<{
        accessToken: string;
      }> | null;
      if (!response.ok || !body?.success || !body.data?.accessToken) {
        clearSession();
        return false;
      }
      accessToken = body.data.accessToken;
      return true;
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await authorizedFetch(path, options);

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = (await response.json().catch(() => null)) as Envelope<T> | null;

  if (!response.ok || !payload?.success) {
    throw new ApiError(
      response.status,
      payload?.error?.code ?? "ERROR",
      payload?.error?.message ?? "Permintaan gagal",
    );
  }

  return payload.data as T;
}

function filenameFromDisposition(header: string | null) {
  if (!header) return "download";
  const utf = header.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf?.[1]) return decodeURIComponent(utf[1].replace(/"/g, ""));
  const plain = header.match(/filename="?([^"]+)"?/i);
  return plain?.[1] ?? "download";
}

async function authorizedFetch(path: string, options: RequestOptions = {}): Promise<Response> {
  const { body, auth = true, retry = false, headers, ...init } = options;
  const isFormData = typeof FormData !== "undefined" && body instanceof FormData;
  const requestHeaders = new Headers(headers);
  if (body !== undefined && !isFormData && !requestHeaders.has("Content-Type")) {
    requestHeaders.set("Content-Type", "application/json");
  }
  if (auth && accessToken) {
    requestHeaders.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`${apiBaseUrl()}${path}`, {
    ...init,
    headers: requestHeaders,
    credentials: "include",
    body: body === undefined ? undefined : isFormData ? body : JSON.stringify(body),
  });

  if (response.status === 401 && auth && !retry && path !== "/auth/refresh") {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return authorizedFetch(path, { ...options, retry: true });
    }
    clearSession();
  }

  return response;
}

export type ApiFileResult<T> =
  | { kind: "json"; data: T; status: number }
  | { kind: "file"; blob: Blob; filename: string };

export async function apiDownload<T = unknown>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiFileResult<T>> {
  const response = await authorizedFetch(path, options);
  const contentType = response.headers.get("Content-Type") ?? "";

  if (contentType.includes("application/json") || response.status === 202) {
    const payload = (await response.json().catch(() => null)) as Envelope<T> | null;
    if (!response.ok || !payload?.success) {
      throw new ApiError(
        response.status,
        payload?.error?.code ?? "ERROR",
        payload?.error?.message ?? "Permintaan gagal",
      );
    }
    return { kind: "json", data: payload.data as T, status: response.status };
  }

  if (!response.ok) {
    throw new ApiError(response.status, "ERROR", "Unduhan gagal");
  }

  return {
    kind: "file",
    blob: await response.blob(),
    filename: filenameFromDisposition(response.headers.get("Content-Disposition")),
  };
}

export function triggerBrowserDownload(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = href;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(href);
}
