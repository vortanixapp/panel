"use client";

import { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, Loader2, X } from "lucide-react";

import {
  Bar,
  Btn,
  Panel,
  VX_CODE,
  VX_FAINT,
  VX_MUTED,
  VX_ROW_LINE,
} from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import type { PanelTransferItem } from "@/lib/api";
import { cn } from "@/lib/utils";
import { formatBytes, SectionLabel, StatusChip } from "./transfer-ui";

const ALL_STAGES = [
  "prepare",
  "freeze",
  "probe_target",
  "install_docker",
  "clone",
  "init_env",
  "compose_up",
  "transfer",
  "health",
  "relay_cert",
  "agents",
  "done",
];

function stagesOf(item: PanelTransferItem) {
  return ALL_STAGES.filter((stage) => {
    if (stage === "freeze" && !item.freeze_writes) return false;
    if ((stage === "relay_cert" || stage === "agents") && item.same_address) return false;
    if (stage === "relay_cert" && item.stage !== "relay_cert" && item.agents_total === 0) return false;
    return true;
  });
}

function StageMark({ state }: { state: "done" | "current" | "failed" | "pending" }) {
  if (state === "done") {
    return (
      <span className="inline-flex size-[18px] items-center justify-center rounded-full bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]">
        <Check className="size-[11px]" />
      </span>
    );
  }
  if (state === "failed") {
    return (
      <span className="inline-flex size-[18px] items-center justify-center rounded-full bg-[rgba(224,122,122,0.14)] text-[var(--vx-danger)]">
        <X className="size-[11px]" />
      </span>
    );
  }
  if (state === "current") {
    return (
      <span className="inline-flex size-[18px] items-center justify-center rounded-full bg-[rgba(232,160,60,0.14)] text-[var(--vx-warn)]">
        <Loader2 className="size-[11px] animate-spin" />
      </span>
    );
  }
  return (
    <span className="inline-flex size-[18px] items-center justify-center rounded-full bg-[var(--vx-inset)]">
      <span className="size-[5px] rounded-full bg-[var(--vx-ghost)]" />
    </span>
  );
}

export function TransferProgress({ item }: { item: PanelTransferItem }) {
  const t = useT();
  const logRef = useRef<HTMLPreElement>(null);
  const running = item.status === "running" || item.status === "pending";
  const [logOpen, setLogOpen] = useState(running);

  useEffect(() => {
    if (!logOpen) return;
    const node = logRef.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [item.log, logOpen]);

  const stages = stagesOf(item);
  const index = stages.indexOf(item.stage);
  const pct = item.bytes_total > 0 ? (item.bytes_done / item.bytes_total) * 100 : 0;

  const stateOf = (i: number): "done" | "current" | "failed" | "pending" => {
    if (item.status === "completed") return "done";
    if (i < index) return "done";
    if (i > index) return "pending";
    if (item.status === "failed" || item.status === "cancelled") return "failed";
    return "current";
  };

  return (
    <Panel
      title={t("admin.panel_transfer.progress.title")}
      aside={
        <div className="flex items-center gap-2.5">
          {item.target_host && (
            <span className={cn("font-mono text-[11.5px]", VX_FAINT)}>{item.target_host}</span>
          )}
          <StatusChip status={item.status} label={t(`admin.panel_transfer.status.${item.status}`)} />
        </div>
      }
    >
      <div className="grid gap-4">
        <div className="grid gap-0">
          {stages.map((stage, i) => {
            const state = stateOf(i);
            return (
              <div
                key={stage}
                className={cn(
                  "flex items-center gap-2.5 py-[7px] text-[12.5px] last:border-b-0",
                  i < stages.length - 1 && VX_ROW_LINE
                )}
              >
                <StageMark state={state} />
                <span
                  className={cn(
                    "flex-1",
                    state === "pending" && VX_FAINT,
                    state === "current" && "font-medium text-[var(--vx-fg)]",
                    state === "failed" && "text-[var(--vx-danger)]"
                  )}
                >
                  {t(`admin.panel_transfer.stage.${stage}`)}
                </span>
                {stage === "transfer" && item.bytes_total > 0 && (
                  <span className={cn("shrink-0 font-mono text-[11.5px]", VX_MUTED)}>
                    {formatBytes(item.bytes_done)} / {formatBytes(item.bytes_total)}
                  </span>
                )}
                {stage === "agents" && item.agents_total > 0 && (
                  <span className={cn("shrink-0 font-mono text-[11.5px]", VX_MUTED)}>
                    {item.agents_done}/{item.agents_total}
                    {item.agents_failed > 0 && (
                      <span className="text-[var(--vx-danger)]"> · {item.agents_failed}</span>
                    )}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {item.bytes_total > 0 && item.stage === "transfer" && <Bar pct={pct} />}

        {item.error && (
          <div className="rounded-[10px] border border-[rgba(224,122,122,0.28)] bg-[rgba(224,122,122,0.07)] px-3 py-2.5 text-[12.5px] leading-[1.5] text-[var(--vx-danger)]">
            {item.error}
          </div>
        )}

        {item.log && (
          <div className="grid gap-2">
            <div className="flex items-center justify-between gap-3">
              <SectionLabel>{t("admin.panel_transfer.progress.log")}</SectionLabel>
              <Btn size="sm" tone="ghost" onClick={() => setLogOpen((prev) => !prev)}>
                {logOpen ? t("admin.panel_transfer.progress.log_hide") : t("admin.panel_transfer.progress.log_show")}
                <ChevronDown className={cn("size-3.5 transition-transform", logOpen && "rotate-180")} />
              </Btn>
            </div>
            {logOpen && (
              <pre
                ref={logRef}
                className={cn(
                  "max-h-[300px] overflow-auto rounded-[10px] px-3 py-2.5 font-mono text-[11.5px] leading-[1.6] whitespace-pre-wrap",
                  VX_CODE,
                  VX_MUTED
                )}
              >
                {item.log}
              </pre>
            )}
          </div>
        )}

        {running && (
          <div className={cn("text-[11.5px]", VX_FAINT)}>{t("admin.panel_transfer.progress.running_hint")}</div>
        )}
      </div>
    </Panel>
  );
}
