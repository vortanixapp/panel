"use client";

import Link from "next/link";
import { ArrowRight, RotateCcw } from "lucide-react";
import { SettingsCard } from "@/components/admin/settings/settings-ui";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { AgentsList } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const ITEMS = [
  { key: "outdated", filter: "outdated", tone: "text-amber-600 dark:text-amber-500" },
  { key: "updating", filter: "updating", tone: "text-sky-600 dark:text-sky-400" },
  { key: "update_failed", filter: "update_failed", tone: "text-rose-600 dark:text-rose-400" },
  { key: "offline", filter: "offline", tone: "text-rose-600 dark:text-rose-400" },
] as const;

export function AgentsSection({
  data,
  isLoading,
  isError,
  onRetry,
}: {
  data: AgentsList | undefined;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}) {
  const t = useT();
  const summary = data?.summary;
  const versioned = /^\d+\.\d+\.\d+/.test(data?.target_version ?? "");

  return (
    <div id="agents" className="scroll-mt-24">
      <SettingsCard
        title={t("admin.updates.agents.title")}
        description={
          versioned ? t("admin.updates.agents.description", { version: data?.target_version ?? "" }) : undefined
        }
        action={
          <Button asChild variant="outline" size="sm">
            <Link href={summary && summary.outdated > 0 ? "/admin/daemons?filter=outdated" : "/admin/daemons"}>
              {t("admin.updates.agents.open")}
              <ArrowRight />
            </Link>
          </Button>
        }
      >
        {isLoading ? (
          <Skeleton className="h-16 w-full rounded-xl" />
        ) : isError || !summary ? (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-[13px] text-destructive">
            <span>{t("admin.updates.agents.load_failed")}</span>
            <Button size="sm" variant="outline" onClick={onRetry}>
              <RotateCcw />
              {t("common.retry")}
            </Button>
          </div>
        ) : summary.total === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed px-6 py-8 text-center">
            <p className="text-[13.5px] text-muted-foreground">{t("admin.updates.agents.empty")}</p>
            <Button asChild variant="outline" size="sm">
              <Link href="/admin/locations">{t("admin.updates.agents.add_location")}</Link>
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-px overflow-hidden rounded-xl border bg-border sm:grid-cols-5">
            <div className="bg-card px-4 py-3">
              <div className="text-[12px] text-muted-foreground">{t("admin.updates.agents.current")}</div>
              <div className="mt-1 text-[20px] font-semibold tabular-nums">
                {Math.max(0, summary.total - summary.outdated - summary.never_connected)}
                <span className="text-[13px] font-normal text-muted-foreground"> / {summary.total}</span>
              </div>
            </div>
            {ITEMS.map((item) => (
              <Link
                key={item.key}
                href={`/admin/daemons?filter=${item.filter}`}
                className="bg-card px-4 py-3 transition-colors hover:bg-muted/40"
              >
                <div className="text-[12px] text-muted-foreground">{t(`admin.updates.agents.filter.${item.key}`)}</div>
                <div
                  className={cn(
                    "mt-1 text-[20px] font-semibold tabular-nums",
                    summary[item.key] > 0 ? item.tone : "text-muted-foreground"
                  )}
                >
                  {summary[item.key]}
                </div>
              </Link>
            ))}
          </div>
        )}
        {data && !data.auto_enabled && summary && summary.outdated > 0 && (
          <p className="mt-3 text-[12px] leading-relaxed text-muted-foreground">{t("admin.updates.agents.auto_hint")}</p>
        )}
      </SettingsCard>
    </div>
  );
}
