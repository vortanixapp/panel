"use client";

import Link from "next/link";
import { MoreHorizontal } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Bar, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import type { AgentRow } from "@/lib/api";
import {
  TONE_CLASS,
  diskUsedPct,
  formatAgo,
  formatDuration,
  labelTone,
  pct,
  stateTone,
} from "@/lib/agents";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function StatePill({ state }: { state: AgentRow["state"] }) {
  const t = useT();
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-[2px] text-[11px] font-medium whitespace-nowrap",
        TONE_CLASS[stateTone(state)]
      )}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {t(`admin.agents.state.${state}`)}
    </span>
  );
}

export function LabelChips({ labels }: { labels: AgentRow["labels"] }) {
  const t = useT();
  if (!labels.length) return null;
  return (
    <span className="inline-flex flex-wrap gap-1">
      {labels.map((l) => (
        <span
          key={l}
          className={cn("rounded-full border px-1.5 py-px text-[10.5px] whitespace-nowrap", TONE_CLASS[labelTone(l)])}
        >
          {t(`admin.agents.label.${l}`)}
        </span>
      ))}
    </span>
  );
}

function Meter({ label, value }: { label: string; value: number | null }) {
  const warn = value != null && value > 85;
  return (
    <div className="grid min-w-[64px] gap-1">
      <div className="flex justify-between gap-2 font-mono text-[10.5px]">
        <span className={VX_FAINT}>{label}</span>
        <span className={cn(warn && "text-[var(--vx-warn)]")}>{value == null ? "—" : `${Math.round(value)}%`}</span>
      </div>
      <Bar pct={value ?? 0} barClassName={warn ? "bg-[var(--vx-warn)]" : undefined} />
    </div>
  );
}

export function UpdateStage({ row }: { row: AgentRow }) {
  const t = useT();
  const status = row.update?.status;
  if (!status || status === "done") return null;
  const tone = status === "failed" ? "text-[var(--vx-danger)]" : "text-[var(--vx-info)]";
  return (
    <span className={cn("text-[11px]", tone)} title={row.update?.error}>
      {t(`admin.agents.update_stage.${status}`)}
      {row.update?.target ? ` → ${row.update.target}` : ""}
    </span>
  );
}

export function AgentsTable({
  rows,
  selected,
  onToggle,
  onToggleAll,
  onUpdate,
  onAuto,
  pending,
}: {
  rows: AgentRow[];
  selected: Set<string>;
  onToggle: (id: string) => void;
  onToggleAll: () => void;
  onUpdate: (row: AgentRow) => void;
  onAuto: (row: AgentRow, value: boolean) => void;
  pending: boolean;
}) {
  const t = useT();
  const allOn = rows.length > 0 && rows.every((r) => selected.has(r.id));
  const th = cn("px-3 py-2.5 text-left text-[10.5px] font-normal tracking-[0.08em] uppercase", VX_FAINT);

  return (
    <div className="vx-tbl-wrap overflow-x-auto rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]">
      <table className="vx-tbl w-full min-w-[1080px] text-[12.5px]">
        <thead className="border-b border-[var(--vx-border)]">
          <tr>
            <th className="w-9 px-3">
              <input
                type="checkbox"
                aria-label={t("admin.agents.col.select_all")}
                checked={allOn}
                onChange={onToggleAll}
              />
            </th>
            <th className={th}>{t("admin.agents.col.node")}</th>
            <th className={th}>{t("admin.agents.col.status")}</th>
            <th className={th}>{t("admin.agents.col.version")}</th>
            <th className={th}>{t("admin.agents.col.uptime")}</th>
            <th className={th}>{t("admin.agents.col.load")}</th>
            <th className={th}>{t("admin.agents.col.servers")}</th>
            <th className={th}>{t("admin.agents.col.contact")}</th>
            <th className="w-10" />
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const href = `/admin/daemons/${row.id}`;
            return (
              <tr
                key={row.id}
                className={cn(
                  "border-b border-[var(--vx-elevated)] transition-colors last:border-b-0 hover:bg-[var(--vx-elevated)]",
                  selected.has(row.id) && "bg-[var(--vx-elevated)]"
                )}
              >
                <td className="px-3 py-3 align-middle" data-cell="lead">
                  <input
                    type="checkbox"
                    aria-label={row.name}
                    checked={selected.has(row.id)}
                    onChange={() => onToggle(row.id)}
                  />
                </td>
                <td className="px-3 py-3" data-cell="full">
                  <Link href={href} className="block min-w-0">
                    <span className="block truncate font-medium hover:underline">{row.name}</span>
                    <span className={cn("block truncate font-mono text-[11px]", VX_FAINT)}>
                      {[row.code, row.host].filter(Boolean).join(" · ") || "—"}
                    </span>
                  </Link>
                </td>
                <td className="px-3 py-3" data-label={t("admin.agents.col.status")}>
                  <span className="inline-flex flex-wrap items-center justify-end gap-1 md:justify-start">
                    <StatePill state={row.state} />
                    {row.maintenance && (
                      <span className={cn("rounded-full border px-1.5 py-px text-[10.5px]", TONE_CLASS.warn)}>
                        {t("admin.agents.label.maintenance")}
                      </span>
                    )}
                    <LabelChips labels={row.labels} />
                  </span>
                </td>
                <td className="px-3 py-3" data-label={t("admin.agents.col.version")}>
                  <span className="inline-grid justify-items-end gap-0.5 md:justify-items-start">
                    <span className="font-mono">{row.version || "—"}</span>
                    <UpdateStage row={row} />
                    {row.auto_update && !row.update?.status && (
                      <span className={cn("text-[10.5px]", VX_FAINT)}>{t("admin.agents.auto.short")}</span>
                    )}
                  </span>
                </td>
                <td className="px-3 py-3 font-mono" data-label={t("admin.agents.col.uptime")}>
                  <span className="inline-grid justify-items-end gap-0.5 md:justify-items-start">
                    <span>{formatDuration(row.uptime_sec)}</span>
                    {row.host_uptime_sec ? (
                      <span className={cn("text-[10.5px]", VX_FAINT)}>
                        {t("admin.agents.host_uptime", { value: formatDuration(row.host_uptime_sec) })}
                      </span>
                    ) : null}
                  </span>
                </td>
                <td className="px-3 py-3" data-label={t("admin.agents.col.load")}>
                  <div className="grid w-full max-w-[260px] grid-cols-3 gap-3">
                    <Meter label="CPU" value={pct(row.resources.cpu_percent)} />
                    <Meter label="RAM" value={pct(row.resources.ram_percent)} />
                    <Meter label={t("admin.agents.disk_short")} value={diskUsedPct(row)} />
                  </div>
                </td>
                <td className="px-3 py-3 font-mono" data-label={t("admin.agents.col.servers")}>
                  {row.servers.running}
                  <span className={VX_FAINT}> / {row.servers.total}</span>
                </td>
                <td className="px-3 py-3" data-label={t("admin.agents.col.contact")}>
                  <span className={cn("whitespace-nowrap", row.state === "offline" ? "text-[var(--vx-danger)]" : VX_MUTED)}>
                    {formatAgo(row.last_seen)}
                  </span>
                </td>
                <td className="px-2 py-3 text-right" data-cell="actions">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button
                        type="button"
                        aria-label={t("common.actions")}
                        className="inline-flex h-7 w-7 items-center justify-center rounded-[7px] text-[var(--vx-muted)] hover:bg-[var(--vx-tint)] hover:text-[var(--vx-fg)]"
                      >
                        <MoreHorizontal className="h-4 w-4" />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-56">
                      <DropdownMenuItem asChild>
                        <Link href={href}>{t("admin.agents.menu.open")}</Link>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={`/admin/locations/${row.id}`}>{t("admin.agents.menu.location")}</Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem disabled={pending || !row.outdated} onClick={() => onUpdate(row)}>
                        {t("admin.agents.menu.update")}
                      </DropdownMenuItem>
                      <DropdownMenuItem disabled={pending} onClick={() => onAuto(row, !row.auto_update)}>
                        {row.auto_update ? t("admin.agents.menu.auto_off") : t("admin.agents.menu.auto_on")}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
