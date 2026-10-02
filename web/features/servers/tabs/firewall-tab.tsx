"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Panel,
  VX_INPUT_MONO,
  VX_ROW_LINE,
  VX_SELECT,
  VX_TBL_TD,
  VX_TBL_TH,
  VX_TBL_WRAP,
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
import { confirmAction } from "@/components/action-dialog";

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
      <div className={VX_TBL_WRAP}>
        <table className="vx-tbl vx-tbl-flat w-full table-fixed border-collapse text-left">
          <thead>
            <tr>
              <th className={cn(VX_TBL_TH, "w-[110px]")}>{t("servers.firewall.col_protocol")}</th>
              <th className={cn(VX_TBL_TH, "w-[110px]")}>{t("servers.firewall.col_port")}</th>
              <th className={VX_TBL_TH}>{t("servers.firewall.col_enabled")}</th>
              <th className={cn(VX_TBL_TH, "w-[80px]")} />
            </tr>
          </thead>
          <tbody>
            {rules.length === 0 ? (
              <tr>
                <td colSpan={4}>
                  <EmptyState>{t("servers.firewall.empty")}</EmptyState>
                </td>
              </tr>
            ) : (
              rules.map((rule) => (
                <tr
                  key={rule.id}
                  className={cn("font-mono text-[12px] last:border-b-0", VX_ROW_LINE)}
                >
                  <td className={VX_TBL_TD} data-cell="lead">
                    {rule.protocol.toUpperCase()}
                  </td>
                  <td className={VX_TBL_TD} data-cell="full">
                    {rule.port_from}
                  </td>
                  <td className={VX_TBL_TD} data-label={t("servers.firewall.col_enabled")}>
                    <Toggle
                      label={t("servers.firewall.rule_toggle")}
                      checked={rule.enabled}
                      disabled={toggleMutation.isPending}
                      onChange={(enabled) => toggleMutation.mutate({ ruleId: rule.id, enabled })}
                    />
                  </td>
                  <td className={cn(VX_TBL_TD, "text-right")} data-cell="actions">
                    <button
                      type="button"
                      disabled={deleteMutation.isPending}
                      onClick={async () => {
                        if (!await confirmAction(t("servers.firewall.delete_confirm"))) return;
                        deleteMutation.mutate(rule.id);
                      }}
                      className="font-sans text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:opacity-40"
                    >
                      {t("common.delete")}
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}
