"use client";

import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";

import { Panel, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { fetchServerInstallLog } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function ServerInstallConsole({
  serverId,
  active,
}: {
  serverId: string;
  active: boolean;
}) {
  const t = useT();
  const boxRef = useRef<HTMLDivElement>(null);

  const { data } = useQuery({
    queryKey: ["server-install-log", serverId],
    queryFn: () => fetchServerInstallLog(serverId),
    refetchInterval: active ? 2000 : false,
    enabled: !!serverId,
  });

  const lines = data?.lines ?? [];
  const progress = data?.progress;
  const percent = Math.max(0, Math.min(100, Number(progress?.percent ?? 0)));
  const message =
    String(progress?.message || "").trim() ||
    t("servers.overview.install_label_default");
  const stage = String(progress?.stage || "").trim();
  const measured = progress?.derived !== true;

  useEffect(() => {
    if (boxRef.current) {
      boxRef.current.scrollTop = boxRef.current.scrollHeight;
    }
  }, [lines.length]);

  return (
    <Panel
      title={t("servers.install.title")}
      aside={
        <span className={cn("font-mono text-[12px]", VX_MUTED)}>
          {measured
            ? stage
              ? `${stage} · ${percent}%`
              : `${percent}%`
            : stage || t("servers.install.waiting")}
        </span>
      }
      bodyClassName="flex flex-col gap-3 px-[18px] py-3.5"
    >
      <div>
        <div className="flex items-center justify-between gap-3 text-[12.5px] text-[var(--vx-warn)]">
          <span>{message}</span>
        </div>
        <div className="mt-2.5 h-1 overflow-hidden rounded-full bg-[var(--vx-border)]">
          {measured ? (
            <div
              className="h-full bg-[var(--vx-warn)] transition-[width] duration-500"
              style={{ width: `${percent}%` }}
            />
          ) : (
            <div className="h-full w-1/3 animate-pulse rounded-full bg-[var(--vx-warn)]" />
          )}
        </div>
      </div>

      <div
        ref={boxRef}
        className="max-h-[320px] min-h-[120px] overflow-y-auto rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-3 font-mono text-[11.5px] leading-[1.6]"
      >
        {lines.length === 0 ? (
          <div className={VX_FAINT}>
            {active
              ? t("servers.install.awaiting_output")
              : t("servers.install.no_output")}
          </div>
        ) : (
          lines.map((line, i) => (
            <div
              key={`${i}-${line.slice(0, 24)}`}
              className={cn("break-all whitespace-pre-wrap", VX_MUTED)}
            >
              {line}
            </div>
          ))
        )}
      </div>
    </Panel>
  );
}
