import { IncidentDetailScreen } from "@/components/incidents/IncidentDetailScreen";

export default async function IncidentDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ incidentId: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { incidentId } = await params;
  const query = await searchParams;
  const tab = typeof query.tab === "string" ? query.tab : "overview";
  return <IncidentDetailScreen incidentId={incidentId} tab={tab} />;
}
