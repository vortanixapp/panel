"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, Check, Copy, Loader2, Play, Plus, Square, SquareTerminal } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { pluralDays } from "@/components/user/panel-parts";
import { powerServer, type DashboardData, type DashboardServer } from "@/lib/api";
import { canStartServer, canStopServer, getServerStatus, getServerStatusDotClass } from "@/lib/server-status";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { daysUntil, serverAddress, shortDate } from "./dashboard-utils";

export function ServersCard({ d }: { d: DashboardData }) {
  const t = useT();
  const servers = d.recent_servers;
  const more = d.total_servers - servers.length;

  return (
    <section className="overflow-hidden rounded-2xl border bg-card">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b px-5 py-3.5 sm:px-6">
        <h2 className="text-[15px] font-semibold">{t("home.servers.title")}</h2>
        {d.total_servers > 0 && (
          <span className="font-mono text-[12px] text-muted-foreground">
            {t("home.servers.running", { running: d.active_servers, total: d.total_servers })}
          </span>
        )}
        {d.total_servers > 0 && (
          <Link href="/servers" className="group ms-auto inline-flex items-center gap-1.5 text-[13px] font-medium text-primary">
            {t("home.servers.all")}
            <ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
          </Link>
        )}
      </div>
      {servers.length === 0 ? (
        <Onboarding />
      ) : (
        <ul className="divide-y">
          {servers.map((server) => (
            <ServerItem key={server.id} server={server} />
          ))}
        </ul>
      )}
      {more > 0 && (
        <div className="border-t px-5 py-3 text-[12.5px] sm:px-6">
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

function Meter({ label, value }: { label: string; value?: number }) {
  if (value === undefined || value === null || !Number.isFinite(value)) return null;
  const pct = Math.max(0, Math.min(100, value));
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[11.5px] text-muted-foreground">
      {label}
      <span className="h-1 w-10 overflow-hidden rounded-full bg-muted">
        <span
          className={cn("block h-full rounded-full", pct >= 90 ? "bg-rose-500" : pct >= 70 ? "bg-amber-500" : "bg-emerald-500")}
          style={{ width: `${pct}%` }}
        />
      </span>
      {Math.round(pct)}%
    </span>
  );
}

function ServerItem({ server }: { server: DashboardServer }) {
  const t = useT();
  const qc = useQueryClient();
  const [copied, setCopied] = useState(false);
  const st = getServerStatus(server);
  const address = serverAddress(server.ip_address, server.port);
  const days = daysUntil(server.expires_at);
  const expiring = days !== null && days > 0 && days <= 7;
  const expired = days !== null && days <= 0;
  const canStart = canStartServer(server);
  const canStop = canStopServer(server);

  const power = useMutation({
    mutationFn: (action: "start" | "stop") => powerServer(server.id, action),
    onSuccess: (_res, action) => {
      toast.success(action === "start" ? t("home.servers.starting", { name: server.name }) : t("home.servers.stopping", { name: server.name }));
      void qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error(t("home.servers.copy_failed"));
    }
  };

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-3 px-5 py-4 sm:px-6 lg:grid lg:grid-cols-[minmax(0,1fr)_150px_130px_176px]">
      <div className="flex min-w-0 flex-1 basis-64 items-center gap-3">
        <GameIcon server={server} />
        <div className="min-w-0">
          <Link href={`/servers/${server.id}`} className="block truncate text-[14.5px] font-medium hover:underline">
            {server.name}
          </Link>
          <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-2 text-[12.5px] text-muted-foreground">
            <span className="truncate">
              {[server.game?.name, server.location?.city || server.location?.name].filter(Boolean).join(" · ") || "—"}
            </span>
            {address && (
              <button
                type="button"
                onClick={() => void copy()}
                className="inline-flex items-center gap-1 font-mono hover:text-foreground"
                title={t("home.servers.copy")}
              >
                {address}
                {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="flex min-w-[140px] flex-col gap-1.5 max-sm:basis-full lg:min-w-0">
        <span className="inline-flex items-center gap-2 text-[13px]">
          <span className={cn("relative size-1.5 rounded-full", getServerStatusDotClass(st), st.category === "running" && "vx-live")} />
          {st.label}
        </span>
        {(server.cpu_percent !== undefined || server.ram_percent !== undefined) && (
          <span className="flex flex-wrap gap-x-3 gap-y-1">
            <Meter label="CPU" value={server.cpu_percent} />
            <Meter label="RAM" value={server.ram_percent} />
          </span>
        )}
      </div>

      <div className="min-w-[120px] text-[12.5px] lg:min-w-0">
        {server.billing_source === "whmcs" ? (
          <span className="text-muted-foreground">{t("home.servers.whmcs")}</span>
        ) : days === null ? (
          <span className="text-muted-foreground">{t("home.servers.no_expiry")}</span>
        ) : expired ? (
          <span className="font-medium text-rose-600 dark:text-rose-400">{t("home.servers.expired")}</span>
        ) : (
          <>
            <div className={cn(expiring ? "font-medium text-amber-600 dark:text-amber-500" : "text-foreground")}>
              {t("home.servers.until", { date: shortDate(server.expires_at) })}
            </div>
            <div className="text-muted-foreground">
              {server.auto_renew ? t("home.servers.auto") : t("home.servers.left", { days: pluralDays(days) })}
            </div>
          </>
        )}
      </div>

      <div className="ms-auto flex items-center justify-end gap-1.5">
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
      </div>
    </li>
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
