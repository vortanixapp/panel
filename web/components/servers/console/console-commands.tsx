"use client";

import { useState } from "react";
import { CornerDownLeft, Zap } from "lucide-react";

import { Btn, VX_FAINT, VX_INPUT, VX_MONO_LABEL } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import type { ConsoleCommand } from "@/lib/api";
import { fillCommand } from "@/lib/console/parse";
import { cn } from "@/lib/utils";

export function QuickCommands({
  commands,
  players,
  disabled,
  onRun,
}: {
  commands: ConsoleCommand[];
  players: string[];
  disabled: boolean;
  onRun: (command: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState<ConsoleCommand | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});

  if (commands.length === 0) return null;

  const start = (command: ConsoleCommand) => {
    if (!command.args || command.args.length === 0) {
      onRun(command.template);
      return;
    }
    setValues({});
    setOpen(open?.id === command.id ? null : command);
  };

  const ready =
    !open ||
    (open.args ?? []).every((arg) => arg.optional || (values[arg.name] ?? "").trim() !== "");

  return (
    <section className="rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-card)] p-3.5">
      <div className="mb-2.5 flex items-center gap-2">
        <Zap className="size-3.5 text-[var(--vx-faint)]" />
        <span className={VX_MONO_LABEL}>{t("servers.console.commands")}</span>
      </div>

      <div className="flex flex-wrap gap-1.5">
        {commands.map((command) => (
          <button
            key={command.id}
            type="button"
            title={command.hint || command.template}
            disabled={disabled}
            onClick={() => start(command)}
            className={cn(
              "rounded-[8px] border px-2.5 py-1.5 text-[12px] transition-colors disabled:opacity-50",
              open?.id === command.id
                ? "border-[var(--vx-border-strong)] bg-[var(--vx-inset)] text-[var(--vx-fg)]"
                : command.danger
                  ? "border-[rgba(224,122,122,0.35)] text-[var(--vx-danger)] hover:border-[rgba(224,122,122,0.6)]"
                  : "border-[var(--vx-border-2)] text-[var(--vx-dim)] hover:border-[var(--vx-border-hover)]"
            )}
          >
            {command.label}
          </button>
        ))}
      </div>

      {open && (
        <div className="mt-3 space-y-2 rounded-[9px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] p-2.5">
          {(open.args ?? []).map((arg) => (
            <label key={arg.name} className="block space-y-1">
              <span className={cn("text-[11.5px]", VX_FAINT)}>
                {arg.label}
                {arg.optional ? ` · ${t("common.optional")}` : ""}
              </span>
              {arg.kind === "player" && players.length > 0 ? (
                <input
                  list={`vx-players-${open.id}-${arg.name}`}
                  value={values[arg.name] ?? ""}
                  onChange={(event) => setValues((prev) => ({ ...prev, [arg.name]: event.target.value }))}
                  placeholder={arg.placeholder}
                  className={VX_INPUT}
                />
              ) : (
                <input
                  value={values[arg.name] ?? ""}
                  onChange={(event) => setValues((prev) => ({ ...prev, [arg.name]: event.target.value }))}
                  placeholder={arg.placeholder}
                  className={VX_INPUT}
                />
              )}
              {arg.kind === "player" && players.length > 0 && (
                <datalist id={`vx-players-${open.id}-${arg.name}`}>
                  {players.map((name) => (
                    <option key={name} value={name} />
                  ))}
                </datalist>
              )}
            </label>
          ))}
          <div className="flex items-center gap-2">
            <Btn
              tone={open.danger ? "danger" : "primary"}
              size="sm"
              disabled={disabled || !ready}
              onClick={() => {
                onRun(fillCommand(open.template, values));
                setOpen(null);
              }}
            >
              <CornerDownLeft className="size-3.5" />
              {t("servers.console.run")}
            </Btn>
            <Btn tone="ghost" size="sm" onClick={() => setOpen(null)}>
              {t("common.cancel")}
            </Btn>
            <span className={cn("ms-auto truncate font-mono text-[11px]", VX_FAINT)}>
              {fillCommand(open.template, values)}
            </span>
          </div>
        </div>
      )}
    </section>
  );
}
