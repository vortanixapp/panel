"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Btn, EmptyState, Panel, VX_FAINT, VX_INPUT, VX_INSET } from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import {
  activateServerMap,
  fetchServerMaps,
  installServerMap,
  uninstallServerMap,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function ServerMapsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [actionKey, setActionKey] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.serverMaps(id),
    queryFn: () => fetchServerMaps(id),
    enabled: !!id,
  });

  const rows = useMemo(() => {
    const items = data?.items ?? [];
    const q = search.trim().toLowerCase();
    return items
      .map((row) => ({
        id: row.map.id,
        name: row.map.name,
        category: row.map.category ?? "",
        installed: row.server_map.installed,
        isActive: row.is_active ?? row.server_map.is_active,
      }))
      .filter((r) => q === "" || r.name.toLowerCase().includes(q));
  }, [data?.items, search]);

  async function runAction(key: string, fn: () => Promise<unknown>) {
    setActionKey(key);
    try {
      await fn();
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverMaps(id) });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setActionKey("");
    }
  }

  return (
    <Panel
      title={t("servers.maps.title")}
      aside={
        <input
          className={cn(VX_INPUT, "h-[30px] w-[220px] rounded-[8px] text-[12px]")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("common.search_placeholder")}
        />
      }
      bodyClassName="flex flex-col gap-2 px-[18px] py-3.5"
    >
      {isLoading ? (
        <VxInlineLoader />
      ) : rows.length === 0 ? (
        <EmptyState>
          {search ? t("common.not_found") : t("servers.maps.empty")}
        </EmptyState>
      ) : (
        rows.map((row) => (
          <div
            key={row.id}
            className={cn(
              "flex flex-wrap items-center justify-between gap-3 rounded-[11px] px-[15px] py-3",
              VX_INSET
            )}
          >
            <div className="min-w-0">
              <div className="flex items-center gap-2 text-[12.5px] font-medium">
                <span className="truncate">{row.name}</span>
                {row.isActive && (
                  <span className="shrink-0 rounded-full border border-[var(--vx-border-strong)] bg-[var(--vx-veil-strong)] px-2 py-0.5 text-[10.5px] text-[var(--vx-fg-strong)]">
                    {t("servers.maps.active")}
                  </span>
                )}
              </div>
              <div className={cn("mt-[3px] font-mono text-[11.5px]", VX_FAINT)}>
                {row.category || t("servers.maps.kind")}
                {row.installed
                  ? t("servers.maps.installed")
                  : t("servers.maps.not_installed")}
              </div>
            </div>
            <div className="flex shrink-0 gap-2">
              {!row.installed ? (
                <Btn
                  size="sm"
                  tone="primary"
                  disabled={!!actionKey}
                  onClick={() => runAction(`install:${row.id}`, () => installServerMap(id, row.id))}
                >
                  {t("servers.maps.install")}
                </Btn>
              ) : (
                <>
                  {!row.isActive && (
                    <Btn
                      size="sm"
                      disabled={!!actionKey}
                      onClick={() =>
                        runAction(`activate:${row.id}`, () => activateServerMap(id, row.id))
                      }
                    >
                      {t("servers.maps.activate")}
                    </Btn>
                  )}
                  <Btn
                    size="sm"
                    className="border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
                    disabled={!!actionKey}
                    onClick={() => {
                      if (
                        !confirm(
                          t("servers.maps.delete_confirm", { name: row.name })
                        )
                      )
                        return;
                      void runAction(`uninstall:${row.id}`, () => uninstallServerMap(id, row.id));
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
