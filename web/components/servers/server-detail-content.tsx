import { ServerTabShell } from "@/components/servers/server-tab-shell";
import { ServerOverviewTab } from "@/features/servers/tabs/overview-tab";
import type { PanelVariant } from "@/lib/panel-paths";

export function ServerDetailContent({
  variant = "user",
}: {
  variant?: PanelVariant;
}) {
  return (
    <ServerTabShell variant={variant} activeTab="main">
      <ServerOverviewTab />
    </ServerTabShell>
  );
}
