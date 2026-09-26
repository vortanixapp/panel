"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Archive,
  Database,
  Globe,
  HardDrive,
  KeyRound,
  Lock,
  MapPin,
  Server,
  ServerCog,
  TriangleAlert,
} from "lucide-react";
import { toast } from "sonner";

import { confirmAction } from "@/components/action-dialog";
import { PageShell } from "@/components/layout/page-shell";
import {
  Btn,
  EmptyState,
  Field,
  Notice,
  Panel,
  Toggle,
  VX_FAINT,
  VX_INPUT,
  VX_MUTED,
  VX_ROW_LINE,
} from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import { usePanelTransfer } from "@/hooks/use-panel-transfer";
import {
  cancelPanelTransfer,
  retryPanelTransferAgents,
  setPanelFreeze,
  startPanelTransferSSH,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { ArchiveExportCard, ArchiveImportCard } from "./transfer-archive";
import { TransferProgress } from "./transfer-progress";
import {
  ChoiceCard,
  formatBytes,
  MarkedItem,
  SectionLabel,
  StatTile,
  StatusChip,
} from "./transfer-ui";
import {
  emptyTarget,
  TransferTargetForm,
  targetPayload,
  targetReady,
  type TargetState,
} from "./transfer-target-form";

export function PanelTransferPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const state = usePanelTransfer();
  const [mode, setMode] = useState<"ssh" | "archive">("ssh");
  const [target, setTarget] = useState<TargetState>(emptyTarget);
  const [sameAddress, setSameAddress] = useState(false);
  const [newAddress, setNewAddress] = useState("");
  const [freeze, setFreeze] = useState(true);

  const data = state.data;
  const active = data?.active ?? null;
  const last = data?.last ?? null;
  const probe = data?.probe;
  const busy = !!active;
  const shown = active ?? last;

  const refresh = () => void qc.invalidateQueries({ queryKey: queryKeys.panelTransfer });

  const start = useMutation({
    mutationFn: () =>
      startPanelTransferSSH({
        ...targetPayload(target),
        new_address: newAddress.trim(),
        same_address: sameAddress,
        freeze_writes: freeze,
      }),
    onSuccess: () => {
      setTarget(emptyTarget);
      toast.success(t("admin.panel_transfer.started"));
      refresh();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const cancel = useMutation({
    mutationFn: cancelPanelTransfer,
    onSuccess: () => {
      toast.success(t("admin.panel_transfer.cancelled"));
      refresh();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const agentsRetry = useMutation({
    mutationFn: () =>
      retryPanelTransferAgents({
        ...targetPayload(target),
        new_address: newAddress.trim() || last?.new_address || "",
      }),
    onSuccess: () => {
      toast.success(t("admin.panel_transfer.agents.started"));
      refresh();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const freezeToggle = useMutation({
    mutationFn: (on: boolean) => setPanelFreeze(on),
    onSuccess: () => refresh(),
    onError: (e: Error) => toast.error(e.message),
  });

  const confirmStart = async () => {
    const ok = await confirmAction(t("admin.panel_transfer.confirm.start"), {
      confirmText: t("admin.panel_transfer.confirm.start_ok"),
      destructive: true,
    });
    if (ok) start.mutate();
  };

  const confirmUnfreeze = async () => {
    const ok = await confirmAction(t("admin.panel_transfer.confirm.unfreeze"), {
      confirmText: t("admin.panel_transfer.confirm.unfreeze_ok"),
      destructive: true,
    });
    if (ok) freezeToggle.mutate(false);
  };

  const total = (probe?.db_bytes ?? 0) + (probe?.uploads_bytes ?? 0);
  const canStart = targetReady(target) && (sameAddress || newAddress.trim().length > 0) && !busy;
  const nodes = data?.nodes ?? [];

  return (
    <PageShell variant="admin">
      <div className="flex w-full flex-col gap-[18px] pb-10">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.panel_transfer.title")}
            </h1>
            <p className="text-sm text-muted-foreground">{t("admin.panel_transfer.subtitle")}</p>
          </div>
          <div className="flex items-center gap-2.5">
            {data?.frozen && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-[rgba(232,160,60,0.12)] px-2.5 py-[3px] text-[11px] font-medium text-[var(--vx-warn)]">
                <Lock className="size-3" />
                {t("admin.panel_transfer.read_only")}
              </span>
            )}
            {shown && (
              <StatusChip
                status={shown.status}
                label={t(`admin.panel_transfer.status.${shown.status}`)}
              />
            )}
          </div>
        </div>

        {data?.frozen && (
          <Notice tone="warn" className="flex flex-wrap items-center justify-between gap-3">
            <span className="inline-flex items-start gap-2.5">
              <TriangleAlert className="mt-[2px] size-4 shrink-0" />
              {t("admin.panel_transfer.frozen_banner")}
            </span>
            <Btn size="sm" disabled={freezeToggle.isPending} onClick={() => void confirmUnfreeze()}>
              {t("admin.panel_transfer.unfreeze")}
            </Btn>
          </Notice>
        )}

        {data?.probe_error && (
          <Notice>
            <span className="inline-flex items-start gap-2.5">
              <TriangleAlert className="mt-[2px] size-4 shrink-0" />
              {data.probe_error}
            </span>
          </Notice>
        )}

        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatTile
            label={t("admin.panel_transfer.overview.db")}
            value={probe ? formatBytes(probe.db_bytes) : "—"}
            sub={t("admin.panel_transfer.tile.db_sub")}
            icon={<Database className="size-3.5" />}
          />
          <StatTile
            label={t("admin.panel_transfer.overview.uploads")}
            value={probe ? formatBytes(probe.uploads_bytes) : "—"}
            sub={t("admin.panel_transfer.tile.uploads_sub")}
            icon={<HardDrive className="size-3.5" />}
          />
          <StatTile
            label={t("admin.panel_transfer.overview.secrets")}
            value={probe ? probe.env_keys.length : "—"}
            sub={t("admin.panel_transfer.tile.secrets_sub")}
            icon={<KeyRound className="size-3.5" />}
          />
          <StatTile
            label={t("admin.panel_transfer.overview.nodes")}
            value={nodes.length}
            sub={t("admin.panel_transfer.tile.nodes_sub")}
            icon={<MapPin className="size-3.5" />}
          />
        </div>

        <div className="grid items-start gap-[18px] xl:grid-cols-[minmax(0,1fr)_340px]">
          <div className="flex flex-col gap-[18px]">
            <div className="grid gap-3 sm:grid-cols-2">
              <ChoiceCard
                active={mode === "ssh"}
                icon={<ServerCog className="size-4" />}
                title={t("admin.panel_transfer.tab.ssh")}
                desc={t("admin.panel_transfer.mode.ssh_desc")}
                onSelect={() => setMode("ssh")}
              />
              <ChoiceCard
                active={mode === "archive"}
                icon={<Archive className="size-4" />}
                title={t("admin.panel_transfer.tab.archive")}
                desc={t("admin.panel_transfer.mode.archive_desc")}
                onSelect={() => setMode("archive")}
              />
            </div>

            {mode === "ssh" ? (
              <Panel
                title={
                  <span className="inline-flex items-center gap-2">
                    <Server className="size-3.5" />
                    {t("admin.panel_transfer.ssh.title")}
                  </span>
                }
              >
                <div className="grid gap-5">
                  <p className={cn("text-[12.5px] leading-[1.55]", VX_FAINT)}>
                    {t("admin.panel_transfer.ssh.hint")}
                  </p>

                  <div className="grid gap-3">
                    <SectionLabel>{t("admin.panel_transfer.ssh.access")}</SectionLabel>
                    <TransferTargetForm target={target} onChange={setTarget} disabled={busy} />
                  </div>

                  <div className={cn("grid gap-3 border-t pt-4", VX_ROW_LINE)}>
                    <SectionLabel>{t("admin.panel_transfer.address.section")}</SectionLabel>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <ChoiceCard
                        active={sameAddress}
                        disabled={busy}
                        icon={<Globe className="size-4" />}
                        title={t("admin.panel_transfer.address.same")}
                        desc={t("admin.panel_transfer.address.same_desc")}
                        onSelect={() => setSameAddress(true)}
                      />
                      <ChoiceCard
                        active={!sameAddress}
                        disabled={busy}
                        icon={<MapPin className="size-4" />}
                        title={t("admin.panel_transfer.address.new")}
                        desc={t("admin.panel_transfer.address.new_desc")}
                        onSelect={() => setSameAddress(false)}
                      />
                    </div>
                    {sameAddress ? (
                      <p className={cn("text-[11.5px] leading-[1.5]", VX_FAINT)}>
                        {t("admin.panel_transfer.address.same_hint", {
                          address: probe?.source_address ?? "—",
                        })}
                      </p>
                    ) : (
                      <Field label={t("admin.panel_transfer.address.new_label")}>
                        <input
                          className={VX_INPUT}
                          value={newAddress}
                          disabled={busy}
                          placeholder="panel2.example.com"
                          onChange={(e) => setNewAddress(e.target.value)}
                        />
                      </Field>
                    )}
                  </div>

                  <div className={cn("flex items-start justify-between gap-4 border-t pt-4", VX_ROW_LINE)}>
                    <div className="grid gap-1">
                      <span className="text-[12.5px] text-[var(--vx-fg)]">
                        {t("admin.panel_transfer.freeze.label")}
                      </span>
                      <span className={cn("text-[11.5px] leading-[1.5]", VX_FAINT)}>
                        {t("admin.panel_transfer.freeze.hint")}
                      </span>
                    </div>
                    <Toggle checked={freeze} disabled={busy} onChange={setFreeze} />
                  </div>

                  <div className={cn("flex flex-wrap items-center gap-2 border-t pt-4", VX_ROW_LINE)}>
                    <Btn
                      tone="primary"
                      disabled={!canStart || start.isPending}
                      onClick={() => void confirmStart()}
                    >
                      {start.isPending
                        ? t("admin.panel_transfer.starting")
                        : t("admin.panel_transfer.start")}
                    </Btn>
                    {busy && (
                      <Btn tone="danger" disabled={cancel.isPending} onClick={() => cancel.mutate()}>
                        {t("admin.panel_transfer.cancel")}
                      </Btn>
                    )}
                    {busy && (
                      <span className={cn("text-[11.5px]", VX_FAINT)}>
                        {t("admin.panel_transfer.busy_hint")}
                      </span>
                    )}
                  </div>
                </div>
              </Panel>
            ) : (
              <div className="flex flex-col gap-[18px]">
                <ArchiveExportCard disabled={busy} />
                <ArchiveImportCard onDone={refresh} />
              </div>
            )}

            {shown && <TransferProgress item={shown} />}

            {last && last.agents_failed > 0 && !active && (
              <Panel title={t("admin.panel_transfer.agents.title")}>
                <div className="grid gap-3">
                  <p className={cn("text-[12.5px] leading-[1.55]", VX_FAINT)}>
                    {t("admin.panel_transfer.agents.hint", { count: last.agents_failed })}
                  </p>
                  <div>
                    <Btn
                      disabled={!targetReady(target) || agentsRetry.isPending}
                      onClick={() => agentsRetry.mutate()}
                    >
                      {t("admin.panel_transfer.agents.retry")}
                    </Btn>
                  </div>
                </div>
              </Panel>
            )}
          </div>

          <div className="flex flex-col gap-[18px]">
            <Panel title={t("admin.panel_transfer.overview.title")}>
              {state.isLoading ? (
                <EmptyState>{t("common.loading")}</EmptyState>
              ) : (
                <div className="grid gap-0">
                  <MarkedItem on>{t("admin.panel_transfer.moves.db")}</MarkedItem>
                  <MarkedItem on>{t("admin.panel_transfer.moves.uploads")}</MarkedItem>
                  <MarkedItem on>{t("admin.panel_transfer.moves.secrets")}</MarkedItem>
                  <MarkedItem on={false}>{t("admin.panel_transfer.skipped.redis")}</MarkedItem>
                  <MarkedItem on={false}>{t("admin.panel_transfer.skipped.certs")}</MarkedItem>
                  <MarkedItem on={false}>{t("admin.panel_transfer.skipped.servers")}</MarkedItem>
                  <div className={cn("mt-2 grid gap-1.5 border-t pt-3 text-[11.5px]", VX_ROW_LINE, VX_MUTED)}>
                    <div className="flex justify-between gap-3">
                      <span>{t("admin.panel_transfer.overview.total")}</span>
                      <span className="font-mono text-[var(--vx-fg)]">{formatBytes(total)}</span>
                    </div>
                    <div className="flex justify-between gap-3">
                      <span>{t("admin.panel_transfer.overview.version")}</span>
                      <span className="font-mono text-[var(--vx-fg)]">{data?.version ?? "—"}</span>
                    </div>
                    <div className="flex justify-between gap-3">
                      <span>{t("admin.panel_transfer.overview.address")}</span>
                      <span className="truncate font-mono text-[var(--vx-fg)]">
                        {probe?.source_address ?? "—"}
                      </span>
                    </div>
                    <div className="flex justify-between gap-3">
                      <span>{t("admin.panel_transfer.overview.mode")}</span>
                      <span className="font-mono text-[var(--vx-fg)]">{probe?.mode ?? "—"}</span>
                    </div>
                  </div>
                </div>
              )}
            </Panel>

            <Panel title={t("admin.panel_transfer.requirements.title")}>
              <div className="grid gap-0">
                <MarkedItem on>{t("admin.panel_transfer.requirements.os")}</MarkedItem>
                <MarkedItem on>{t("admin.panel_transfer.requirements.root")}</MarkedItem>
                <MarkedItem on>{t("admin.panel_transfer.requirements.ports")}</MarkedItem>
                <MarkedItem on>
                  {t("admin.panel_transfer.requirements.space", { size: formatBytes(total * 3) })}
                </MarkedItem>
              </div>
            </Panel>

            <Panel title={t("admin.panel_transfer.nodes.title")} flush>
              {nodes.length === 0 ? (
                <EmptyState>{t("admin.panel_transfer.nodes.empty")}</EmptyState>
              ) : (
                <div className="px-[18px] py-1.5">
                  {nodes.map((node, i) => (
                    <div
                      key={node.id}
                      className={cn(
                        "flex items-center justify-between gap-3 py-[9px] text-[12.5px]",
                        i < nodes.length - 1 && VX_ROW_LINE
                      )}
                    >
                      <span className="inline-flex min-w-0 items-center gap-2">
                        <span
                          className={cn(
                            "size-[6px] shrink-0 rounded-full",
                            node.status === "online"
                              ? "bg-[var(--vx-ok)]"
                              : "bg-[var(--vx-ghost)]"
                          )}
                        />
                        <span className="truncate">{node.name || node.host}</span>
                      </span>
                      <span className={cn("shrink-0 font-mono text-[11.5px]", VX_MUTED)}>{node.host}</span>
                    </div>
                  ))}
                </div>
              )}
            </Panel>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
