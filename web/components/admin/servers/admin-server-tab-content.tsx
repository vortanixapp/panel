"use client";

import { AdminServerShell } from "@/components/admin/servers/admin-server-shell";
import { AdminServerAuditTab } from "@/components/admin/servers/tabs/audit-tab";
import { AdminServerLogsTab } from "@/components/admin/servers/tabs/logs-tab";
import { AdminServerNotesTab } from "@/components/admin/servers/tabs/notes-tab";
import { AdminServerOverviewTab } from "@/components/admin/servers/tabs/overview-tab";
import { AdminServerOwnerTab } from "@/components/admin/servers/tabs/owner-tab";
import { AdminServerRentTab } from "@/components/admin/servers/tabs/rent-tab";
import { AdminServerTechTab } from "@/components/admin/servers/tabs/tech-tab";
import { ServerConsolePanel } from "@/components/servers/server-console-content";
import { ServerSettingsBody } from "@/components/servers/server-settings-content";
import { ServerCopiesTab } from "@/features/servers/tabs/copies-tab";
import { ServerCronTab } from "@/features/servers/tabs/cron-tab";
import { ServerFirewallTab } from "@/features/servers/tabs/firewall-tab";
import { ServerFtpTab } from "@/features/servers/tabs/ftp-tab";
import { ServerMapsTab } from "@/features/servers/tabs/maps-tab";
import { ServerMetricsTab } from "@/features/servers/tabs/metrics-tab";
import { ServerMysqlTab } from "@/features/servers/tabs/mysql-tab";
import { ServerPluginsTab } from "@/features/servers/tabs/plugins-tab";
import { ServerPortsTab } from "@/features/servers/tabs/ports-tab";
import type { AdminServerTabKey } from "@/lib/server-tabs";
import { useT } from "@/hooks/use-translations";

const TAB_COMPONENTS: Record<string, React.ComponentType> = {
  logs: AdminServerLogsTab,
  rent: AdminServerRentTab,
  owner: AdminServerOwnerTab,
  tech: AdminServerTechTab,
  audit: AdminServerAuditTab,
  notes: AdminServerNotesTab,
  metrics: ServerMetricsTab,
  ftp: ServerFtpTab,
  mysql: ServerMysqlTab,
  cron: ServerCronTab,
  firewall: ServerFirewallTab,
  ports: ServerPortsTab,
  plugins: ServerPluginsTab,
  maps: ServerMapsTab,
  copies: ServerCopiesTab,
};

export function AdminServerTabContent({ tab }: { tab: string }) {
  const t = useT();
  const Tab = TAB_COMPONENTS[tab];

  if (!Tab) {
    return (
      <AdminServerShell>
        <p className="text-sm text-muted-foreground">{t("servers.tab.not_found")}</p>
      </AdminServerShell>
    );
  }

  return (
    <AdminServerShell activeTab={tab as AdminServerTabKey}>
      <Tab />
    </AdminServerShell>
  );
}

export function AdminServerDetailContent() {
  return (
    <AdminServerShell activeTab="main">
      <AdminServerOverviewTab />
    </AdminServerShell>
  );
}

export function AdminServerConsoleContent() {
  return (
    <AdminServerShell activeTab="console">
      <ServerConsolePanel />
    </AdminServerShell>
  );
}

export function AdminServerSettingsContent() {
  return (
    <AdminServerShell activeTab="settings">
      <ServerSettingsBody variant="admin" dangerZone={false} />
    </AdminServerShell>
  );
}
