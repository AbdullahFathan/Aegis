"use client";

import {
  ClipboardList,
  FileText,
  LayoutDashboard,
  MapPin,
  ScrollText,
  ShieldCheck,
  Users,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { canSeeNavItem, type NavItem } from "@/lib/rbac";
import { useSession } from "@/lib/queries/useSession";
import { cn } from "@/lib/utils";

const navItems: Array<{
  id: NavItem;
  href: string;
  label: string;
  icon: typeof LayoutDashboard;
}> = [
  { id: "dashboard", href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { id: "incidents", href: "/incidents", label: "Insiden", icon: ClipboardList },
  { id: "corrective-actions", href: "/corrective-actions", label: "CA Tracker", icon: ShieldCheck },
  { id: "reports", href: "/reports", label: "Laporan", icon: FileText },
  { id: "locations", href: "/locations", label: "Lokasi", icon: MapPin },
  { id: "users", href: "/admin/users", label: "Users", icon: Users },
  { id: "audit-logs", href: "/admin/audit-logs", label: "Audit log", icon: ScrollText },
];

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const session = useSession();
  const role = session.data?.role ?? "";

  return (
    <aside className="flex h-full min-h-screen w-60 shrink-0 flex-col bg-navy-800 text-white/70">
      <div className="flex h-14 items-center px-4">
        <span className="text-base font-semibold text-white">Aegis</span>
      </div>
      <nav className="flex flex-col gap-1 px-3 py-2">
        {navItems.filter((item) => canSeeNavItem(role, item.id)).map((item) => {
          const active =
            item.href === "/dashboard" ? pathname === item.href : pathname.startsWith(item.href);
          const Icon = item.icon;
          return (
            <Link
              key={item.id}
              href={item.href}
              onClick={() => onNavigate?.()}
              className={cn(
                "flex min-h-11 items-center gap-2 rounded-md px-3 py-2 text-sm",
                active ? "bg-orange-500 text-white" : "text-white/70 hover:bg-white/10 hover:text-white",
              )}
            >
              <Icon data-icon="inline-start" />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
