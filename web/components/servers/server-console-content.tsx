"use client";

import { useRef, useState } from "react";
import { useParams } from "next/navigation";
import { toast } from "sonner";
import dynamic from "next/dynamic";
import type { ConsoleStatus, ServerConsoleHandle } from "@/components/ServerConsole";

// Терминал тянет @xterm, а тот на верхнем уровне модуля обращается к self.
// При серверном рендере self не существует, и страница падала с
// "self is not defined" — в логе панели это висело необработанным отказом.
const ServerConsole = dynamic(
  () => import("@/components/ServerConsole").then((m) => m.ServerConsole),
  { ssr: false },
);
import { ServerTabShell } from "@/components/servers/server-tab-shell";
import { Btn, Panel, VX_INPUT_MONO, VX_MUTED } from "@/components/vx/panel-ui";
import type { PanelVariant } from "@/lib/panel-paths";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

// Собирается на каждый рендер: в константе модуля подписи застыли бы на языке,
// который стоял в момент импорта, и не менялись бы при переключении языка.
function statusText(t: TranslateFn): Record<ConsoleStatus, string> {
  return {
    connecting: t("servers.console.status_connecting"),
    connected: t("servers.console.status_connected"),
    disconnected: t("servers.console.status_disconnected"),
    error: t("servers.console.status_error"),
  };
}

const STATUS_DOT: Record<ConsoleStatus, string> = {
  connecting: "bg-[var(--vx-warn)]",
  connected: "bg-[var(--vx-info)]",
  disconnected: "bg-[var(--vx-faint)]",
  error: "bg-[var(--vx-danger)]",
};

export function ServerConsoleContent({ variant = "user" }: { variant?: PanelVariant }) {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const consoleRef = useRef<ServerConsoleHandle>(null);
  const [status, setStatus] = useState<ConsoleStatus>("connecting");
  const [command, setCommand] = useState("");

  function send() {
    const value = command.trim();
    if (!value) return;
    if (!consoleRef.current?.send(value)) {
      toast.error(t("servers.console.not_connected"));
      return;
    }
    setCommand("");
  }

  return (
    <ServerTabShell variant={variant} activeTab="console">
      <Panel
        title={t("server.tab.console")}
        aside={
          <span className={cn("inline-flex items-center gap-[7px] font-mono text-[11px]", VX_MUTED)}>
            <span className={cn("h-1.5 w-1.5 rounded-full", STATUS_DOT[status])} />
            {statusText(t)[status]}
          </span>
        }
      >
        <ServerConsole
          ref={consoleRef}
          serverId={id}
          onStatusChange={setStatus}
          className="h-[min(60vh,420px)]"
        />
        <div className="mt-3 flex gap-2">
          <input
            className={cn(VX_INPUT_MONO, "h-9 flex-1")}
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                send();
              }
            }}
            placeholder={t("servers.console.command_placeholder")}
          />
          <Btn
            tone="primary"
            className="h-9 px-[18px]"
            onClick={send}
            disabled={status !== "connected"}
          >
            {t("common.send")}
          </Btn>
        </div>
      </Panel>
    </ServerTabShell>
  );
}
