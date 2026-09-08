"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Hammer, Search } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  buildAdminImages,
  fetchAdminImages,
  setAdminImageBuild,
  type AdminImageItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

type Filter = "build" | "catalog" | "all";

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const FILTERS: { id: Filter; labelKey: string }[] = [
  { id: "build", labelKey: "admin.images.filter.marked" },
  { id: "catalog", labelKey: "admin.images.filter.catalog" },
  { id: "all", labelKey: "admin.images.filter.all" },
];

function BuildState({ img }: { img: AdminImageItem }) {
  const t = useT();
  if (!img.build_image) {
    return (
      <span className="text-[13px] text-muted-foreground">
        {t("admin.images.state.off")}
      </span>
    );
  }
  if (img.building > 0) {
    return (
      <span className="text-[13px] text-amber-500">
        {t("admin.images.state.building", { count: img.building })}
      </span>
    );
  }
  if (img.build_failed > 0) {
    return (
      <span className="text-[13px] text-destructive">
        {t("admin.images.state.failed", {
          failed: img.build_failed,
          total: img.build_nodes,
        })}
      </span>
    );
  }
  if (img.build_ready > 0) {
    return (
      <span className="text-[13px] text-emerald-500">
        {t("admin.images.state.ready", { count: img.build_ready })}
        {img.build_nodes > img.build_ready
          ? t("admin.images.state.of_total", { total: img.build_nodes })
          : ""}
      </span>
    );
  }
  return (
    <span className="text-[13px] text-muted-foreground">
      {t("admin.images.state.waiting")}
    </span>
  );
}

export function ImagesPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("catalog");
  const [search, setSearch] = useState("");

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.adminImages,
    queryFn: async () => (await fetchAdminImages()).images,
  });

  const images = useMemo(() => data ?? [], [data]);

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return images.filter((img) => {
      if (filter === "build" && !img.build_image) return false;
      if (filter === "catalog" && !img.in_catalog) return false;
      if (!q) return true;
      return (
        img.name.toLowerCase().includes(q) || img.game.toLowerCase().includes(q)
      );
    });
  }, [images, filter, search]);

  const marked = useMemo(() => images.filter((i) => i.build_image).length, [images]);

  const toggle = useMutation({
    mutationFn: (p: { game_slug: string; build: boolean }) => setAdminImageBuild(p),
    onSuccess: () => void qc.invalidateQueries({ queryKey: queryKeys.adminImages }),
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const build = useMutation({
    mutationFn: () => buildAdminImages(),
    onSuccess: (res) => {
      toast.success(
        res.nodes === 1
          ? t("admin.images.queued_one")
          : t("admin.images.queued_many", { count: res.nodes })
      );
      void qc.invalidateQueries({ queryKey: queryKeys.adminImages });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.images.build_failed")),
  });

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.images.title")}
            </h1>
            <p className="max-w-[620px] text-sm text-muted-foreground">
              {t("admin.images.subtitle")}
            </p>
          </div>
          <Button
            className="h-[38px] text-[13px]"
            disabled={build.isPending || marked === 0}
            onClick={() => build.mutate()}
          >
            <Hammer className="size-4" />
            {build.isPending
              ? t("admin.images.queueing")
              : t("admin.images.build")}
          </Button>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1 rounded-[10px] border border-border bg-card p-1">
            {FILTERS.map((f) => (
              <button
                key={f.id}
                type="button"
                onClick={() => setFilter(f.id)}
                className={cn(
                  "h-8 rounded-[7px] px-3 text-[13px] transition-colors",
                  filter === f.id
                    ? "bg-muted font-medium text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {t(f.labelKey)}
              </button>
            ))}
          </div>

          <div className="relative min-w-[220px] flex-1 sm:max-w-[320px]">
            <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("admin.images.search_placeholder")}
              className="h-[38px] pl-9 text-[13px]"
            />
          </div>

          <span className="text-[13px] text-muted-foreground">
            {t("admin.images.marked", {
              marked,
              total: images.length,
            })}
          </span>
        </div>

        {isLoading ? (
          <div className="space-y-2">
            {[0, 1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-[62px] w-full rounded-[12px]" />
            ))}
          </div>
        ) : isError ? (
          <div className="rounded-[12px] border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            {t("admin.images.load_failed")}
          </div>
        ) : visible.length === 0 ? (
          <div className="rounded-[12px] border border-border bg-card px-4 py-10 text-center text-sm text-muted-foreground">
            {filter === "build"
              ? t("admin.images.none_marked")
              : t("common.not_found")}
          </div>
        ) : (
          <div className="overflow-hidden rounded-[12px] border border-border">
            <div className="hidden grid-cols-[1.6fr_1.4fr_1fr_auto] gap-4 bg-muted/40 px-4 py-2.5 text-[12px] text-muted-foreground sm:grid">
              <div>{t("common.game")}</div>
              <div>{t("admin.images.col_image")}</div>
              <div>{t("admin.images.col_state")}</div>
              <div className="text-right">{t("admin.images.col_build")}</div>
            </div>
            {visible.map((img) => (
              <div
                key={img.game}
                className="grid grid-cols-1 items-center gap-2 border-t border-border px-4 py-3 first:border-t-0 sm:grid-cols-[1.6fr_1.4fr_1fr_auto] sm:gap-4"
              >
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium">{img.name}</div>
                  <div className="font-mono text-[11.5px] text-muted-foreground">
                    {img.game}
                    {!img.in_catalog && t("admin.images.not_connected")}
                  </div>
                </div>
                <div className="min-w-0 font-mono text-[12px] break-all text-muted-foreground">
                  {img.repository}:{img.tag}
                </div>
                <div>
                  <BuildState img={img} />
                </div>
                <div className="flex justify-start sm:justify-end">
                  <Switch
                    checked={img.build_image}
                    disabled={!img.in_catalog || toggle.isPending}
                    onCheckedChange={(v) =>
                      toggle.mutate({ game_slug: img.game, build: v })
                    }
                  />
                </div>
              </div>
            ))}
          </div>
        )}

        <p className="text-[12px] text-muted-foreground">
          {t("admin.images.footer_hint")}
        </p>
      </div>
    </PageShell>
  );
}
