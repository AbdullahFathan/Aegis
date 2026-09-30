import { afterEach, describe, expect, it, vi } from "vitest";

import { api, getAccessToken, resetAuthState, setAccessToken } from "@/lib/api/client";

function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => {
  resetAuthState();
  vi.unstubAllGlobals();
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
    vi.stubGlobal("fetch", fetchMock);

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
    vi.stubGlobal("fetch", fetchMock);

    await expect(api("/users")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(getAccessToken()).toBeNull();
  });
});
