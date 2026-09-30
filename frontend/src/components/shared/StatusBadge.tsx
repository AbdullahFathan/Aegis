"use client";

import { AlertCircle, AlertTriangle, Info, type LucideIcon } from "lucide-react";

import {
  caStatusConfig,
  componentClasses,
  type CorrectiveActionStatus,
  type IncidentStatus,
  type SeverityLevel,
  severityConfig,
  statusConfig,
} from "@/lib/tokens";

const severityIcons: Record<string, LucideIcon> = {
  AlertTriangle,
  AlertCircle,
  Info,
};

function ToneBadge({
  label,
  icon,
  background,
  border,
  text,
}: {
  label: string;
  icon?: string;
  background: string;
  border: string;
  text: string;
}) {
  const Icon = icon ? severityIcons[icon] : undefined;
  return (
    <span
      className={componentClasses.badge}
      data-icon={icon}
      style={{ backgroundColor: background, borderColor: border, color: text }}
    >
      {Icon ? <Icon aria-hidden /> : null}
      {label}
    </span>
  );
}

export function SeverityBadge({ severity }: { severity: SeverityLevel }) {
  const config = severityConfig[severity];
  return (
    <ToneBadge
      label={config.label}
      icon={config.icon}
      background={config.badgeBg}
      border={config.badgeBorder}
      text={config.badgeText}
    />
  );
}

export function StatusBadge({ status }: { status: IncidentStatus }) {
  const config = statusConfig[status];
  return (
    <ToneBadge
      label={config.label}
      background={config.badgeBg}
      border={config.badgeBorder}
      text={config.badgeText}
    />
  );
}

export function CaStatusBadge({ status }: { status: CorrectiveActionStatus }) {
  const config = caStatusConfig[status];
  return (
    <ToneBadge
      label={config.label}
      background={config.badgeBg}
      border={config.badgeBorder}
      text={config.badgeText}
    />
  );
}
