import type { DashboardServer } from "@/lib/api";
import { t } from "@/lib/i18n";
import { isServerExpired, isServerProvisioning, serverProvisioning } from "@/lib/server-lifecycle";

export type ServerTabKey =
  | "main"
  | "console"
  | "logs"
  | "metrics"
  | "ftp"
  | "mysql"
  | "cron"
  | "firewall"
  | "ports"
  | "settings"
  | "tariff"
  | "plugins"
  | "maps"
  | "copies"
  | "friends";

type ServerTabDef = { key: ServerTabKey; suffix: string; label: string };

const SERVER_TAB_SUFFIXES: { key: ServerTabKey; suffix: string }[] = [
  { key: "main", suffix: "" },
  { key: "console", suffix: "/console" },
  { key: "logs", suffix: "/logs" },
  { key: "metrics", suffix: "/metrics" },
  { key: "ftp", suffix: "/ftp" },
  { key: "mysql", suffix: "/mysql" },
  { key: "cron", suffix: "/cron" },
  { key: "firewall", suffix: "/firewall" },
  { key: "ports", suffix: "/ports" },
  { key: "settings", suffix: "/settings" },
  { key: "tariff", suffix: "/tariff" },
  { key: "plugins", suffix: "/plugins" },
  { key: "maps", suffix: "/maps" },
  { key: "copies", suffix: "/copies" },
  { key: "friends", suffix: "/friends" },
];

// Подписи собираются на каждый вызов, а не лежат в константе модуля: язык
// и переопределения хостера приходят уже после импорта, и готовые строки
// застыли бы на том, что стояло в момент загрузки страницы.
export function serverTabDefs(): ServerTabDef[] {
  return SERVER_TAB_SUFFIXES.map(({ key, suffix }) => ({
    key,
    suffix,
    label: t(`server.tab.${key}`),
  }));
}

function gameSlug(server: DashboardServer): string {
  return (server.game?.slug || server.game_id || "").toLowerCase();
}

export function isCs16Server(server: DashboardServer): boolean {
  return ["cs16", "cstrike", "counter_strike", "cs_1_6"].includes(gameSlug(server));
}

export function canViewServerTab(server: DashboardServer, tab: ServerTabKey, isOwner: boolean): boolean {
  if (tab === "main") return true;
  if (tab === "tariff") return isOwner;
  if (tab === "maps" && !isCs16Server(server)) return false;

  const perms = server.viewer_permissions ?? {};
  const keyByTab: Partial<Record<ServerTabKey, keyof typeof perms>> = {
    console: "can_view_console",
    logs: "can_view_logs",
    metrics: "can_view_metrics",
    ftp: "can_view_ftp",
    mysql: "can_view_mysql",
    cron: "can_view_cron",
    firewall: "can_view_firewall",
    ports: "can_view_ports",
    settings: "can_view_settings",
    plugins: "can_settings_edit",
    maps: "can_settings_edit",
    copies: "can_settings_edit",
    friends: "can_view_friends",
  };

  const permKey = keyByTab[tab];
  if (!permKey) return true;
  const value = perms[permKey as keyof typeof perms];
  return value !== false;
}

export function isServerTabDisabled(server: DashboardServer, tab: ServerTabKey): boolean {
  if (tab === "main") return false;

  const expired = isServerExpired(server);
  const provisioning = isServerProvisioning(server);
  const prov = serverProvisioning(server);
  const installFailed = prov === "failed";

  if (tab === "tariff") return provisioning || installFailed || expired;
  if (
    ["console", "metrics", "ftp", "mysql", "cron", "firewall", "settings", "maps", "copies", "ports"].includes(
      tab
    )
  ) {
    return provisioning || installFailed || expired;
  }
  if (tab === "logs") return installFailed || expired;
  if (tab === "plugins" || tab === "friends") return provisioning || installFailed || expired;
  return false;
}
