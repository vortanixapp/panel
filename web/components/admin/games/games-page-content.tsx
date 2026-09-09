"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Gamepad2, Pencil, Pause, Play, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  deleteAdminGame,
  fetchAdminGames,
  toggleAdminGame,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

export function GamesPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminGames,
    queryFn: fetchAdminGames,
  });
  const games = data?.games ?? [];

  const toggleMut = useMutation({
    mutationFn: toggleAdminGame,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminGames });
      toast.success(t("admin.locations.status_updated"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: deleteAdminGame,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminGames });
      toast.success(t("admin.games.deleted"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onDelete(id: string, name: string) {
    if (!confirm(t("admin.games.delete_confirm", { name }))) return;
    await deleteMut.mutateAsync(id);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.games.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.games.subtitle")}
          </p>
        </div>
        <Button asChild>
          <Link href="/admin/games/create">
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.games.add")}
          </Link>
        </Button>
      </div>

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : games.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <Gamepad2 className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              {t("admin.games.empty")}
            </p>
          </div>
        ) : (
          <div className="w-full overflow-x-auto">
            <table className="min-w-[920px] w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40">
                  <th className="px-4 py-3 text-left font-medium">
                    {t("common.id")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("common.game")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">Slug</th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("common.status")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("admin.analytics.col_servers")}
                  </th>
                  <th className="px-4 py-3 text-right font-medium">
                    {t("common.actions")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {games.map((g) => (
                  <tr key={g.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {g.id.slice(0, 8)}
                    </td>
                    <td className="px-4 py-3 font-medium">{g.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {g.slug}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap items-center gap-1">
                        {g.is_active ? (
                          <Badge
                            variant="secondary"
                            className="bg-emerald-500/10 text-emerald-600"
                          >
                            {t("admin.locations.active")}
                          </Badge>
                        ) : (
                          <Badge variant="secondary">
                            {t("admin.locations.inactive")}
                          </Badge>
                        )}
                        {!g.is_visible && (
                          <Badge
                            variant="secondary"
                            className="bg-amber-500/10 text-amber-600"
                          >
                            {t("admin.maps.hidden")}
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {t("admin.games.counts", {
                        servers: g.server_count ?? 0,
                        tariffs: g.tariff_count ?? 0,
                      })}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                          <Link
                            href={`/admin/games/${g.id}/edit`}
                            title={t("common.edit")}
                          >
                            <Pencil className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          disabled={toggleMut.isPending}
                          onClick={() => toggleMut.mutate(g.id)}
                          title={
                            g.is_active
                              ? t("common.disable")
                              : t("common.enable")
                          }
                        >
                          {g.is_active ? (
                            <Pause className="h-4 w-4 text-amber-500" />
                          ) : (
                            <Play className="h-4 w-4 text-emerald-500" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-rose-500"
                          disabled={deleteMut.isPending}
                          onClick={() => onDelete(g.id, g.name)}
                          title={t("common.delete")}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </PageShell>
  );
}
