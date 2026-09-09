import { TariffEditContent } from "@/components/admin/tariffs/tariff-edit-content";

export default async function AdminTariffEditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <TariffEditContent id={id} />;
}
