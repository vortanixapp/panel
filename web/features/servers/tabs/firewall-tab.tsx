"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Panel,
  VX_FAINT,
  VX_INPUT_MONO,
  VX_ROW_LINE,
  VX_SELECT,
  Toggle,
} from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createServerFirewallRule,
  deleteServerFirewallRule,
  fetchServerFirewall,
  toggleServerFirewallRule,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const GRID = "grid-cols-[90px_90px_minmax(0,1fr)_70px] sm:grid-cols-[110px_110px_minmax(0,1fr)_80px]";

export function ServerFirewallTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [protocol, setProtocol] = useState("tcp");
  const [port, setPort] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["server-firewall", id],
    queryFn: () => fetchServerFirewall(id),
    enabled: !!id,
  });

  const createMutation = useMutation({
    mutationFn: () => createServerFirewallRule(id, protocol, parseInt(port, 10)),
    onSuccess: () => {
      toast.success(t("servers.firewall.rule_added"));
      setPort("");
      void queryClient.invalidateQueries({ queryKey: ["server-firewall", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.firewall.create_error")
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (ruleId: string) => deleteServerFirewallRule(id, ruleId),
    onSuccess: () => {
      toast.success(t("servers.firewall.rule_deleted"));
      void queryClient.invalidateQueries({ queryKey: ["server-firewall", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.firewall.delete_error")
      ),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ ruleId, enabled }: { ruleId: string; enabled: boolean }) =>
      toggleServerFirewallRule(id, ruleId, enabled),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["server-firewall", id] }),
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.firewall.update_error")
      ),
  });

  if (isLoading) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;

  const rules = data?.rules ?? [];
  const portNum = parseInt(port, 10);
  const invalid = !port || Number.isNaN(portNum) || portNum < 1 || portNum > 65535;

  return (
    <Panel
      title={t("servers.firewall.title")}
      aside={
        <div className="flex flex-wrap gap-2">
          <select
            className={cn(VX_SELECT, "h-[30px] rounded-[8px] text-[12px]")}
            value={protocol}
            onChange={(e) => setProtocol(e.target.value)}
          >
            <option value="tcp">TCP</option>
            <option value="udp">UDP</option>
          </select>
          <input
            className={cn(VX_INPUT_MONO, "h-[30px] w-[110px] rounded-[8px] text-[12px]")}
            value={port}
            onChange={(e) => setPort(e.target.value)}
            inputMode="numeric"
            placeholder="25565"
          />
          <Btn
            size="sm"
            tone="primary"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending || invalid}
          >
            {t("common.add")}
          </Btn>
        </div>
      }
      flush
    >
      <div className="overflow-x-auto px-[18px] pt-1.5 pb-4">
        <div className="min-w-[400px]">
          <div
            className={cn(
              "grid gap-3 py-2.5 text-[10.5px] tracking-[0.08em] uppercase",
              GRID,
              VX_ROW_LINE,
              VX_FAINT
            )}
          >
            <span>{t("servers.firewall.col_protocol")}</span>
            <span>{t("servers.firewall.col_port")}</span>
            <span>{t("servers.firewall.col_enabled")}</span>
            <span />
          </div>

          {rules.length === 0 ? (
            <EmptyState>{t("servers.firewall.empty")}</EmptyState>
          ) : (
            rules.map((rule) => (
              <div
                key={rule.id}
                className={cn(
                  "grid items-center gap-3 py-2.5 font-mono text-[12px] last:border-b-0",
                  GRID,
                  VX_ROW_LINE
                )}
              >
                <span>{rule.protocol.toUpperCase()}</span>
                <span>{rule.port_from}</span>
                <Toggle
                  label={t("servers.firewall.rule_toggle")}
                  checked={rule.enabled}
                  disabled={toggleMutation.isPending}
                  onChange={(enabled) => toggleMutation.mutate({ ruleId: rule.id, enabled })}
                />
                <button
                  type="button"
                  disabled={deleteMutation.isPending}
                  onClick={() => {
                    if (!confirm(t("servers.firewall.delete_confirm"))) return;
                    deleteMutation.mutate(rule.id);
                  }}
                  className="justify-self-end font-sans text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:opacity-40"
                >
                  {t("common.delete")}
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </Panel>
  );
}
