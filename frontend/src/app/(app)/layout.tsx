import { AuthGate } from "@/components/layout/PageShell";

export default function AppLayout({ children }: LayoutProps<"/">) {
  return <AuthGate>{children}</AuthGate>;
}
