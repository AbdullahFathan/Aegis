import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SeverityBadge, StatusBadge } from "@/components/shared/StatusBadge";
import { colors } from "@/lib/tokens";

describe("badges", () => {
  it("renders critical severity with the danger icon", () => {
    render(<SeverityBadge severity="CRITICAL" />);
    const badge = screen.getByText("Kritis").closest("span");
    expect(badge).toHaveAttribute("data-icon", "AlertTriangle");
    expect(badge).toHaveStyle({ color: colors.dangerText });
  });

  it("renders pending review with the navy badge", () => {
    render(<StatusBadge status="PENDING_REVIEW" />);
    const badge = screen.getByText("Menunggu review").closest("span");
    expect(badge).toHaveStyle({ backgroundColor: colors.navy[50] });
    expect(screen.queryByText("IN_REVIEW")).not.toBeInTheDocument();
  });
});
