"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  EmptyState,
  Panel,
  VX_FAINT,
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
    >
      <div className="rounded-[10px] border border-[var(--vx-inset)] px-3.5 py-3">
        <p className={cn("text-[12px] leading-relaxed", VX_MUTED)}>
          {t("servers.ports.staff_only")}
        </p>
        <Link
          href="/support/create"
          className="mt-2 inline-block text-[11.5px] text-[var(--vx-accent)] transition-opacity hover:opacity-80"
        >
          {t("servers.ports.staff_only_cta")}
        </Link>
      </div>

      <div className="mt-4 overflow-hidden rounded-[10px] border border-[var(--vx-inset)]">
        {portsQuery.isLoading ? (
          <VxInlineLoader />
        ) : existing.length === 0 ? (
          <EmptyState>{t("servers.ports.empty")}</EmptyState>
        ) : (
          existing.map((p) => (
            <div
              key={p.id}
              className={cn(
                "flex items-center justify-between gap-3 px-3.5 py-2.5 last:border-b-0",
                VX_ROW_LINE
              )}
            >
              <span className="flex flex-wrap items-center gap-3 font-mono text-[12px]">
                {p.port}
                <span className={VX_FAINT}>{p.protocol.toUpperCase()}</span>
                {p.purpose && <span className={VX_FAINT}>{p.purpose}</span>}
              </span>
              {p.is_primary && (
                <span className="text-[11.5px] text-[var(--vx-warn)]">
                  {t("servers.ports.is_primary")}
                </span>
              )}
            </div>
          ))
        )}
      </div>
    </Panel>
  );
}
