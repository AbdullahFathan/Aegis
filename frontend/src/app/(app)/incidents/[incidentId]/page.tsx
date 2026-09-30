import { IncidentDetailScreen } from "@/components/incidents/IncidentDetailScreen";

export default async function IncidentDetailPage({
  params,
}: {
  params: Promise<{ incidentId: string }>;
}) {
  const { incidentId } = await params;
  return <IncidentDetailScreen incidentId={incidentId} />;
}
