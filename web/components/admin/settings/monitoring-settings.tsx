"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { fetchAdminMonitoringInfo } from "@/lib/api";
import { pollMs } from "@/lib/public-settings";
import { cn } from "@/lib/utils";

import { SettingsCard } from "./settings-ui";

function StatusBadge({ ok }: { ok: boolean }) {
  const t = useT();
  return (
    <Badge
      variant="outline"
      className={cn(ok ? "border-emerald-500/40 text-emerald-600 dark:text-emerald-400" : "border-destructive/40 text-destructive")}
    >
      {ok ? t("admin.settings.monitoring.running") : t("admin.settings.monitoring.down")}
    </Badge>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[140px_minmax(0,1fr)] items-center gap-3 border-b py-3 text-sm last:border-b-0">
      <div className="text-muted-foreground">{label}</div>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

export function MonitoringSettings() {
  const t = useT();
  const [reveal, setReveal] = useState(false);
  const query = useQuery({
    queryKey: ["admin-monitoring-info"],
    queryFn: fetchAdminMonitoringInfo,
    refetchInterval: pollMs(15_000),
    retry: false,
  });

  if (query.isLoading) return <Skeleton className="h-96 w-full" />;
  if (query.isError || !query.data) {
    return (
      <SettingsCard>
        <div className="py-8 text-center text-sm text-muted-foreground">
          {t("admin.settings.monitoring.load_failed")}
        </div>
      </SettingsCard>
    );
  }

  const { grafana, prometheus } = query.data;

  const copy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t("admin.settings.monitoring.copied"));
    } catch {
      toast.error(t("admin.settings.monitoring.load_failed"));
    }
  };

  return (
    <div className="space-y-6">
      <SettingsCard
        title={t("admin.settings.monitoring.grafana_title")}
        description={t("admin.settings.monitoring.grafana_description")}
        action={<StatusBadge ok={grafana.ready} />}
      >
        <Row label={t("admin.settings.monitoring.address")}>
          <div className="flex flex-wrap items-center gap-2">
            <code className="rounded bg-muted px-2 py-1 text-xs">{grafana.url}</code>
            <Button size="sm" variant="outline" asChild>
              <a href={grafana.url} target="_blank" rel="noopener noreferrer">
                {t("admin.settings.monitoring.open")}
              </a>
            </Button>
          </div>
        </Row>
        <Row label={t("admin.settings.monitoring.login")}>
          <code className="rounded bg-muted px-2 py-1 text-xs">{grafana.login}</code>
        </Row>
        <Row label={t("admin.settings.monitoring.password")}>
          {grafana.password ? (
            <div className="flex flex-wrap items-center gap-2">
              <code className="rounded bg-muted px-2 py-1 font-mono text-xs">
                {reveal ? grafana.password : "•".repeat(16)}
              </code>
              <Button size="sm" variant="outline" onClick={() => setReveal((v) => !v)}>
                {reveal ? t("admin.settings.monitoring.hide") : t("admin.settings.monitoring.show")}
              </Button>
              <Button size="sm" variant="outline" onClick={() => void copy(grafana.password)}>
                {t("admin.settings.monitoring.copy")}
              </Button>
            </div>
          ) : (
            <span className="text-muted-foreground">{t("admin.settings.monitoring.no_password")}</span>
          )}
        </Row>
        <div className="mt-3 space-y-1 text-[12.5px] text-muted-foreground">
          <p>{t("admin.settings.monitoring.cert_hint")}</p>
          <p>{t("admin.settings.monitoring.firewall_hint", { port: grafana.port })}</p>
          {grafana.password && <p>{t("admin.settings.monitoring.password_hint")}</p>}
        </div>
      </SettingsCard>

      <SettingsCard
        title={t("admin.settings.monitoring.prometheus_title")}
        description={t("admin.settings.monitoring.prometheus_description")}
        action={<StatusBadge ok={prometheus.ready} />}
      >
        <Row label={t("admin.settings.monitoring.internal_address")}>
          <code className="rounded bg-muted px-2 py-1 text-xs">{prometheus.internal_url}</code>
        </Row>
        <Row label={t("admin.settings.monitoring.retention")}>
          <div>
            {t("admin.settings.monitoring.retention_value", {
              time: prometheus.retention,
              size: prometheus.retention_size,
            })}
          </div>
          <div className="mt-1 text-[12.5px] text-muted-foreground">
            {t("admin.settings.monitoring.retention_hint")}
          </div>
        </Row>
        <Row label={t("admin.settings.monitoring.targets")}>
          {prometheus.targets.length === 0 ? (
            <span className="text-muted-foreground">{t("admin.settings.monitoring.no_targets")}</span>
          ) : (
            <ul className="space-y-1">
              {prometheus.targets.map((target) => (
                <li key={target.job} className="flex flex-wrap items-center gap-2">
                  <code className="rounded bg-muted px-2 py-0.5 text-xs">{target.job}</code>
                  <span
                    className={cn(
                      "text-xs",
                      target.state === "up" ? "text-emerald-600 dark:text-emerald-400" : "text-destructive"
                    )}
                  >
                    {target.state === "up"
                      ? t("admin.settings.monitoring.target_up")
                      : t("admin.settings.monitoring.target_down")}
                  </span>
                  {target.error && <span className="truncate text-xs text-muted-foreground">{target.error}</span>}
                </li>
              ))}
            </ul>
          )}
        </Row>
        <div className="mt-3 flex flex-wrap items-center gap-3 text-[12.5px] text-muted-foreground">
          <span>{t("admin.settings.monitoring.queries_hint")}</span>
          <Button size="sm" variant="outline" asChild>
            <Link href="/admin/performance">{t("admin.settings.monitoring.open_performance")}</Link>
          </Button>
        </div>
      </SettingsCard>
    </div>
  );
}
