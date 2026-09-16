"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Btn, EmptyState, InfoRow, Panel, VX_CODE } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { AssignIPDialog } from "@/components/admin/servers/assign-ip-dialog";
import { MigrateServerDialog } from "@/components/admin/servers/migrate-server-dialog";
import { fetchAdminServerCard, fetchServerMigrations } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useServerDetail } from "@/hooks/use-queries";

type PortEntry = { port?: number; protocol?: string; purpose?: string };

export function AdminServerTechTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const { data: server } = useServerDetail(id);

  const [ipOpen, setIpOpen] = useState(false);
  const [migrateOpen, setMigrateOpen] = useState(false);

  const cardQuery = useQuery({
    queryKey: queryKeys.adminServerCard(id),
    queryFn: () => fetchAdminServerCard(id),
    enabled: !!id,
  });
  const migrationsQuery = useQuery({
    queryKey: ["admin-servers", id, "migrations"],
    queryFn: () => fetchServerMigrations(id),
    enabled: !!id,
  });

  if (cardQuery.isLoading || !cardQuery.data) {
    return <Skeleton className="h-[420px] w-full rounded-[14px]" />;
  }
  const card = cardQuery.data;
  const dash = "—";
  const ports = (card.ports ?? []) as PortEntry[];
  const migrations = migrationsQuery.data?.migrations ?? [];

  return (
    <div className="grid gap-[18px] lg:grid-cols-2">
      <Panel
        title={t("servers.admin.tech.section_node")}
        aside={
          <Btn size="sm" onClick={() => setMigrateOpen(true)}>
            {t("servers.admin.migrate")}
          </Btn>
        }
      >
        <InfoRow k={t("servers.admin.card.node")} v={card.node.name || dash} />
        <InfoRow k="FQDN" v={card.node.fqdn || dash} />
        <InfoRow k={t("common.status")} v={card.node.status || dash} />
        <InfoRow k="ID" v={<span className="font-mono text-[11.5px]">{card.node.id}</span>} />
      </Panel>

      <Panel
        title={t("servers.admin.tech.section_network")}
        aside={
          <Btn size="sm" onClick={() => setIpOpen(true)}>
            {t("servers.admin.actions.ip")}
          </Btn>
        }
      >
        <InfoRow k={t("servers.admin.list.col_address")} v={server?.ip_address || dash} />
        <InfoRow
          k={t("servers.admin.tech.primary_port")}
          v={server?.port ? String(server.port) : dash}
        />
        {ports.length === 0 ? (
          <InfoRow k={t("servers.admin.tech.extra_ports")} v={dash} />
        ) : (
          ports.map((port, index) => (
            <InfoRow
              key={`${port.port}-${index}`}
              k={`${t("servers.admin.tech.extra_ports")} ${index + 1}`}
              v={`${port.port ?? dash}${port.protocol ? ` / ${port.protocol}` : ""}${
                port.purpose ? ` · ${port.purpose}` : ""
              }`}
            />
          ))
        )}
      </Panel>

      <Panel title={t("servers.admin.tech.section_container")}>
        <InfoRow
          k={t("servers.admin.tech.container_name")}
          v={
            card.container_name ? (
              <span className="font-mono text-[11.5px]">{card.container_name}</span>
            ) : (
              dash
            )
          }
        />
        <InfoRow
          k={t("servers.admin.tech.container_id")}
          v={
            card.container_id ? (
              <span className="font-mono text-[11.5px]">{card.container_id.slice(0, 16)}</span>
            ) : (
              dash
            )
          }
        />
        <InfoRow
          k={t("servers.admin.card.provisioning")}
          v={t(`servers.admin.card.prov_${card.provisioning_status}`)}
        />
        <InfoRow
          k={t("servers.admin.tech.steam_updatable")}
          v={card.steam_updatable ? t("common.yes") : t("common.no")}
        />
        <InfoRow
          k={t("servers.admin.tech.auto_start")}
          v={card.auto_start ? t("common.yes") : t("common.no")}
        />
        {card.provisioning_error && (
          <InfoRow
            k={t("servers.admin.tech.prov_error")}
            v={<span className="text-[var(--vx-danger)]">{card.provisioning_error}</span>}
          />
        )}
      </Panel>

      <Panel title={t("servers.admin.tech.section_limits")}>
        <pre
          className={cn(
            "m-0 max-h-[260px] overflow-auto rounded-[10px] p-3 font-mono text-[11.5px] whitespace-pre-wrap text-[var(--vx-dim)]",
            VX_CODE
          )}
        >
          {JSON.stringify(card.limits, null, 2)}
        </pre>
      </Panel>

      <div className="lg:col-span-2">
        <Panel title={t("servers.admin.tech.section_migrations")}>
          {migrations.length === 0 ? (
            <EmptyState>{t("servers.admin.tech.migrations_empty")}</EmptyState>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px] border-collapse text-[12.5px]">
                <thead>
                  <tr className="border-b border-[var(--vx-border)] text-left text-[var(--vx-muted)]">
                    <th className="py-2 pr-3 font-medium">
                      {t("servers.admin.audit.col_when")}
                    </th>
                    <th className="py-2 pr-3 font-medium">
                      {t("servers.admin.tech.migration_route")}
                    </th>
                    <th className="py-2 pr-3 font-medium">{t("common.status")}</th>
                    <th className="py-2 font-medium">{t("servers.admin.tech.migration_stage")}</th>
                  </tr>
                </thead>
                <tbody>
                  {migrations.map((migration) => (
                    <tr
                      key={migration.id}
                      className="border-b border-[var(--vx-elevated)]"
                    >
                      <td className="py-2 pr-3 whitespace-nowrap text-[var(--vx-muted)]">
                        {new Date(migration.created_at).toLocaleString(localeTag())}
                      </td>
                      <td className="py-2 pr-3">
                        {migration.from_node || dash} → {migration.to_node || dash}
                      </td>
                      <td className="py-2 pr-3">{migration.status}</td>
                      <td className="py-2">
                        {migration.error ? (
                          <span className="text-[var(--vx-danger)]">{migration.error}</span>
                        ) : (
                          migration.stage || dash
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Panel>
      </div>

      {ipOpen && (
        <AssignIPDialog
          serverId={id}
          serverName={card.name}
          nodeId={card.node.id}
          open
          onOpenChange={setIpOpen}
        />
      )}
      {migrateOpen && (
        <MigrateServerDialog
          serverId={id}
          serverName={card.name}
          open
          onOpenChange={setMigrateOpen}
        />
      )}
    </div>
  );
}
