"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminGame } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function formatDate(value?: string) {
  if (!value) return "—";
  return new Date(value).toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function GameDetailContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminGame(id),
    queryFn: () => fetchAdminGame(id),
    enabled: !!id,
  });

  const game = data?.game;

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <Skeleton className="h-32 w-full" />
      </PageShell>
    );
  }

  if (!game) {
    return (
      <PageShell variant="admin">
        <p className="py-20 text-center text-muted-foreground">
          {t("admin.games.not_found")}
        </p>
      </PageShell>
    );
  }

  const active = !!(game.is_active ?? game.status);

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t("admin.games.detail_title", { name: game.name })}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.games.detail_subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link href="/admin/games">← {t("admin.tariffs.to_list")}</Link>
          </Button>
          <Button asChild>
            <Link href={`/admin/games/${id}/edit`}>{t("common.edit")}</Link>
          </Button>
        </div>
      </div>

      <div className="mb-4 grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs text-muted-foreground">
              {t("common.status")}
            </p>
            <Badge
              variant="secondary"
              className={
                active ? "mt-1 bg-emerald-500/10 text-emerald-600" : "mt-1"
              }
            >
              {active
                ? t("admin.locations.active")
                : t("admin.locations.inactive")}
            </Badge>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs text-muted-foreground">Slug</p>
            <p className="mt-1 font-mono font-semibold">{game.slug}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs text-muted-foreground">
              {t("admin.games.created_at")}
            </p>
            <p className="mt-1 font-semibold">{formatDate(game.created_at)}</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">
              {t("admin.games.info_title")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            {[
              [t("common.name"), game.name],
              ["Slug", game.slug],
              [t("common.description"), game.description || "—"],
              [t("admin.games.code"), game.code || "—"],
              [t("admin.games.query"), game.query || "—"],
              [
                t("admin.games.port_range"),
                `${game.minport ?? 1024} — ${game.maxport ?? 65535}`,
              ],
              [t("admin.games.last_update"), formatDate(game.updated_at)],
            ].map(([label, value]) => (
              <div
                key={label}
                className="flex gap-4 border-b py-2 last:border-0"
              >
                <span className="w-40 text-muted-foreground">{label}</span>
                <span className="flex-1 font-medium">{value}</span>
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              {t("admin.games.stats")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b py-2">
              <span className="text-muted-foreground">
                {t("admin.analytics.col_servers")}
              </span>
              <span className="font-semibold">{game.server_count ?? 0}</span>
            </div>
            <div className="flex justify-between border-b py-2">
              <span className="text-muted-foreground">
                {t("admin.games.versions")}
              </span>
              <span className="font-semibold">{game.versions?.length ?? 0}</span>
            </div>
            <div className="flex justify-between py-2">
              <span className="text-muted-foreground">
                {t("common.updated_at")}
              </span>
              <span className="font-semibold">{formatDate(game.updated_at)}</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageShell>
  );
}
