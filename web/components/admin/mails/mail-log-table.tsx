"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { fetchMailLog } from "@/lib/api";
import { cn } from "@/lib/utils";

const FILTERS = ["", "sent", "failed", "skipped"] as const;

const TONE: Record<string, string> = {
  sent: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  failed: "bg-destructive/10 text-destructive",
  skipped: "bg-muted text-muted-foreground",
};

function when(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function MailLogTable() {
  const t = useT();
  const [status, setStatus] = useState<(typeof FILTERS)[number]>("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-mail-log", status],
    queryFn: () => fetchMailLog(status),
    refetchInterval: 30_000,
  });

  const items = data?.items ?? [];
  const week = data?.week;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-1.5">
          {FILTERS.map((value) => (
            <button
              key={value || "all"}
              type="button"
              onClick={() => setStatus(value)}
              className={cn(
                "rounded-[9px] border px-3 py-1.5 text-[12.5px] transition-colors",
                status === value
                  ? "border-primary bg-primary/10 text-foreground"
                  : "border-input text-muted-foreground hover:bg-accent"
              )}
            >
              {t(`admin.mails.log_${value || "all"}`)}
            </button>
          ))}
        </div>
        {week && (
          <div className="text-[12.5px] text-muted-foreground">
            {t("admin.mails.log_week", {
              sent: String(week.sent),
              failed: String(week.failed),
              skipped: String(week.skipped),
            })}
          </div>
        )}
      </div>

      <div className="vx-tbl-wrap rounded-2xl border bg-card">
        {isLoading ? (
          <div className="space-y-2 p-4">
            {[0, 1, 2, 3].map((row) => (
              <Skeleton key={row} className="h-9 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="p-8 text-center text-[13px] text-muted-foreground">
            {t("admin.mails.log_empty")}
          </div>
        ) : (
          <table className="vx-tbl w-full text-[13px]">
            <thead>
              <tr className="border-b bg-muted/30 text-[12px] text-muted-foreground">
                <th className="px-4 py-2 text-left">{t("admin.mails.log_when")}</th>
                <th className="px-4 py-2 text-left">{t("admin.mails.log_to")}</th>
                <th className="px-4 py-2 text-left">{t("admin.mails.log_subject")}</th>
                <th className="px-4 py-2 text-left">{t("admin.mails.log_status")}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-b last:border-0 align-top">
                  <td
                    className="px-4 py-2 whitespace-nowrap text-muted-foreground"
                    data-label={t("admin.mails.log_when")}
                  >
                    {when(item.created_at)}
                  </td>
                  <td className="px-4 py-2" data-label={t("admin.mails.log_to")}>
                    <div className="truncate">{item.to}</div>
                    <div className="text-[11.5px] text-muted-foreground">{item.template}</div>
                  </td>
                  <td className="px-4 py-2" data-cell="full">
                    <div className="truncate">{item.subject}</div>
                    {item.error && (
                      <div className="mt-0.5 text-[11.5px] text-destructive">{item.error}</div>
                    )}
                  </td>
                  <td className="px-4 py-2" data-label={t("admin.mails.log_status")}>
                    <span
                      className={cn(
                        "rounded-full px-2 py-0.5 text-[11.5px]",
                        TONE[item.status] ?? TONE.skipped
                      )}
                    >
                      {t(`admin.mails.log_${item.status}`)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
