import type { Site, User } from "@/lib/types";

export type AssigneeOption = { id: string; label: string };

export function assigneeOptions({
  session,
  site,
  reporterId,
  users,
}: {
  session?: User | null;
  site?: Site | null;
  reporterId?: string | null;
  users?: User[];
}): AssigneeOption[] {
  const map = new Map<string, string>();
  const add = (id: string | undefined | null, label: string) => {
    if (!id) return;
    if (!map.has(id)) map.set(id, label);
  };
  if (session) add(session.id, session.name);
  if (site) {
    add(site.supervisorId, `Supervisor ${site.code}`);
    add(site.hseOfficerId, `HSE Officer ${site.code}`);
  }
  if (reporterId) add(reporterId, "Pelapor");
  for (const user of users ?? []) {
    add(user.id, user.name);
  }
  return [...map.entries()].map(([id, label]) => ({ id, label }));
}
