"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_INPUT_MONO,
  VX_MUTED,
  VX_ROW_LINE,
  VX_SELECT,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { useServerDetail } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import {
  createServerPort,
  deleteServerPort,
  fetchServerPorts,
  type ServerPortEntry,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";

export function ServerPortsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);
  const [port, setPort] = useState("");
  const [protocol, setProtocol] = useState<"tcp" | "udp" | "both">("udp");
  const [purpose, setPurpose] = useState("game");

  const portsQuery = useQuery({
    queryKey: queryKeys.serverPorts(id),
    queryFn: () => fetchServerPorts(id),
    enabled: !!id,
  });

  const createMutation = useMutation({
    mutationFn: () =>
      createServerPort(id, { port: Number(port), protocol, purpose: purpose.trim() || "game" }),
    onSuccess: () => {
      toast.success(t("servers.ports.added"));
      setPort("");
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverPorts(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.ports.add_error")
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (portId: string) => deleteServerPort(id, portId),
    onSuccess: () => {
      toast.success(t("servers.ports.deleted"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverPorts(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.ports.delete_error")
      ),
  });

  const addr = server?.ip_address && server?.port ? `${server.ip_address}:${server.port}` : "—";
  const existing = useMemo<ServerPortEntry[]>(
    () =>
      portsQuery.data?.ports ??
      ((server as { extra_ports?: ServerPortEntry[] } | undefined)?.extra_ports ?? []),
    [portsQuery.data?.ports, server]
  );

  const portNum = Number(port);
  const invalidPort = !Number.isFinite(portNum) || portNum < 1 || portNum > 65535;
  const localConflict = existing.some(
    (p) => p.port === portNum && (p.protocol === "both" || protocol === "both" || p.protocol === protocol)
  );
  const primaryConflict = Number(server?.port) > 0 && Number(server?.port) === portNum;
  const error =
    port && invalidPort
      ? t("servers.ports.invalid")
      : primaryConflict
        ? t("servers.ports.primary_conflict")
        : localConflict
          ? t("servers.ports.conflict")
          : "";

  return (
    <Panel
      title={t("servers.ports.title")}
      aside={
        <span className={cn("font-mono text-[12px]", VX_MUTED)}>
          {t("servers.ports.primary", { addr })}
        </span>
      }
    >
      <div className="grid items-end gap-2.5 sm:grid-cols-[120px_110px_minmax(0,1fr)_auto]">
        <Field label={t("servers.ports.port")}>
          <input
            className={VX_INPUT_MONO}
            value={port}
            onChange={(e) => setPort(e.target.value)}
            inputMode="numeric"
            placeholder="27015"
          />
        </Field>
        <Field label={t("servers.ports.protocol")}>
          <select
            className={VX_SELECT}
            value={protocol}
            onChange={(e) => setProtocol(e.target.value as "tcp" | "udp" | "both")}
          >
            <option value="udp">UDP</option>
            <option value="tcp">TCP</option>
            <option value="both">Both</option>
          </select>
        </Field>
        <Field label={t("servers.ports.purpose")}>
          <input
            className={VX_INPUT}
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
            placeholder="query/rcon/game"
          />
        </Field>
        <Btn
          tone="primary"
          className="px-4"
          onClick={() => createMutation.mutate()}
          disabled={
            createMutation.isPending || !port || invalidPort || localConflict || primaryConflict
          }
        >
          {createMutation.isPending
            ? t("servers.ports.adding")
            : t("servers.ports.add")}
        </Btn>
      </div>

      {error && <div className="mt-2.5 text-[11.5px] text-[var(--vx-danger)]">{error}</div>}

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
                {p.is_primary && <span className="text-[var(--vx-warn)]">primary</span>}
              </span>
              <button
                type="button"
                disabled={deleteMutation.isPending || p.is_primary}
                onClick={() => {
                  if (
                    !confirm(
                      t("servers.ports.delete_confirm", {
                        port: p.port,
                        protocol: p.protocol,
                      })
                    )
                  )
                    return;
                  deleteMutation.mutate(p.id);
                }}
                className="text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40"
              >
                {p.is_primary ? t("servers.ports.is_primary") : t("common.delete")}
              </button>
            </div>
          ))
        )}
      </div>
    </Panel>
  );
}
