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

  const payload = (await response.json().catch(() => null)) as Envelope<T> | null;

  if (response.status === 401 && auth && !retry && path !== "/auth/refresh") {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return api<T>(path, { ...options, retry: true });
    }
    clearSession();
  }

  if (!response.ok || !payload?.success) {
    throw new ApiError(
      response.status,
      payload?.error?.code ?? "ERROR",
      payload?.error?.message ?? "Permintaan gagal",
    );
  }

  return payload.data as T;
}
