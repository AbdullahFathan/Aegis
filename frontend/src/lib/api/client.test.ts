import { afterEach, describe, expect, it, vi } from "vitest";

import { api, getAccessToken, resetAuthState, setAccessToken } from "@/lib/api/client";
import { getFileUrl, validateUploadFiles } from "@/lib/files";
import { restoreFetch, stubFetch } from "@/test/stub-fetch";

function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => {
  resetAuthState();
  restoreFetch();
});

describe("api client", () => {
  it("refreshes once after a 401 and retries the request", async () => {
    setAccessToken("expired");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json(401, { success: false, error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockResolvedValueOnce(
        json(200, { success: true, data: { accessToken: "fresh", user: { id: "1" } } }),
      )
      .mockResolvedValueOnce(json(200, { success: true, data: { ok: true } }));
    stubFetch(fetchMock as unknown as typeof fetch);

    const data = await api<{ ok: boolean }>("/users");

    expect(data.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain("/auth/refresh");
    expect(getAccessToken()).toBe("fresh");
    const retryHeaders = new Headers(fetchMock.mock.calls[2]?.[1].headers);
    expect(retryHeaders.get("Authorization")).toBe("Bearer fresh");
  });

  it("clears the session when refresh fails", async () => {
    setAccessToken("expired");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json(401, { success: false, error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockResolvedValueOnce(json(401, { success: false, error: { code: "UNAUTHORIZED", message: "invalid refresh token" } }));
    stubFetch(fetchMock as unknown as typeof fetch);

    await expect(api("/users")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(getAccessToken()).toBeNull();
  });

  it("treats HTTP 204 as success without a JSON envelope", async () => {
    setAccessToken("token");
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    stubFetch(fetchMock as unknown as typeof fetch);
    await expect(api("/notifications/1/read", { method: "PATCH" })).resolves.toBeUndefined();
  });

  it("sends FormData without JSON content-type", async () => {
    setAccessToken("token");
    const fetchMock = vi.fn().mockResolvedValue(json(201, { success: true, data: [] }));
    stubFetch(fetchMock as unknown as typeof fetch);
    const body = new FormData();
    body.append("files", new Blob(["x"], { type: "image/jpeg" }), "shot.jpg");
    await api("/incidents/1/files", { method: "POST", body });
    const headers = new Headers(fetchMock.mock.calls[0]?.[1].headers);
    expect(headers.get("Content-Type")).toBeNull();
    expect(fetchMock.mock.calls[0]?.[1].body).toBeInstanceOf(FormData);
  });
});

describe("validateUploadFiles", () => {
  it("rejects more than 5 files", () => {
    const files = Array.from({ length: 6 }, (_, index) => new File(["a"], `a${index}.jpg`, { type: "image/jpeg" }));
    expect(validateUploadFiles(0, files)[0]).toMatch(/5 file/);
  });

  it("rejects a file larger than 10MB", () => {
    const big = new File([new Uint8Array(10 * 1024 * 1024 + 1)], "big.jpg", { type: "image/jpeg" });
    expect(validateUploadFiles(0, [big])[0]).toMatch(/10MB/);
  });
});

describe("getFileUrl", () => {
  it("refreshes once after a 403", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 403 }));
    stubFetch(fetchMock as unknown as typeof fetch);
    const refresh = vi.fn().mockResolvedValue("https://example.com/fresh");
    await expect(getFileUrl("https://example.com/expired", refresh)).resolves.toBe(
      "https://example.com/fresh",
    );
    expect(refresh).toHaveBeenCalledTimes(1);
  });
});
