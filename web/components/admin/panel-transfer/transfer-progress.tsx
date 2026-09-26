"use client";

import { useEffect, useRef } from "react";

import { Bar, InfoRow, Panel, VX_CODE, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import type { PanelTransferItem } from "@/lib/api";
import { cn } from "@/lib/utils";

const STAGES = [
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

export function formatBytes(n: number) {
  if (!n) return "0";
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GB`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(n / 1024))} KB`;
}

export function TransferProgress({ item }: { item: PanelTransferItem }) {
  const t = useT();
  const logRef = useRef<HTMLPreElement>(null);
  const running = item.status === "running" || item.status === "pending";

  useEffect(() => {
    const node = logRef.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [item.log]);

  const index = STAGES.indexOf(item.stage);
  const pct = item.bytes_total > 0 ? (item.bytes_done / item.bytes_total) * 100 : 0;

  return (
    <Panel
      title={t("admin.panel_transfer.progress.title")}
      aside={
        <span className={cn("text-[11.5px]", VX_MUTED)}>
          {t(`admin.panel_transfer.status.${item.status}`)}
        </span>
      }
    >
      <div className="grid gap-3">
        <div className="flex flex-wrap gap-1.5">
          {STAGES.map((stage, i) => (
            <span
              key={stage}
              className={cn(
                "rounded-full px-2.5 py-1 text-[11px]",
                index >= 0 && i < index && "bg-[var(--vx-tint-hover)] text-[var(--vx-fg)]",
                index === i && "bg-[var(--vx-fg-strong)] text-[var(--vx-bg)]",
                (index < 0 || i > index) && cn("bg-[var(--vx-inset)]", VX_FAINT)
              )}
            >
              {t(`admin.panel_transfer.stage.${stage}`)}
            </span>
          ))}
        </div>

        {item.bytes_total > 0 && (
          <div className="grid gap-1.5">
            <Bar pct={pct} />
            <div className={cn("text-[11.5px]", VX_MUTED)}>
              {t("admin.panel_transfer.progress.bytes", {
                done: formatBytes(item.bytes_done),
                total: formatBytes(item.bytes_total),
              })}
            </div>
          </div>
        )}

        {item.agents_total > 0 && (
          <InfoRow
            k={t("admin.panel_transfer.progress.agents")}
            v={`${item.agents_done}/${item.agents_total}${
              item.agents_failed > 0 ? ` (${item.agents_failed})` : ""
            }`}
          />
        )}

        {item.error && (
          <div className="rounded-[10px] border border-[rgba(224,122,122,0.28)] bg-[rgba(224,122,122,0.07)] px-3 py-2 text-[12.5px] text-[var(--vx-danger)]">
            {item.error}
          </div>
        )}

        {item.log && (
          <pre
            ref={logRef}
            className={cn(
              "max-h-[320px] overflow-auto rounded-[10px] px-3 py-2.5 font-mono text-[11.5px] leading-[1.6] whitespace-pre-wrap",
              VX_CODE
            )}
          >
            {item.log}
          </pre>
        )}

        {running && (
          <div className={cn("text-[11.5px]", VX_FAINT)}>{t("admin.panel_transfer.progress.running_hint")}</div>
        )}
      </div>
    </Panel>
  );
}
