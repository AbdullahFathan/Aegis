import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { resetAuthState, setAccessToken } from "@/lib/api/client";
import { useSubmitIncident } from "@/lib/queries/useIncidents";
import { incidentsQueryKey } from "@/lib/types";
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

describe("useSubmitIncident", () => {
  it("invalidates the incidents list query", async () => {
    setAccessToken("token");
    const fetchMock = vi.fn().mockResolvedValue(
      json(200, {
        success: true,
        data: { id: "inc-1", status: "PENDING_REVIEW", incidentNumber: "INC-2026-09-0001" },
      }),
    );
    stubFetch(fetchMock as unknown as typeof fetch);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const filters = { page: 1, pageSize: 20 };
    queryClient.setQueryData(incidentsQueryKey(filters), {
      items: [],
      page: 1,
      pageSize: 20,
      total: 0,
    });
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);
    const { result } = renderHook(() => useSubmitIncident(), { wrapper });
    await result.current.mutateAsync("inc-1");
    await waitFor(() => {
      const state = queryClient.getQueryState(incidentsQueryKey(filters));
      expect(state?.isInvalidated).toBe(true);
    });
  });
});
