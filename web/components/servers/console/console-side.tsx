"use client";

import { Circle, Users } from "lucide-react";

import { VX_FAINT, VX_MONO_LABEL, VX_MUTED } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export type ConsolePlayer = { name: string; live: boolean };

export function PlayersCard({
  players,
  max,
  onPick,
}: {
  players: ConsolePlayer[];
  max: number;
  onPick?: (name: string) => void;
}) {
  const t = useT();

  return (
    <section className="rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-card)] p-3.5">
      <div className="mb-2.5 flex items-center justify-between gap-2">
        <span className={VX_MONO_LABEL}>{t("servers.console.players")}</span>
        <span className="font-mono text-[11.5px] text-[var(--vx-muted)]">
          {players.length}
          {max > 0 ? ` / ${max}` : ""}
        </span>
      </div>
      {players.length === 0 ? (
        <p className={cn("text-[12.5px]", VX_MUTED)}>{t("servers.console.players_empty")}</p>
      ) : (
        <ul className="space-y-0.5">
          {players.map((player) => (
            <li key={player.name}>
              <button
                type="button"
                onClick={() => onPick?.(player.name)}
                disabled={!onPick}
                className={cn(
                  "flex w-full items-center gap-2 rounded-[7px] px-1.5 py-1 text-start text-[12.5px] transition-colors",
                  onPick ? "hover:bg-[var(--vx-inset)]" : "cursor-default"
                )}
              >
                <Circle
                  className={cn(
                    "size-2 shrink-0",
                    player.live ? "fill-[var(--vx-info)] text-[var(--vx-info)]" : "fill-[var(--vx-ghost)] text-[var(--vx-ghost)]"
                  )}
                />
                <span className="min-w-0 flex-1 truncate">{player.name}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {players.some((player) => !player.live) && (
        <p className={cn("mt-2 text-[11px] leading-snug", VX_FAINT)}>
          {t("servers.console.players_from_log")}
        </p>
      )}
    </section>
  );
}

export type StateRow = { label: string; value: string };

export function StateCard({ rows, note }: { rows: StateRow[]; note?: string }) {
  const t = useT();
  const filled = rows.filter((row) => row.value);

  return (
    <section className="rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-card)] p-3.5">
      <div className="mb-2.5 flex items-center gap-2">
        <Users className="size-3.5 text-[var(--vx-faint)]" />
        <span className={VX_MONO_LABEL}>{t("servers.console.state")}</span>
      </div>
      {filled.length === 0 ? (
        <p className={cn("text-[12.5px]", VX_MUTED)}>{t("servers.console.state_empty")}</p>
      ) : (
        <dl className="space-y-1.5">
          {filled.map((row) => (
            <div key={row.label} className="flex items-baseline justify-between gap-3">
              <dt className={cn("shrink-0 text-[12px]", VX_MUTED)}>{row.label}</dt>
              <dd className="min-w-0 truncate text-end font-mono text-[12px] text-[var(--vx-fg)]">
                {row.value}
              </dd>
            </div>
          ))}
        </dl>
      )}
      {note && <p className={cn("mt-2.5 text-[11px] leading-snug", VX_FAINT)}>{note}</p>}
    </section>
  );
}
