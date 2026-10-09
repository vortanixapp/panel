"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight, Check, Copy, Loader2, Play, Plus, Square, SquareTerminal } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { pluralDays } from "@/components/user/panel-parts";
import { useQuery } from "@tanstack/react-query";
import { livePollMs } from "@/lib/live-link";
import { fetchServerMetrics, type DashboardData, type DashboardServer } from "@/lib/api";
import { usePowerServer } from "@/hooks/use-queries";
import { canStartServer, canStopServer, getServerStatus, getServerStatusDotClass } from "@/lib/server-status";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { daysUntil, serverAddress, shortDate } from "./dashboard-utils";

export function ServersCard({ d }: { d: DashboardData }) {
  const t = useT();
  const servers = d.recent_servers;
  const more = d.total_servers - servers.length;

  return (
    <section className="srv2-rise overflow-hidden rounded-[22px] border bg-card" style={{ animationDelay: "300ms" }}>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b px-6 py-[18px]">
        <h2 className="text-[16px] font-semibold">{t("home.servers.title")}</h2>
        {d.total_servers > 0 && (
          <span className="font-mono text-[12px] text-muted-foreground">
            {t("home.servers.running", { running: d.active_servers, total: d.total_servers })}
          </span>
        )}
        {d.total_servers > 0 && (
          <Link href="/servers" className="group ms-auto inline-flex items-center gap-1.5 text-[13px] font-medium">
            {t("home.servers.all")}
            <ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
          </Link>
        )}
      </div>
      {servers.length === 0 ? (
        <Onboarding />
      ) : (
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,260px),1fr))] gap-3.5 p-[18px]">
          {servers.map((server) => (
            <ServerItem key={server.id} server={server} />
          ))}
          <Link
            href="/rent-server"
            className="flex min-h-[150px] flex-col items-center justify-center gap-2.5 rounded-2xl border border-dashed p-4 text-center text-muted-foreground transition-colors hover:border-foreground hover:text-foreground"
          >
            <span className="srv2-float grid size-10 place-items-center rounded-full bg-muted text-foreground">
              <Plus className="size-5" />
            </span>
            {t("home.servers.add")}
            <span className="text-[12px] text-muted-foreground/70">{t("home.servers.add_hint")}</span>
          </Link>
        </div>
      )}
      {more > 0 && (
        <div className="border-t px-6 py-3 text-[12.5px]">
          <Link href="/servers" className="text-muted-foreground hover:text-foreground">
            {t("home.servers.more", { count: more })}
          </Link>
        </div>
      )}
    </section>
  );
}

function GameIcon({ server }: { server: DashboardServer }) {
  const image = server.game?.image;
  const letter = (server.game?.name || server.name || "?").trim().charAt(0).toUpperCase();
  return image ? (
    <img src={image} alt="" className="size-10 shrink-0 rounded-xl border object-cover" />
  ) : (
    <span className="grid size-10 shrink-0 place-items-center rounded-xl border bg-muted text-[15px] font-semibold text-muted-foreground">
      {letter}
    </span>
  );
}

function Meter({ label, value, text }: { label: string; value?: number; text?: string }) {
  if (value === undefined || value === null || !Number.isFinite(value)) return null;
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="min-w-0 font-mono">
      <div className="text-[10px] tracking-[0.1em] text-muted-foreground/70 uppercase">{label}</div>
      <div className="mt-0.5 mb-1.5 text-[14px] text-foreground">{text ?? `${Math.round(pct)}%`}</div>
      <div className="h-1 overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full", pct >= 90 ? "bg-rose-500" : pct >= 70 ? "bg-amber-500" : "bg-foreground/70")}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function LoadSpark({ serverId, running }: { serverId: string; running: boolean }) {
  const { data } = useQuery({
    queryKey: ["dashboard-spark", serverId],
    queryFn: () => fetchServerMetrics(serverId),
    enabled: running,
    staleTime: 60_000,
    refetchInterval: () => livePollMs(60_000, 300_000),
    retry: false,
  });
  const values = (data ?? []).slice(-48).map((p) => Math.max(0, Math.min(100, p.cpu_pct)));
  const W = 200;
  const H = 44;
  const pts = values.length > 1 ? values : [0, 0];
  const line = pts
    .map((v, i) => `${i === 0 ? "M" : "L"}${((i / (pts.length - 1)) * W).toFixed(1)} ${(H - 4 - (v / 100) * (H - 10)).toFixed(1)}`)
    .join("");
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="block h-11 w-full text-foreground/70" aria-hidden>
      <path d={`${line}L${W} ${H}L0 ${H}Z`} fill="currentColor" opacity="0.12" />
      <path d={line} fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function ServerItem({ server }: { server: DashboardServer }) {
  const t = useT();
  const [copied, setCopied] = useState(false);
  const st = getServerStatus(server);
  const address = serverAddress(server.ip_address, server.port);
  const days = daysUntil(server.expires_at);
  const expiring = days !== null && days > 0 && days <= 7;
  const expired = days !== null && days <= 0;
  const canStart = canStartServer(server);
  const canStop = canStopServer(server);

  const powerHook = usePowerServer();
  const power = {
    isPending: powerHook.isPending,
    mutate: (action: "start" | "stop") =>
      powerHook.mutate(
        { id: server.id, action },
        {
          onSuccess: () =>
            toast.success(
              action === "start"
                ? t("home.servers.starting", { name: server.name })
                : t("home.servers.stopping", { name: server.name })
            ),
          onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
        }
      ),
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error(t("home.servers.copy_failed"));
    }
  };

  const online = server.online_players;
  const max = server.max_players;
  const playersPct = online !== undefined && max ? (online / max) * 100 : undefined;
  const playersText = online !== undefined ? `${online}${max ? `/${max}` : ""}` : undefined;

  return (
    <div data-testid="server-card" data-server-id={server.id} className="flex flex-col gap-3.5 rounded-2xl border bg-muted/20 p-4 transition-colors hover:border-foreground/30">
      <div className="flex items-center gap-2.5">
        <GameIcon server={server} />
        <div className="min-w-0 flex-1">
          <Link href={`/servers/${server.id}`} className="block truncate font-semibold hover:underline">
            {server.name}
          </Link>
          <div className="truncate text-[12px] text-muted-foreground">
            {[server.game?.name, server.location?.city || server.location?.name].filter(Boolean).join(" · ") || "—"}
          </div>
        </div>
        <span className="inline-flex items-center gap-[7px] text-[12px]">
          <span className={cn("relative size-[7px] rounded-full", getServerStatusDotClass(st), st.category === "running" && "vx-live")} />
          {st.label}
        </span>
      </div>

      <LoadSpark serverId={server.id} running={st.category === "running"} />

      {(server.cpu_percent !== undefined || server.ram_percent !== undefined || playersText) && (
        <div className="grid grid-cols-3 gap-2.5">
          <Meter label="CPU" value={server.cpu_percent} />
          <Meter label="RAM" value={server.ram_percent} />
          <Meter label={t("home.servers.players")} value={playersPct ?? (online !== undefined ? 0 : undefined)} text={playersText} />
        </div>
      )}

      <div className="text-[12.5px]">
        {server.billing_source === "whmcs" ? (
          <span className="text-muted-foreground">{t("home.servers.whmcs")}</span>
        ) : days === null ? (
          <span className="text-muted-foreground">{t("home.servers.no_expiry")}</span>
        ) : expired ? (
          <span className="font-medium text-rose-600 dark:text-rose-400">{t("home.servers.expired")}</span>
        ) : (
          <span className={cn(expiring ? "font-medium text-amber-600 dark:text-amber-500" : "text-muted-foreground")}>
            {t("home.servers.until", { date: shortDate(server.expires_at) })} ·{" "}
            {server.auto_renew ? t("home.servers.auto") : t("home.servers.left", { days: pluralDays(days) })}
          </span>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-3">
        <button
          type="button"
          onClick={() => void copy()}
          className="inline-flex min-w-0 items-center gap-1.5 truncate font-mono text-[12px] text-muted-foreground hover:text-foreground"
          title={t("home.servers.copy")}
        >
          {address}
          {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
        </button>
        <div className="flex items-center gap-1.5">
          {(expiring || expired) && server.billing_source !== "whmcs" && (
            <Button asChild size="sm" variant={expired ? "default" : "outline"} className="h-8">
              <Link href={`/servers/${server.id}/tariff`}>{t("home.servers.renew")}</Link>
            </Button>
          )}
          {canStart ? (
            <Button
              variant="outline"
              size="icon"
              className="size-8 text-emerald-600 hover:text-emerald-600 dark:text-emerald-400"
              title={t("home.servers.start")}
              aria-label={t("home.servers.start")}
              disabled={power.isPending || server.is_blocked}
              onClick={() => power.mutate("start")}
            >
              {power.isPending ? <Loader2 className="animate-spin" /> : <Play />}
            </Button>
          ) : (
            <Button
              variant="outline"
              size="icon"
              className="size-8 text-rose-600 hover:text-rose-600 dark:text-rose-400"
              title={t("home.servers.stop")}
              aria-label={t("home.servers.stop")}
              disabled={power.isPending || !canStop}
              onClick={() => power.mutate("stop")}
            >
              {power.isPending ? <Loader2 className="animate-spin" /> : <Square />}
            </Button>
          )}
          <Button asChild variant="outline" size="icon" className="size-8" title={t("home.servers.console")}>
            <Link href={`/servers/${server.id}/console`} aria-label={t("home.servers.console")}>
              <SquareTerminal />
            </Link>
          </Button>
          <Button asChild size="sm" variant="outline" className="h-8">
            <Link href={`/servers/${server.id}`}>{t("home.servers.manage")}</Link>
          </Button>
        </div>
      </div>
    </div>
  );
}

function Onboarding() {
  const t = useT();
  const steps = [
    { icon: "ri-wallet-3-line", title: t("home.onboarding.step1"), text: t("home.onboarding.step1_hint") },
    { icon: "ri-gamepad-line", title: t("home.onboarding.step2"), text: t("home.onboarding.step2_hint") },
    { icon: "ri-rocket-2-line", title: t("home.onboarding.step3"), text: t("home.onboarding.step3_hint") },
  ];
  return (
    <div className="px-5 py-7 sm:px-8 sm:py-9">
      <h3 className="text-[18px] font-semibold tracking-tight">{t("home.onboarding.title")}</h3>
      <p className="mt-1.5 max-w-xl text-[13.5px] leading-relaxed text-muted-foreground">{t("home.onboarding.text")}</p>
      <ol className="mt-6 grid gap-3 sm:grid-cols-3">
        {steps.map((step, index) => (
          <li key={step.title} className="rounded-xl border bg-muted/30 p-4">
            <div className="flex items-center gap-2.5">
              <span className="grid size-7 place-items-center rounded-lg bg-background text-[12px] font-semibold">
                {index + 1}
              </span>
              <i className={cn(step.icon, "text-[17px] text-muted-foreground")} />
            </div>
            <div className="mt-3 text-[13.5px] font-medium">{step.title}</div>
            <p className="mt-1 text-[12.5px] leading-snug text-muted-foreground">{step.text}</p>
          </li>
        ))}
      </ol>
      <div className="mt-6 flex flex-wrap gap-2">
        <Button asChild>
          <Link href="/rent-server">
            <Plus />
            {t("home.onboarding.rent")}
          </Link>
        </Button>
        <Button asChild variant="outline">
          <Link href="/billing#topup">{t("home.onboarding.topup")}</Link>
        </Button>
      </div>
    </div>
  );
}
