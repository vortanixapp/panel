"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import {
  Btn,
  InfoRow,
  Notice,
  Panel,
  Toggle,
  VX_FAINT,
  VX_MUTED,
  VX_TEXTAREA,
} from "@/components/vx/panel-ui";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import {
  agentInstallScript,
  agentSSHCheck,
  agentSSHExec,
  agentSSHInstall,
  apiErrorCode,
  setAgentAutoUpdate,
  updateAgent,
} from "@/lib/api";
import { agentErrorText, formatDateTime } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

function errText(e: unknown) {
  return agentErrorText(apiErrorCode(e), e instanceof Error ? e.message : "");
}

export function AgentMaintenanceTab({ id, agent, viewer }: AgentTabProps) {
  const t = useT();
  const qc = useQueryClient();
  const [reinstallOpen, setReinstallOpen] = useState(false);
  const [install, setInstall] = useState<{ script?: string; env_file?: string } | null>(null);
  const [cmd, setCmd] = useState("");
  const [execOut, setExecOut] = useState<{ output: string; exit_code: number; duration_ms: number; truncated: boolean; error?: string } | null>(null);
  const refresh = () => void qc.invalidateQueries({ queryKey: queryKeys.agent(id) });

  const autoMutation = useMutation({
    mutationFn: (v: boolean) => setAgentAutoUpdate(id, v),
    onSuccess: (_r, v) => {
      toast.success(v ? t("admin.agents.auto.enabled_some") : t("admin.agents.auto.disabled_some"));
      refresh();
    },
    onError: (e) => toast.error(errText(e)),
  });
  const updateMutation = useMutation({
    mutationFn: () => updateAgent(id),
    onSuccess: () => {
      toast.success(t("admin.agents.update.sent"));
      refresh();
    },
    onError: (e) => toast.error(errText(e)),
  });
  const checkTask = useNodeTask(id, {
    invalidate: [queryKeys.agent(id)],
    successText: t("admin.agents.ssh.check_done"),
    failureText: t("admin.agents.ssh.check_failed"),
  });
  const reinstallTask = useNodeTask(id, {
    invalidate: [queryKeys.agent(id)],
    successText: t("admin.agents.reinstall.done"),
    failureText: t("admin.agents.reinstall.failed"),
  });
  const scriptMutation = useMutation({
    mutationFn: () => agentInstallScript(id),
    onSuccess: (res) => setInstall(res),
    onError: (e) => toast.error(errText(e)),
  });
  const execMutation = useMutation({
    mutationFn: () => agentSSHExec(id, cmd),
    onSuccess: (res) => setExecOut(res),
    onError: (e) => toast.error(errText(e)),
  });

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(t("admin.agents.copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  const u = agent.update;
  const updating = agent.labels.includes("updating");

  return (
    <div className="grid gap-[18px] lg:grid-cols-2">
      <Panel title={t("admin.agents.maint.update_title")}>
        <InfoRow k={t("admin.agents.maint.current")} v={agent.version || "—"} />
        <InfoRow k={t("admin.agents.maint.target")} v={agent.target_version || "—"} />
        <InfoRow
          k={t("admin.agents.maint.last_update")}
          v={
            u?.status
              ? `${t(`admin.agents.update_stage.${u.status}`)} · ${formatDateTime(u.finished_at ?? u.updated_at ?? u.started_at)}`
              : "—"
          }
        />
        {u?.status === "failed" && u.error && <div className="mt-2 text-[12px] text-[var(--vx-danger)]">{u.error}</div>}
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="inline-flex items-center gap-2.5 text-[12.5px]">
            <Toggle
              checked={agent.auto_update}
              disabled={!viewer.can_write || autoMutation.isPending}
              onChange={(v) => autoMutation.mutate(v)}
              label={t("admin.agents.maint.auto")}
            />
            <span className={VX_MUTED}>{t("admin.agents.maint.auto")}</span>
          </label>
          {viewer.can_write && (
            <Btn tone="primary" disabled={!agent.outdated || updating || updateMutation.isPending} onClick={() => updateMutation.mutate()}>
              {updating ? t("admin.agents.update.running") : t("admin.agents.update.to", { version: agent.target_version })}
            </Btn>
          )}
        </div>
      </Panel>

      <Panel title={t("admin.agents.maint.ssh_title")}>
        <div className={cn("text-[12.5px] leading-[1.6]", VX_MUTED)}>
          {agent.ssh ? t("admin.agents.maint.ssh_on") : t("admin.agents.maint.ssh_off")}
        </div>
        <div className={cn("mt-2 text-[12px] leading-[1.6]", VX_FAINT)}>{t("admin.agents.maint.restart_explain")}</div>
        {viewer.can_write && (
          <div className="mt-4 flex flex-wrap gap-2">
            <Btn disabled={!agent.ssh || checkTask.busy} onClick={() => void checkTask.run(() => agentSSHCheck(id))}>
              {checkTask.busy ? t("admin.agents.ssh.checking") : t("admin.agents.ssh.check")}
            </Btn>
            <Btn tone="danger" disabled={!agent.ssh || reinstallTask.busy} onClick={() => setReinstallOpen(true)}>
              {reinstallTask.busy ? t("admin.agents.reinstall.running") : t("admin.agents.reinstall.menu")}
            </Btn>
            {!agent.ssh && (
              <Link href={`/admin/locations/${id}/edit`} className="self-center text-[12.5px] text-primary hover:underline">
                {t("admin.agents.maint.ssh_setup")}
              </Link>
            )}
          </div>
        )}
      </Panel>

      {viewer.can_write && (
        <Panel title={t("admin.agents.maint.install_title")}>
          <div className={cn("text-[12.5px] leading-[1.6]", VX_MUTED)}>{t("admin.agents.maint.install_body")}</div>
          {install ? (
            <div className="mt-3 grid gap-3">
              {install.script && (
                <div className="grid gap-1.5">
                  <textarea readOnly className={cn(VX_TEXTAREA, "h-[120px]")} value={install.script} />
                  <Btn size="sm" className="justify-self-end" onClick={() => void copy(install.script!)}>
                    {t("admin.agents.copy")}
                  </Btn>
                </div>
              )}
              {install.env_file && (
                <div className="grid gap-1.5">
                  <textarea readOnly className={cn(VX_TEXTAREA, "h-[80px]")} value={install.env_file} />
                  <Btn size="sm" className="justify-self-end" onClick={() => void copy(install.env_file!)}>
                    {t("admin.agents.copy")}
                  </Btn>
                </div>
              )}
            </div>
          ) : (
            <Btn className="mt-3" disabled={scriptMutation.isPending} onClick={() => scriptMutation.mutate()}>
              {t("admin.agents.maint.install_show")}
            </Btn>
          )}
        </Panel>
      )}

      {viewer.can_ssh_exec && (
        <Panel title={t("admin.agents.maint.exec_title")}>
          <Notice tone="warn" className="mb-3 px-3 py-2 text-[12px]">
            {t("admin.agents.maint.exec_warning")}
          </Notice>
          <textarea
            className={cn(VX_TEXTAREA, "h-[70px]")}
            placeholder="docker ps"
            value={cmd}
            onChange={(e) => setCmd(e.target.value)}
            spellCheck={false}
          />
          <div className="mt-2 flex justify-end">
            <Btn tone="danger" disabled={!agent.ssh || !cmd.trim() || execMutation.isPending} onClick={() => execMutation.mutate()}>
              {execMutation.isPending ? t("admin.agents.maint.exec_running") : t("admin.agents.maint.exec_run")}
            </Btn>
          </div>
          {execOut && (
            <div className="mt-3 grid gap-1.5">
              <div className={cn("flex flex-wrap gap-3 font-mono text-[11px]", VX_FAINT)}>
                <span className={execOut.exit_code === 0 ? "text-[var(--vx-ok)]" : "text-[var(--vx-danger)]"}>
                  exit {execOut.exit_code}
                </span>
                <span>{execOut.duration_ms} ms</span>
                {execOut.truncated && <span>{t("admin.agents.maint.exec_truncated")}</span>}
              </div>
              <pre className="max-h-[320px] overflow-auto rounded-[10px] border border-[var(--vx-inset)] bg-[var(--vx-code)] p-3 font-mono text-[11.5px] whitespace-pre-wrap">
                {execOut.output || execOut.error || "—"}
              </pre>
            </div>
          )}
        </Panel>
      )}

      <ConfirmDialog
        open={reinstallOpen}
        onOpenChange={setReinstallOpen}
        title={t("admin.agents.reinstall.title")}
        description={t("admin.agents.reinstall.body")}
        confirmLabel={t("admin.agents.reinstall.confirm")}
        requirePhrase={agent.code || agent.name}
        pending={reinstallTask.busy}
        onConfirm={() => {
          setReinstallOpen(false);
          void reinstallTask.run(() => agentSSHInstall(id));
        }}
      />
    </div>
  );
}
