import { ServerTabContent } from "@/components/servers/server-tab-content";

export default async function AdminServerTabPage({
  params,
}: {
  params: Promise<{ tab: string }>;
}) {
  const { tab } = await params;
  return <ServerTabContent tab={tab} variant="admin" />;
}
