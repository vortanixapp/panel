"use client";

import type { PanelVariant } from "@/lib/panel-paths";
import type { ServerTabKey } from "@/lib/server-tabs";
import { useT } from "@/hooks/use-translations";
import { ServerTabShell } from "@/components/servers/server-tab-shell";
import { ServerCronTab } from "@/features/servers/tabs/cron-tab";
import { ServerCopiesTab } from "@/features/servers/tabs/copies-tab";
import { ServerFirewallTab } from "@/features/servers/tabs/firewall-tab";
import { ServerFriendsTab } from "@/features/servers/tabs/friends-tab";
import { ServerFtpTab } from "@/features/servers/tabs/ftp-tab";
import { ServerLogsTab } from "@/features/servers/tabs/logs-tab";
import { ServerMapsTab } from "@/features/servers/tabs/maps-tab";
import { ServerMetricsTab } from "@/features/servers/tabs/metrics-tab";
import { ServerMysqlTab } from "@/features/servers/tabs/mysql-tab";
import { ServerPluginsTab } from "@/features/servers/tabs/plugins-tab";
import { ServerPortsTab } from "@/features/servers/tabs/ports-tab";
import { ServerTariffTab } from "@/features/servers/tabs/tariff-tab";

const TAB_COMPONENTS: Record<string, React.ComponentType> = {
  logs: ServerLogsTab,
  metrics: ServerMetricsTab,
  ftp: ServerFtpTab,
  mysql: ServerMysqlTab,
  cron: ServerCronTab,
  firewall: ServerFirewallTab,
  ports: ServerPortsTab,
  tariff: ServerTariffTab,
  plugins: ServerPluginsTab,
  maps: ServerMapsTab,
  copies: ServerCopiesTab,
  friends: ServerFriendsTab,
};

export function ServerTabContent({
  tab,
  variant = "user",
}: {
  tab: string;
  variant?: PanelVariant;
}) {
  const t = useT();
  const Tab = TAB_COMPONENTS[tab];

  if (!Tab) {
    return (
      <ServerTabShell variant={variant}>
        <p className="text-sm text-muted-foreground">
          {t("servers.tab.not_found")}
        </p>
      </ServerTabShell>
    );
  }

  return (
    <ServerTabShell variant={variant} activeTab={tab as ServerTabKey}>
      <Tab />
    </ServerTabShell>
  );
}
