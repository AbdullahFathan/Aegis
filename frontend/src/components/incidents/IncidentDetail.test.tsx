import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusBadge } from "@/components/shared/StatusBadge";

describe("closed report badge copy", () => {
  it("uses the closed lifecycle label", () => {
    render(<StatusBadge status="CLOSED" />);
    expect(screen.getByText("Ditutup")).toBeInTheDocument();
  });
});
