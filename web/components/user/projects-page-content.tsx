"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { promptAction } from "@/components/action-dialog";
import { Btn, VX_MUTED, btnClass } from "@/components/vx/panel-ui";
import { createProject, fetchProjects, type UserProject } from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const CARD = "rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]";
const DIM = "text-[var(--vx-faint)]";

function money(value: number, currency: string): string {
  const symbol = currency === "RUB" || !currency ? "₽" : currency;
  return `${formatAmount(value, 0)} ${symbol}`;
}

export function ProjectsPageContent() {
  useT();
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.projects,
    queryFn: fetchProjects,
  });

  const createMutation = useMutation({
    mutationFn: createProject,
    onSuccess: () => {
      toast.success(t("projects.created"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("projects.create_failed")),
  });

  async function onCreate() {
    const name = await promptAction(t("projects.create_prompt"), {
      title: t("projects.create_title"),
      confirmText: t("common.create"),
      placeholder: t("projects.name_placeholder"),
    });
    if (!name || !name.trim()) return;
    createMutation.mutate({ name: name.trim() });
  }

  const projects = data?.projects ?? [];
  const unassigned = data?.unassigned_servers ?? 0;

  return (
    <PageShell variant="user">
      <div className="font-panel flex w-full flex-col gap-5">
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-[240px] flex-1">
            <h1 className="m-0 text-[22px] font-bold">{t("projects.title")}</h1>
            <p className={cn("mt-1 text-[13px]", VX_MUTED)}>{t("projects.subtitle")}</p>
          </div>
          <Btn tone="primary" onClick={() => void onCreate()} disabled={createMutation.isPending}>
            <i className="ri-add-line" />
            {t("projects.create")}
          </Btn>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-[150px] rounded-[14px]" />
            ))}
          </div>
        ) : projects.length === 0 ? (
          <div className={cn(CARD, "flex flex-col items-center gap-3 px-5 py-14 text-center")}>
            <i className="ri-folder-3-line text-[30px] text-[var(--vx-ghost)]" />
            <span className="text-[15px] font-semibold">{t("projects.empty_title")}</span>
            <span className={cn("max-w-[420px] text-[13px]", VX_MUTED)}>
              {t("projects.empty_text")}
            </span>
            <Btn tone="primary" className="mt-1" onClick={() => void onCreate()}>
              <i className="ri-add-line" />
              {t("projects.create")}
            </Btn>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {projects.map((project) => (
                <ProjectCard key={project.id} project={project} />
              ))}
            </div>
            {unassigned > 0 && (
              <div
                className={cn(
                  CARD,
                  "flex flex-wrap items-center justify-between gap-3 px-5 py-4"
                )}
              >
                <span className={cn("text-[13px]", VX_MUTED)}>
                  {t("projects.unassigned", { count: unassigned })}
                </span>
                <Link href="/servers" className={btnClass("default", "sm")}>
                  {t("nav.servers")}
                </Link>
              </div>
            )}
          </>
        )}
      </div>
    </PageShell>
  );
}

function ProjectCard({ project }: { project: UserProject }) {
  return (
    <Link
      href={`/projects/${project.id}`}
      className={cn(
        CARD,
        "flex flex-col gap-3 p-5 transition-colors hover:border-[var(--vx-border-strong)]"
      )}
    >
      <div className="flex items-start gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-[var(--vx-tint)]">
          <i className="ri-folder-3-line text-[18px] text-[var(--vx-fg)]" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[15px] font-semibold">{project.name}</div>
          <div className={cn("mt-0.5 truncate text-[12px]", DIM)}>
            {project.comment || t("projects.no_comment")}
          </div>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[12px]">
        <span className={VX_MUTED}>{t("projects.servers_count", { count: project.servers })}</span>
        <span className={VX_MUTED}>{t("projects.members_count", { count: project.members })}</span>
      </div>

      <div className="mt-auto border-t border-[var(--vx-border)] pt-3 font-mono text-[14px]">
        {t("projects.monthly", { amount: money(project.monthly_cost, project.currency) })}
      </div>
    </Link>
  );
}
