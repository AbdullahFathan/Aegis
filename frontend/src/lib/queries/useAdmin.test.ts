import { describe, expect, it } from "vitest";

import { usersQueryKey } from "@/lib/types";

describe("usersQueryKey", () => {
  it("includes the filter object", () => {
    expect(usersQueryKey({ page: 1, pageSize: 20, role: "ADMIN" })).toEqual([
      "users",
      { page: 1, pageSize: 20, role: "ADMIN" },
    ]);
  });
});
