"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  FormRow,
  Panel,
  VX_FAINT,
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
  setServerFirewallLimit,
  toggleServerFirewallRule,
  type FirewallPort,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { confirmAction } from "@/components/action-dialog";

const IPV4 = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;

function portKey(p: FirewallPort) {
  return `${p.protocol}:${p.port}`;
}

export function ServerFirewallTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [action, setAction] = useState<"deny" | "allow">("deny");
  const [portChoice, setPortChoice] = useState("");
  const [source, setSource] = useState("");
  const [limitInput, setLimitInput] = useState("0");

  const { data, isLoading } = useQuery({
    queryKey: ["server-firewall", id],
    queryFn: () => fetchServerFirewall(id),
    enabled: !!id,
  });

  const ports = data?.ports ?? [];
  const rules = data?.rules ?? [];
  const connLimit = data?.conn_limit ?? 0;

  useEffect(() => {
    setLimitInput(String(connLimit));
  }, [connLimit]);

  useEffect(() => {
    if (!portChoice && ports.length > 0) setPortChoice(portKey(ports[0]));
  }, [ports, portChoice]);

  const refresh = () => void queryClient.invalidateQueries({ queryKey: ["server-firewall", id] });

  const selected = ports.find((p) => portKey(p) === portChoice);
  const trimmedSource = source.trim();
  const sourceInvalid = trimmedSource !== "" && !IPV4.test(trimmedSource);
  const needsSource = action === "allow" && trimmedSource === "";
  const closesPrimary = action === "deny" && trimmedSource === "" && !!selected?.primary;
  const cannotAdd = !selected || sourceInvalid || needsSource || closesPrimary;

  const createMutation = useMutation({
    mutationFn: () =>
      createServerFirewallRule(id, {
        protocol: selected?.protocol ?? "tcp",
        port_from: selected?.port ?? 0,
        action,
        source: trimmedSource,
      }),
    onSuccess: () => {
      toast.success(t("servers.firewall.rule_added"));
      setSource("");
      refresh();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("servers.firewall.create_error")),
  });

  const deleteMutation = useMutation({
    mutationFn: (ruleId: string) => deleteServerFirewallRule(id, ruleId),
    onSuccess: () => {
      toast.success(t("servers.firewall.rule_deleted"));
      refresh();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("servers.firewall.delete_error")),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ ruleId, enabled }: { ruleId: string; enabled: boolean }) =>
      toggleServerFirewallRule(id, ruleId, enabled),
    onSuccess: refresh,
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("servers.firewall.update_error")),
  });

  const limitMutation = useMutation({
    mutationFn: (value: number) => setServerFirewallLimit(id, value),
    onSuccess: () => {
      toast.success(t("servers.firewall.limit_saved"));
      refresh();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("servers.firewall.update_error")),
  });

  if (isLoading) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;

  const limitValue = parseInt(limitInput, 10);
  const limitInvalid = Number.isNaN(limitValue) || limitValue < 0 || limitValue > 1000;

  return (
    <div className="flex flex-col gap-[18px]">
      <div className="grid grid-cols-1 items-start gap-[18px] lg:grid-cols-2">
        <Panel title={t("servers.firewall.title")} bodyClassName="px-6 py-1">
          <p className={cn("pt-3 pb-1 text-[12.5px] leading-[1.5]", VX_FAINT)}>
            {t("servers.firewall.intro")}
          </p>
          <FormRow label={t("servers.firewall.col_action")}>
            <select
              className={cn(VX_SELECT, "h-[38px] rounded-[10px] text-[13px]")}
              value={action}
              onChange={(e) => setAction(e.target.value as "deny" | "allow")}
            >
              <option value="deny">{t("servers.firewall.action_deny")}</option>
              <option value="allow">{t("servers.firewall.action_allow")}</option>
            </select>
          </FormRow>
          <FormRow label={t("servers.firewall.col_port")}>
            <select
              className={cn(VX_SELECT, "h-[38px] min-w-[200px] rounded-[10px] text-[13px]")}
              value={portChoice}
              onChange={(e) => setPortChoice(e.target.value)}
              disabled={ports.length === 0}
            >
              {ports.length === 0 && <option value="">{t("servers.firewall.no_ports")}</option>}
              {ports.map((p) => (
                <option key={portKey(p)} value={portKey(p)}>
                  {p.protocol.toUpperCase()} {p.port}
                  {p.purpose ? ` · ${p.purpose}` : ""}
                </option>
              ))}
            </select>
          </FormRow>
          <FormRow label={t("servers.firewall.col_source")}>
            <input
              className={cn(VX_INPUT_MONO, "h-[38px] w-[240px] max-w-full rounded-[10px] text-[13px]")}
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder={
                action === "allow"
                  ? t("servers.firewall.source_allow_placeholder")
                  : t("servers.firewall.source_deny_placeholder")
              }
            />
          </FormRow>
          {(sourceInvalid || needsSource || closesPrimary) && (
            <p className="pt-3 text-[12px] text-[var(--vx-danger)]">
              {sourceInvalid
                ? t("servers.firewall.source_invalid")
                : needsSource
                  ? t("servers.firewall.source_required")
                  : t("servers.firewall.primary_warning")}
            </p>
          )}
          {action === "allow" && selected?.primary && !sourceInvalid && !needsSource && (
            <p className={cn("pt-3 text-[12px]", VX_FAINT)}>{t("servers.firewall.primary_allow_note")}</p>
          )}
          <div className="flex justify-end py-4">
            <Btn
              tone="primary"
              onClick={() => createMutation.mutate()}
              disabled={createMutation.isPending || cannotAdd}
            >
              {t("common.add")}
            </Btn>
          </div>
        </Panel>

        <Panel title={t("servers.firewall.limit_title")} bodyClassName="px-6 py-1">
          <FormRow label={t("servers.firewall.limit_title")} hint={t("servers.firewall.limit_hint")}>
            <input
              className={cn(VX_INPUT_MONO, "h-[38px] w-[110px] rounded-[10px] text-[13px]")}
              value={limitInput}
              onChange={(e) => setLimitInput(e.target.value)}
              inputMode="numeric"
            />
            <Btn
              tone="primary"
              onClick={() => limitMutation.mutate(limitValue)}
              disabled={limitMutation.isPending || limitInvalid || limitValue === connLimit}
            >
              {t("common.save")}
            </Btn>
          </FormRow>
          <p className={cn("py-3.5 text-[12.5px]", VX_FAINT)}>
            {connLimit > 0
              ? t("servers.firewall.limit_on", { count: connLimit })
              : t("servers.firewall.limit_off")}
          </p>
        </Panel>
      </div>

      <Panel title={t("servers.firewall.rules_title")} flush>
        <div className={VX_TBL_WRAP}>
          <table className="vx-tbl vx-tbl-flat w-full table-fixed border-collapse text-left">
            <thead>
              <tr>
                <th className={cn(VX_TBL_TH, "w-[150px]")}>{t("servers.firewall.col_action")}</th>
                <th className={cn(VX_TBL_TH, "w-[90px]")}>{t("servers.firewall.col_protocol")}</th>
                <th className={cn(VX_TBL_TH, "w-[110px]")}>{t("servers.firewall.col_port")}</th>
                <th className={VX_TBL_TH}>{t("servers.firewall.col_source")}</th>
                <th className={cn(VX_TBL_TH, "w-[90px]")}>{t("servers.firewall.col_enabled")}</th>
                <th className={cn(VX_TBL_TH, "w-[80px]")} />
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={6}>
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
                      <span
                        className={cn(
                          "inline-block rounded-full px-[11px] py-[3px] font-sans text-[12px]",
                          rule.action === "allow"
                            ? "bg-[var(--srv-accent-soft)] text-[var(--srv-accent)]"
                            : "bg-[var(--vx-danger-tint)] text-[var(--vx-danger)]"
                        )}
                      >
                        {rule.action === "allow"
                          ? t("servers.firewall.badge_allow")
                          : t("servers.firewall.badge_deny")}
                      </span>
                    </td>
                    <td className={VX_TBL_TD}>{rule.protocol.toUpperCase()}</td>
                    <td className={VX_TBL_TD}>
                      {rule.port_to && rule.port_to > rule.port_from
                        ? `${rule.port_from}–${rule.port_to}`
                        : rule.port_from}
                    </td>
                    <td className={VX_TBL_TD}>
                      {rule.source || t("servers.firewall.source_any")}
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
                          if (!(await confirmAction(t("servers.firewall.delete_confirm")))) return;
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
    </div>
  );
}
