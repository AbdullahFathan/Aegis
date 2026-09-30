import { RCAWizard } from "@/components/forms/RCAWizard";

export default async function RcaPage({
  params,
}: {
  params: Promise<{ incidentId: string }>;
}) {
  const { incidentId } = await params;
  return <RCAWizard incidentId={incidentId} />;
}
