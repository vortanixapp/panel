"use client";

import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Activity, BookOpen, CopyCheck, Cpu, ExternalLink, LifeBuoy, MessageSquare, RotateCcw } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { btnGhost } from "@/components/user/panel-parts";
import { useT } from "@/hooks/use-translations";
import { fetchBugReportSimilar, type BugReportInfo, type BugReportIssue } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export type EnvRow = { label: string; value: string };

export function EnvironmentCard({
  rows,
  attachLogs,
  onAttachLogs,
}: {
  rows: EnvRow[];
  attachLogs: boolean;
  onAttachLogs: (value: boolean) => void;
}) {
  const t = useT();
  return (
    <section className="flex flex-col gap-3.5 rounded-2xl border border-border bg-card p-[18px]">
      <div className="flex items-center gap-2">
        <Cpu className="h-4 w-4 text-muted-foreground" />
        <span className="text-[13.5px] font-semibold">{t("admin.bug.env")}</span>
        <span className="ms-auto text-[11px] text-muted-foreground/70">{t("admin.bug.env_auto")}</span>
      </div>
      <div className="flex flex-col">
        {rows.map((row) => (
          <div
            key={row.label}
            className="flex min-w-0 items-center gap-3 border-b border-border/60 py-[7px] last:border-b-0"
          >
            <span className="shrink-0 text-[12.5px] text-muted-foreground">{row.label}</span>
            <span className="ms-auto min-w-0 truncate text-end text-[12.5px] font-medium" title={row.value}>
              {row.value || "—"}
            </span>
          </div>
        ))}
      </div>
      <label className="flex cursor-pointer items-start gap-2.5">
        <Switch checked={attachLogs} onCheckedChange={onAttachLogs} className="mt-px" />
        <span className="flex flex-col gap-0.5">
          <span className="text-[12.5px]">{t("admin.bug.attach_logs")}</span>
          <span className="text-[11.5px] leading-snug text-muted-foreground">{t("admin.bug.attach_logs_hint")}</span>
        </span>
      </label>
    </section>
  );
}

function issueState(issue: BugReportIssue): { key: string; tone: string } {
  if (issue.state === "open") return { key: "admin.bug.issue_open", tone: "text-emerald-600 dark:text-emerald-400" };
  if (issue.state_reason === "completed") return { key: "admin.bug.issue_fixed", tone: "text-violet-600 dark:text-violet-400" };
  if (issue.state_reason === "not_planned") return { key: "admin.bug.issue_not_planned", tone: "text-muted-foreground" };
  return { key: "admin.bug.issue_closed", tone: "text-muted-foreground" };
}

function IssueRow({ issue }: { issue: BugReportIssue }) {
  const t = useT();
  const state = issueState(issue);
  const updated = issue.updated_at
    ? new Date(issue.updated_at).toLocaleDateString(localeTag(), { day: "numeric", month: "short" })
    : "";
  return (
    <a
      href={issue.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex flex-col gap-1.5 rounded-xl border border-border bg-background px-3 py-[11px] transition-colors hover:border-ring"
    >
      <span className="line-clamp-2 text-[12.5px] leading-snug font-medium">{issue.title}</span>
      <span className="flex min-w-0 items-center gap-2 text-[11.5px] text-muted-foreground">
        <span className="font-mono">#{issue.number}</span>
        {issue.comments > 0 && (
          <span className="inline-flex items-center gap-1">
            <MessageSquare className="h-3 w-3" />
            {issue.comments}
          </span>
        )}
        {updated && <span className="truncate">{updated}</span>}
        <span className={cn("ms-auto inline-flex flex-none items-center gap-1.5 font-medium", state.tone)}>
          <span className="h-1.5 w-1.5 rounded-full bg-current" />
          {t(state.key)}
        </span>
      </span>
    </a>
  );
}

export function SimilarCard({ query, info }: { query: string; info?: BugReportInfo }) {
  const t = useT();
  const similar = useQuery({
    queryKey: ["admin-bug-report-similar", query],
    queryFn: () => fetchBugReportSimilar(query),
    staleTime: 600_000,
    retry: false,
    placeholderData: keepPreviousData,
  });
  const matched = similar.data?.matched ?? query.trim() !== "";
  const issues = similar.data?.issues ?? [];

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-[18px]">
      <div className="flex items-center gap-2">
        <CopyCheck className="h-4 w-4 text-muted-foreground" />
        <span className="text-[13.5px] font-semibold">
          {matched ? t("admin.bug.similar") : t("admin.bug.similar_open")}
        </span>
        {info?.issues_url && (
          <a
            href={info.issues_url}
            target="_blank"
            rel="noopener noreferrer"
            className="ms-auto inline-flex items-center gap-1 text-[11.5px] text-muted-foreground hover:text-foreground"
          >
            {t("admin.bug.similar_all")}
            <ExternalLink className="h-3 w-3" />
          </a>
        )}
      </div>
      <p className="text-xs leading-relaxed text-muted-foreground/80">{t("admin.bug.similar_hint")}</p>
      {similar.isPending ? (
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-[62px] w-full rounded-xl" />
          ))}
        </div>
      ) : similar.isError ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-[11.5px]" style={{ color: "var(--vx-danger)" }}>
            {t("admin.bug.similar_failed", {
              message: similar.error instanceof Error ? similar.error.message : String(similar.error),
            })}
          </span>
          <button
            type="button"
            onClick={() => void similar.refetch()}
            className={cn(btnGhost, "h-7 px-2.5 text-[11.5px]")}
          >
            <RotateCcw className="h-3 w-3" />
            {t("common.retry")}
          </button>
        </div>
      ) : issues.length === 0 ? (
        <span className="text-[12px] text-muted-foreground/80">
          {matched ? t("admin.bug.similar_empty") : t("admin.bug.similar_open_empty")}
        </span>
      ) : (
        <div className={cn("flex flex-col gap-2 transition-opacity", similar.isPlaceholderData && "opacity-60")}>
          {issues.map((issue) => (
            <IssueRow key={issue.number} issue={issue} />
          ))}
        </div>
      )}
    </section>
  );
}

export function UrgentCard() {
  const t = useT();
  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-[18px]">
      <div className="flex items-center gap-2">
        <LifeBuoy className="h-4 w-4 text-muted-foreground" />
        <span className="text-[13.5px] font-semibold">{t("admin.bug.urgent")}</span>
      </div>
      <p className="text-[12.5px] leading-relaxed text-muted-foreground">{t("admin.bug.urgent_hint")}</p>
      <div className="flex flex-col gap-[7px]">
        <Link href="/admin/nodes" className={cn(btnGhost, "h-[34px] justify-start text-[12.5px]")}>
          <Activity className="h-3.5 w-3.5" />
          {t("admin.bug.nodes_status")}
        </Link>
        <Link href="/admin/support/kb" className={cn(btnGhost, "h-[34px] justify-start text-[12.5px]")}>
          <BookOpen className="h-3.5 w-3.5" />
          {t("admin.bug.kb")}
        </Link>
      </div>
    </section>
  );
}
