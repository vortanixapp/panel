import { AdminServerTabContent } from "@/components/admin/servers/admin-server-tab-content";

export default async function AdminServerTabPage({
  params,
}: {
  params: Promise<{ tab: string }>;
}) {
  const { tab } = await params;
  return <AdminServerTabContent tab={tab} />;
}
