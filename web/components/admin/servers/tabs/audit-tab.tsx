"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Btn, EmptyState, Panel, VX_CODE, VX_INPUT, VX_SELECT } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminServerAudit } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { auditActionLabel } from "@/components/admin/servers/labels";

const LIMITS = [50, 100, 250, 500];

export function AdminServerAuditTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();

  const [action, setAction] = useState("");
  const [search, setSearch] = useState("");
  const [limit, setLimit] = useState(100);
  const [expanded, setExpanded] = useState<string | null>(null);

  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: [...queryKeys.adminServerAudit(id), action, search, limit],
    queryFn: () => fetchAdminServerAudit(id, { action, search, limit }),
    enabled: !!id,
  });

  const entries = data?.entries ?? [];

  return (
    <Panel
      title={t("servers.admin.audit.title")}
      aside={
        <Btn size="sm" onClick={() => refetch()} disabled={isFetching}>
          {isFetching ? t("common.updating") : t("common.refresh")}
        </Btn>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <input
          className={cn(VX_INPUT, "h-[32px] w-[220px]")}
          placeholder={t("servers.admin.audit.search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select
          className={cn(VX_SELECT, "h-[32px]")}
          value={action}
          onChange={(e) => setAction(e.target.value)}
        >
          <option value="">{t("servers.admin.audit.all_actions")}</option>
          {(data?.actions ?? []).map((a) => (
            <option key={a} value={a}>
              {auditActionLabel(a)}
            </option>
          ))}
        </select>
        <select
          className={cn(VX_SELECT, "h-[32px]")}
          value={limit}
          onChange={(e) => setLimit(Number(e.target.value))}
        >
          {LIMITS.map((n) => (
            <option key={n} value={n}>
              {t("servers.admin.audit.limit", { count: n })}
            </option>
          ))}
        </select>
      </div>

      {isLoading ? (
        <Skeleton className="h-[320px] w-full rounded-[10px]" />
      ) : entries.length === 0 ? (
        <EmptyState>{t("servers.admin.audit.empty")}</EmptyState>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] border-collapse text-[12.5px]">
            <thead>
              <tr className="border-b border-[var(--vx-border)] text-left text-[var(--vx-muted)]">
                <th className="py-2 pr-3 font-medium">{t("servers.admin.audit.col_when")}</th>
                <th className="py-2 pr-3 font-medium">{t("servers.admin.audit.col_who")}</th>
                <th className="py-2 pr-3 font-medium">{t("servers.admin.audit.col_action")}</th>
                <th className="py-2 font-medium">{t("servers.admin.audit.col_details")}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => {
                const hasMeta = Object.keys(entry.meta ?? {}).length > 0;
                const open = expanded === entry.id;
                return (
                  <tr
                    key={entry.id}
                    className="border-b border-[var(--vx-elevated)] align-top"
                  >
                    <td className="py-2 pr-3 whitespace-nowrap text-[var(--vx-muted)]">
                      {new Date(entry.created_at).toLocaleString(localeTag())}
                    </td>
                    <td className="py-2 pr-3">
                      {entry.actor_email ?? t("servers.admin.audit.system")}
                    </td>
                    <td className="py-2 pr-3">{auditActionLabel(entry.action)}</td>
                    <td className="py-2">
                      {hasMeta ? (
                        <>
                          <button
                            type="button"
                            className="text-[var(--vx-muted)] underline-offset-2 hover:underline"
                            onClick={() => setExpanded(open ? null : entry.id)}
                          >
                            {open
                              ? t("servers.admin.audit.hide_details")
                              : t("servers.admin.audit.show_details")}
                          </button>
                          {open && (
                            <pre
                              className={cn(
                                "mt-2 overflow-x-auto rounded-[8px] p-2 font-mono text-[11.5px] whitespace-pre-wrap text-[var(--vx-dim)]",
                                VX_CODE
                              )}
                            >
                              {JSON.stringify(entry.meta, null, 2)}
                            </pre>
                          )}
                        </>
                      ) : (
                        <span className="text-[var(--vx-faint)]">—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}
