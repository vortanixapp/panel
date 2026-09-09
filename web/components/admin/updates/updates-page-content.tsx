"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, RefreshCw } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminUpdates } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const UPGRADE_FROM_SOURCE = `git pull
docker compose -f deploy/docker-compose.yml up -d --build`;

const UPGRADE_FROM_IMAGES = `docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml pull
docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml up -d`;

function formatDate(value: string | undefined): string {
  if (!value) return "—";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function UpdatesPageContent() {
  const t = useT();
  const queryClient = useQueryClient();

  const updates = useQuery({
    queryKey: ["admin-updates"],
    queryFn: () => fetchAdminUpdates(),
    staleTime: 5 * 60_000,
  });

  const data = updates.data;
  const available = data?.update_available === true;

  async function recheck() {
    const fresh = await fetchAdminUpdates(true);
    queryClient.setQueryData(["admin-updates"], fresh);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.updates.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.updates.subtitle")}
          </p>
        </div>
        <Button
          variant="outline"
          onClick={recheck}
          disabled={updates.isFetching}
        >
          <RefreshCw className={updates.isFetching ? "animate-spin" : ""} />
          {t("admin.updates.recheck")}
        </Button>
      </div>

      {updates.isLoading ? (
        <Skeleton className="h-40 w-full rounded-xl" />
      ) : (
        <div className="grid gap-4">
          <div className="rounded-xl border border-border p-6">
            <div className="flex flex-wrap items-center gap-x-8 gap-y-4">
              <div>
                <div className="text-xs tracking-wide text-muted-foreground uppercase">
                  {t("admin.updates.installed")}
                </div>
                <div className="mt-1 font-mono text-lg font-semibold">
                  {data?.current_version || "—"}
                </div>
              </div>
              <div>
                <div className="text-xs tracking-wide text-muted-foreground uppercase">
                  {t("admin.updates.latest")}
                </div>
                <div className="mt-1 font-mono text-lg font-semibold">
                  {data?.latest_version || "—"}
                </div>
              </div>
              <div className="ms-auto">
                {data?.checks_disabled ? (
                  <Badge variant="secondary">
                    {t("admin.updates.checks_disabled")}
                  </Badge>
                ) : data?.error ? (
                  <Badge variant="destructive">{data.error}</Badge>
                ) : available ? (
                  <Badge>{t("admin.updates.available")}</Badge>
                ) : (
                  <Badge variant="secondary">
                    {t("admin.updates.up_to_date")}
                  </Badge>
                )}
              </div>
            </div>

            <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 border-t border-border pt-4 text-xs text-muted-foreground">
              <span>
                {t("admin.updates.checked_at", {
                  when: formatDate(data?.checked_at),
                })}
              </span>
              {data?.published_at && (
                <span>
                  {t("admin.updates.published_at", {
                    when: formatDate(data.published_at),
                  })}
                </span>
              )}
              {data?.repo && <span className="font-mono">{data.repo}</span>}
              {data?.release_url && (
                <a
                  href={data.release_url}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="inline-flex items-center gap-1 text-primary hover:underline"
                >
                  {t("admin.updates.open_release")}
                  <ExternalLink className="size-3" />
                </a>
              )}
            </div>
          </div>

          {data?.notes && (
            <div className="rounded-xl border border-border p-6">
              <h2 className="text-sm font-semibold">
                {t("admin.updates.notes")}
              </h2>
              <pre className="mt-3 max-h-[420px] overflow-auto text-[13px] leading-relaxed whitespace-pre-wrap text-muted-foreground">
                {data.notes}
              </pre>
            </div>
          )}

          <div className="rounded-xl border border-border p-6">
            <h2 className="text-sm font-semibold">{t("admin.updates.how")}</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {t("admin.updates.how_hint")}
            </p>

            <div className="mt-4 grid gap-4 lg:grid-cols-2">
              <div>
                <div className="text-xs font-medium tracking-wide uppercase">
                  {t("admin.updates.from_source")}
                </div>
                <pre className="mt-2 overflow-x-auto rounded-lg bg-[var(--vx-surface-2)] p-4 font-mono text-[12.5px]">
                  {UPGRADE_FROM_SOURCE}
                </pre>
              </div>
              <div>
                <div className="text-xs font-medium tracking-wide uppercase">
                  {t("admin.updates.from_images")}
                </div>
                <pre className="mt-2 overflow-x-auto rounded-lg bg-[var(--vx-surface-2)] p-4 font-mono text-[12.5px]">
                  {UPGRADE_FROM_IMAGES}
                </pre>
              </div>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              {t("admin.updates.backup_first")}
            </p>
          </div>
        </div>
      )}
    </PageShell>
  );
}
