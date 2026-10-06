"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  EmptyState,
  Panel,
  VX_MUTED,
  VX_ROW_LINE,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { useServerDetail } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { fetchServerPorts, type ServerPortEntry } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";

export function ServerPortsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const { data: server } = useServerDetail(id);

  const portsQuery = useQuery({
    queryKey: queryKeys.serverPorts(id),
    queryFn: () => fetchServerPorts(id),
    enabled: !!id,
  });

  const addr = server?.ip_address && server?.port ? `${server.ip_address}:${server.port}` : "—";
  const existing = useMemo<ServerPortEntry[]>(
    () =>
      portsQuery.data?.ports ??
      ((server as { extra_ports?: ServerPortEntry[] } | undefined)?.extra_ports ?? []),
    [portsQuery.data?.ports, server]
  );

  return (
    <Panel
      title={t("servers.ports.title")}
      aside={
        <span className={cn("font-mono text-[12px]", VX_MUTED)}>
          {t("servers.ports.primary", { addr })}
        </span>
      }
      flush
    >
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--vx-border)] px-6 py-3.5">
        <p className={cn("text-[12.5px] leading-relaxed", VX_MUTED)}>
          {t("servers.ports.staff_only")}
        </p>
        <Link
          href="/support/create"
          className="text-[12px] text-[var(--srv-accent)] transition-opacity hover:opacity-80"
        >
          {t("servers.ports.staff_only_cta")}
        </Link>
      </div>

      {portsQuery.isLoading ? (
        <VxInlineLoader />
      ) : existing.length === 0 ? (
        <EmptyState>{t("servers.ports.empty")}</EmptyState>
      ) : (
        <div className="overflow-x-auto">
          <div className="min-w-[520px]">
            <div className="grid grid-cols-[110px_100px_minmax(0,1fr)_120px] gap-3 border-b border-[var(--vx-border)] px-6 py-3 text-[11px] tracking-[0.08em] text-[var(--vx-ghost)] uppercase">
              <span>{t("servers.firewall.col_port")}</span>
              <span>{t("servers.firewall.col_protocol")}</span>
              <span>{t("servers.ports.col_purpose")}</span>
              <span>{t("servers.ports.col_status")}</span>
            </div>
            {existing.map((p) => (
              <div
                key={p.id}
                className={cn(
                  "grid grid-cols-[110px_100px_minmax(0,1fr)_120px] items-center gap-3 px-6 py-[13px] text-[13px] last:border-b-0",
                  VX_ROW_LINE
                )}
              >
                <span className="font-mono text-[12.5px]">{p.port}</span>
                <span className="font-mono text-[12.5px]">{p.protocol.toUpperCase()}</span>
                <span className="truncate">{p.purpose || "—"}</span>
                <span>
                  <span
                    className={cn(
                      "inline-block rounded-full px-[11px] py-[3px] text-[12px]",
                      p.is_primary
                        ? "bg-[var(--srv-accent-soft)] text-[var(--srv-accent)]"
                        : "bg-[color-mix(in_oklab,var(--srv-info)_14%,transparent)] text-[var(--srv-info)]"
                    )}
                  >
                    {p.is_primary ? t("servers.ports.is_primary") : t("servers.ports.status_open")}
                  </span>
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </Panel>
  );
}
