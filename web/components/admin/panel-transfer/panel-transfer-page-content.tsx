"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { confirmAction } from "@/components/action-dialog";
import { PageShell } from "@/components/layout/page-shell";
import {
  Btn,
  EmptyState,
  Field,
  InfoRow,
  Notice,
  Panel,
  SubTabs,
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
import { formatBytes, TransferProgress } from "./transfer-progress";
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
  const [tab, setTab] = useState("ssh");
  const [target, setTarget] = useState<TargetState>(emptyTarget);
  const [sameAddress, setSameAddress] = useState(false);
  const [newAddress, setNewAddress] = useState("");
  const [freeze, setFreeze] = useState(true);

  const data = state.data;
  const active = data?.active ?? null;
  const last = data?.last ?? null;
  const probe = data?.probe;
  const busy = !!active;

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

  const canStart = targetReady(target) && (sameAddress || newAddress.trim().length > 0) && !busy;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6 pb-10">
        <div className="space-y-1.5">
          <h1 className="text-[26px] leading-none font-bold tracking-tight">
            {t("admin.panel_transfer.title")}
          </h1>
          <p className="text-sm text-muted-foreground">{t("admin.panel_transfer.subtitle")}</p>
        </div>

        {data?.frozen && (
          <Notice tone="warn" className="flex flex-wrap items-center justify-between gap-3">
            <span>{t("admin.panel_transfer.frozen_banner")}</span>
            <Btn size="sm" disabled={freezeToggle.isPending} onClick={() => void confirmUnfreeze()}>
              {t("admin.panel_transfer.unfreeze")}
            </Btn>
          </Notice>
        )}

        {data?.probe_error && <Notice>{data.probe_error}</Notice>}

        <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
          <div className="space-y-6">
            <SubTabs
              items={[
                { id: "ssh", title: t("admin.panel_transfer.tab.ssh") },
                { id: "archive", title: t("admin.panel_transfer.tab.archive") },
              ]}
              active={tab}
              onSelect={setTab}
            />

            {tab === "ssh" ? (
              <Panel title={t("admin.panel_transfer.ssh.title")}>
                <div className="grid gap-4">
                  <p className={cn("text-[12.5px]", VX_FAINT)}>{t("admin.panel_transfer.ssh.hint")}</p>
                  <TransferTargetForm target={target} onChange={setTarget} disabled={busy} />

                  <div className={cn("grid gap-3 border-t pt-4", VX_ROW_LINE)}>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-[12.5px]">{t("admin.panel_transfer.address.same")}</span>
                      <Toggle checked={sameAddress} disabled={busy} onChange={setSameAddress} />
                    </div>
                    {sameAddress ? (
                      <p className={cn("text-[11.5px]", VX_FAINT)}>
                        {t("admin.panel_transfer.address.same_hint", {
                          address: probe?.source_address ?? "",
                        })}
                      </p>
                    ) : (
                      <Field label={t("admin.panel_transfer.address.new")}>
                        <input
                          className={VX_INPUT}
                          value={newAddress}
                          disabled={busy}
                          placeholder="panel2.example.com"
                          onChange={(e) => setNewAddress(e.target.value)}
                        />
                      </Field>
                    )}
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-[12.5px]">{t("admin.panel_transfer.freeze.label")}</span>
                      <Toggle checked={freeze} disabled={busy} onChange={setFreeze} />
                    </div>
                    <p className={cn("text-[11.5px]", VX_FAINT)}>{t("admin.panel_transfer.freeze.hint")}</p>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <Btn tone="primary" disabled={!canStart || start.isPending} onClick={() => void confirmStart()}>
                      {start.isPending ? t("admin.panel_transfer.starting") : t("admin.panel_transfer.start")}
                    </Btn>
                    {busy && (
                      <Btn tone="danger" disabled={cancel.isPending} onClick={() => cancel.mutate()}>
                        {t("admin.panel_transfer.cancel")}
                      </Btn>
                    )}
                  </div>
                </div>
              </Panel>
            ) : (
              <div className="space-y-6">
                <ArchiveExportCard disabled={busy} />
                <ArchiveImportCard onDone={refresh} />
              </div>
            )}

            {active && <TransferProgress item={active} />}
            {!active && last && <TransferProgress item={last} />}

            {last && last.agents_failed > 0 && (
              <Panel title={t("admin.panel_transfer.agents.title")}>
                <div className="grid gap-3">
                  <p className={cn("text-[12.5px]", VX_FAINT)}>
                    {t("admin.panel_transfer.agents.hint", { count: last.agents_failed })}
                  </p>
                  <Btn disabled={!targetReady(target) || agentsRetry.isPending} onClick={() => agentsRetry.mutate()}>
                    {t("admin.panel_transfer.agents.retry")}
                  </Btn>
                </div>
              </Panel>
            )}
          </div>

          <div className="space-y-6">
            <Panel title={t("admin.panel_transfer.overview.title")}>
              {state.isLoading ? (
                <EmptyState>{t("common.loading")}</EmptyState>
              ) : (
                <div className="grid gap-1">
                  <InfoRow k={t("admin.panel_transfer.overview.version")} v={data?.version ?? ""} />
                  <InfoRow
                    k={t("admin.panel_transfer.overview.db")}
                    v={probe ? formatBytes(probe.db_bytes) : "—"}
                  />
                  <InfoRow
                    k={t("admin.panel_transfer.overview.uploads")}
                    v={probe ? formatBytes(probe.uploads_bytes) : "—"}
                  />
                  <InfoRow
                    k={t("admin.panel_transfer.overview.secrets")}
                    v={probe ? String(probe.env_keys.length) : "—"}
                  />
                  <InfoRow
                    k={t("admin.panel_transfer.overview.address")}
                    v={probe?.source_address ?? "—"}
                  />
                  <InfoRow k={t("admin.panel_transfer.overview.mode")} v={probe?.mode ?? "—"} />
                  <InfoRow k={t("admin.panel_transfer.overview.nodes")} v={String(data?.nodes.length ?? 0)} />
                </div>
              )}
            </Panel>

            <Panel title={t("admin.panel_transfer.skipped.title")}>
              <ul className={cn("grid gap-1.5 text-[12px]", VX_MUTED)}>
                <li>{t("admin.panel_transfer.skipped.redis")}</li>
                <li>{t("admin.panel_transfer.skipped.certs")}</li>
                <li>{t("admin.panel_transfer.skipped.servers")}</li>
              </ul>
            </Panel>

            <Panel title={t("admin.panel_transfer.nodes.title")} flush>
              {(data?.nodes.length ?? 0) === 0 ? (
                <EmptyState>{t("admin.panel_transfer.nodes.empty")}</EmptyState>
              ) : (
                <div className="px-[18px] py-2">
                  {data?.nodes.map((node) => (
                    <InfoRow key={node.id} k={node.name || node.host} v={node.host} />
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
