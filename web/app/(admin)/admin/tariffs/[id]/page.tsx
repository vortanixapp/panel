import { TariffDetailContent } from "@/components/admin/tariffs/tariff-detail-content";

export default async function AdminTariffDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <TariffDetailContent id={id} />;
}
