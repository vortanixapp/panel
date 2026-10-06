"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_ROW_LINE,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import {
  fetchServerPlugins,
  installServerPlugin,
  toggleServerPlugin,
  uninstallServerPlugin,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { confirmAction } from "@/components/action-dialog";

export function ServerPluginsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [actionKey, setActionKey] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.serverPlugins(id),
    queryFn: () => fetchServerPlugins(id),
    enabled: !!id,
  });

  const rows = useMemo(() => {
    const items = data?.items ?? [];
    const q = search.trim().toLowerCase();
    return items
      .map((row) => ({
        id: row.plugin.id,
        name: row.plugin.name,
        version: row.plugin.version ?? "",
        category: row.plugin.category ?? t("servers.plugins.category_other"),
        installed: row.server_plugin.installed,
        enabled: row.server_plugin.enabled,
      }))
      .filter((r) => q === "" || r.name.toLowerCase().includes(q));
  }, [data?.items, search, t]);

  async function runAction(key: string, fn: () => Promise<unknown>) {
    setActionKey(key);
    try {
      await fn();
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverPlugins(id) });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setActionKey("");
    }
  }

  return (
    <Panel
      title={t("servers.plugins.title")}
      aside={
        <input
          className={cn(VX_INPUT, "h-[30px] w-[220px] rounded-[8px] text-[12px]")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("common.search_placeholder")}
        />
      }
      bodyClassName="flex flex-col px-6 py-1"
    >
      {isLoading ? (
        <VxInlineLoader />
      ) : rows.length === 0 ? (
        <EmptyState>
          {search ? t("common.not_found") : t("servers.plugins.empty")}
        </EmptyState>
      ) : (
        rows.map((row) => (
          <div
            key={row.id}
            className={cn(
              "grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2 py-[13px] last:border-b-0 sm:grid-cols-[minmax(0,1fr)_90px_120px_auto]",
              VX_ROW_LINE
            )}
          >
            <div className="min-w-0">
              <div className="truncate text-[13.5px]">{row.name}</div>
              <div className={cn("mt-[2px] truncate text-[11.5px] sm:hidden", VX_FAINT)}>
                {row.category}
                {row.version ? ` · v${row.version}` : ""}
              </div>
            </div>
            <span className="hidden font-mono text-[12.5px] sm:block">
              {row.version ? row.version : "—"}
            </span>
            <span className="hidden sm:block">
              <span
                className={cn(
                  "inline-block rounded-full px-[11px] py-[3px] text-[12px]",
                  !row.installed
                    ? "bg-[var(--vx-tint)] text-[var(--vx-muted)]"
                    : row.enabled
                      ? "bg-[var(--srv-accent-soft)] text-[var(--srv-accent)]"
                      : "bg-[var(--vx-tint)] text-[var(--vx-muted)]"
                )}
              >
                {row.installed
                  ? row.enabled
                    ? t("servers.plugins.on").replace(/^[s·]+/, "")
                    : t("servers.plugins.off").replace(/^[s·]+/, "")
                  : t("servers.plugins.not_installed").replace(/^[s·]+/, "")}
              </span>
            </span>
            <div className="flex shrink-0 gap-2">
              {!row.installed ? (
                <Btn
                  size="sm"
                  tone="primary"
                  disabled={!!actionKey}
                  onClick={() => runAction(`install:${row.id}`, () => installServerPlugin(id, row.id))}
                >
                  {t("servers.plugins.install")}
                </Btn>
              ) : (
                <>
                  <Btn
                    size="sm"
                    disabled={!!actionKey}
                    onClick={() =>
                      runAction(`toggle:${row.id}`, () => toggleServerPlugin(id, row.id, !row.enabled))
                    }
                  >
                    {row.enabled ? t("common.disable") : t("common.enable")}
                  </Btn>
                  <Btn
                    size="sm"
                    className="border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
                    disabled={!!actionKey}
                    onClick={async () => {
                      if (
                        !await confirmAction(
                          t("servers.plugins.delete_confirm", { name: row.name })
                        )
                      )
                        return;
                      void runAction(`uninstall:${row.id}`, () => uninstallServerPlugin(id, row.id));
                    }}
                  >
                    {t("common.delete")}
                  </Btn>
                </>
              )}
            </div>
          </div>
        ))
      )}
    </Panel>
  );
}
