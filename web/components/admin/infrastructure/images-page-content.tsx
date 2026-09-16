"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  AlertTriangle,
  Hammer,
  Layers,
  Loader2,
  ScrollText,
  Search,
  Server,
  X,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  fetchAdminImages,
  setAdminImageBuild,
  type AdminImageItem,
  type AdminImageNode,
  type AdminImagesData,
  type AdminRuntimeImage,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import {
  BuildImagesDialog,
  useImageBuild,
  type BuildPreset,
} from "@/components/admin/infrastructure/images/build-images-dialog";
import { ImageBuildLogSheet } from "@/components/admin/infrastructure/images/image-build-log-sheet";
import {
  ImageStatusBadge,
  RUNTIME_META,
  SummaryBar,
  hasActivity,
  nodeReasonText,
  runtimeLabel,
  stateDetail,
  stateTone,
  summarize,
  summaryText,
} from "@/components/admin/infrastructure/images/image-state";

type Filter = "catalog" | "auto" | "problems" | "all";

const FILTERS: Filter[] = ["catalog", "auto", "problems", "all"];

export function ImagesPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.adminImages,
    queryFn: fetchAdminImages,
    refetchInterval: (query) => (hasActivity(query.state.data) ? 3000 : false),
  });

  const scopeParam = searchParams.get("node") ?? "";
  const scopeNode = data?.nodes.find((node) => node.id === scopeParam);

  const setScope = useCallback(
    (id: string) => {
      const params = new URLSearchParams(searchParams.toString());
      if (id) params.set("node", id);
      else params.delete("node");
      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  const [filter, setFilter] = useState<Filter>("catalog");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [preset, setPreset] = useState<BuildPreset | null>(null);
  const [logNode, setLogNode] = useState<string | null>(null);

  const build = useImageBuild();

  const toggleAuto = useMutation({
    mutationFn: setAdminImageBuild,
    onMutate: async ({ game_slug, build: on }) => {
      await qc.cancelQueries({ queryKey: queryKeys.adminImages });
      const previous = qc.getQueryData<AdminImagesData>(queryKeys.adminImages);
      if (previous) {
        qc.setQueryData<AdminImagesData>(queryKeys.adminImages, {
          ...previous,
          images: previous.images.map((img) => (img.game === game_slug ? { ...img, build_image: on } : img)),
        });
      }
      return { previous };
    },
    onError: (error, _vars, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.adminImages, context.previous);
      toast.error(error instanceof Error && error.message ? error.message : t("common.save_failed"));
    },
    onSettled: () => void qc.invalidateQueries({ queryKey: queryKeys.adminImages }),
  });

  const nodes = useMemo(() => data?.nodes ?? [], [data]);
  const images = useMemo(() => data?.images ?? [], [data]);

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return images.filter((img) => {
      if (filter === "catalog" && !img.in_catalog) return false;
      if (filter === "auto" && !img.build_image) return false;
      if (filter === "problems") {
        const tones = scopeNode
          ? [stateTone(img.states[scopeNode.id])]
          : Object.values(img.states).map((state) => stateTone(state));
        if (!tones.some((tone) => tone === "failed" || tone === "outdated")) return false;
      }
      if (!q) return true;
      return img.name.toLowerCase().includes(q) || img.game.toLowerCase().includes(q);
    });
  }, [images, filter, search, scopeNode]);

  const requestBuild = (keys: string[] | null) => {
    if (scopeNode && keys) {
      build.mutate({ node_ids: [scopeNode.id], images: keys });
      return;
    }
    setPreset({ images: keys, nodeIds: scopeNode ? [scopeNode.id] : [] });
  };

  const allVisibleSelected = visible.length > 0 && visible.every((img) => selected.has(img.game));
  const toggleSelected = (key: string, on: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (on) next.add(key);
      else next.delete(key);
      return next;
    });
  };

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-5">
          <Skeleton className="h-14 w-80 rounded-xl" />
          <div className="flex gap-2">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-[88px] w-[220px] rounded-xl" />
            ))}
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {[0, 1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-[210px] rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-[360px] rounded-xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !data) {
    return (
      <PageShell variant="admin">
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed px-5 py-16 text-center">
          <AlertTriangle className="size-5 text-destructive" />
          <div className="text-sm font-medium">{t("admin.images.load_failed")}</div>
          <Button variant="outline" size="sm" onClick={() => void refetch()}>
            {t("common.retry")}
          </Button>
        </div>
      </PageShell>
    );
  }

  const marked = images.filter((img) => img.build_image).length;
  const buildableNodes = nodes.filter((node) => node.can_build).length;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6">
        <header className="flex flex-wrap items-end justify-between gap-4">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">{t("admin.images.title")}</h1>
            <p className="max-w-[700px] text-sm text-muted-foreground">{t("admin.images.subtitle")}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {data.ref && (
              <span className="rounded-full border px-2.5 py-1 font-mono text-[11.5px] text-muted-foreground">
                {t("admin.images.recipes", { ref: data.ref })}
              </span>
            )}
            <Button
              className="h-9"
              disabled={buildableNodes === 0 || (scopeNode !== undefined && !scopeNode.can_build)}
              onClick={() => setPreset({ images: null, nodeIds: scopeNode ? [scopeNode.id] : [] })}
            >
              <Hammer className="size-4" />
              {scopeNode ? t("admin.images.build_on", { name: scopeNode.name }) : t("admin.images.build")}
            </Button>
          </div>
        </header>

        {nodes.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed bg-card px-5 py-10 text-center">
            <Server className="size-5 text-muted-foreground" />
            <div className="text-sm font-medium">{t("admin.images.no_nodes")}</div>
            <Button variant="outline" size="sm" asChild>
              <Link href="/admin/locations/create">{t("admin.images.add_location")}</Link>
            </Button>
          </div>
        ) : (
          <div className="no-scrollbar -mx-1 flex gap-2 overflow-x-auto px-1 pb-1">
            <ScopeTile active={!scopeNode} onClick={() => setScope("")}>
              <div className="flex items-center gap-2">
                <Layers className="size-4 text-muted-foreground" />
                <span className="text-[13.5px] font-semibold">{t("admin.images.scope_all")}</span>
              </div>
              <span className="text-[12px] text-muted-foreground">
                {t("admin.images.scope_all_hint", { count: nodes.length, ready: buildableNodes })}
              </span>
            </ScopeTile>
            {nodes.map((node) => (
              <NodeTile
                key={node.id}
                node={node}
                data={data}
                active={scopeNode?.id === node.id}
                onSelect={() => setScope(node.id)}
                onLog={() => setLogNode(node.id)}
              />
            ))}
          </div>
        )}

        {scopeNode && !scopeNode.can_build && (
          <div className="flex flex-wrap items-center gap-3 rounded-xl border border-[var(--vx-warn)]/30 bg-[var(--vx-warn-tint)] px-4 py-3 text-[13px]">
            <AlertTriangle className="size-4 text-[var(--vx-warn)]" />
            <span className="flex-1">{t("admin.images.node_blocked", { reason: nodeReasonText(scopeNode) })}</span>
            <Button variant="outline" size="sm" asChild>
              <Link href={`/admin/locations/${scopeNode.id}/setup`}>{t("admin.images.open_setup")}</Link>
            </Button>
          </div>
        )}

        <section className="space-y-3">
          <SectionHeading title={t("admin.images.runtimes_title")} hint={t("admin.images.runtimes_hint")} />
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {data.runtimes.map((runtime) => (
              <RuntimeCard
                key={runtime.key}
                runtime={runtime}
                nodes={nodes}
                scopeNode={scopeNode}
                pending={build.isPending}
                onBuild={() => requestBuild([runtime.key])}
                onLog={() => scopeNode && setLogNode(scopeNode.id)}
              />
            ))}
          </div>
        </section>

        <section className="space-y-3">
          <SectionHeading title={t("admin.images.games_title")} hint={t("admin.images.games_hint")} />

          <div className="flex flex-wrap items-center gap-2">
            <div className="flex gap-1 overflow-x-auto rounded-[10px] border bg-card p-1">
              {FILTERS.map((id) => (
                <button
                  key={id}
                  type="button"
                  aria-pressed={filter === id}
                  onClick={() => setFilter(id)}
                  className={cn(
                    "h-8 shrink-0 rounded-[7px] px-3 text-[13px] whitespace-nowrap transition-colors",
                    filter === id ? "bg-muted font-medium text-foreground" : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(`admin.images.filter.${id}`)}
                </button>
              ))}
            </div>
            <div className="relative min-w-[200px] flex-1 sm:max-w-[300px]">
              <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t("admin.images.search_placeholder")}
                className="h-[38px] pl-9 text-[13px]"
              />
            </div>
            <span className="text-[12.5px] text-muted-foreground">
              {t("admin.images.auto_count", { marked, total: images.length })}
            </span>
          </div>

          {visible.length === 0 ? (
            <div className="rounded-xl border border-dashed bg-card px-4 py-12 text-center text-sm text-muted-foreground">
              {filter === "auto"
                ? t("admin.images.none_auto")
                : filter === "problems"
                  ? t("admin.images.none_problems")
                  : t("common.not_found")}
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl border bg-card">
              <div className="hidden grid-cols-[20px_minmax(0,1.5fr)_84px_minmax(0,1.2fr)_minmax(0,1.3fr)_92px_112px] items-center gap-4 border-b bg-muted/40 px-4 py-2.5 text-[12px] text-muted-foreground lg:grid">
                <Checkbox
                  checked={allVisibleSelected}
                  aria-label={t("admin.images.select_all")}
                  onCheckedChange={(on) =>
                    setSelected((prev) => {
                      const next = new Set(prev);
                      for (const img of visible) {
                        if (on === true) next.add(img.game);
                        else next.delete(img.game);
                      }
                      return next;
                    })
                  }
                />
                <span>{t("common.game")}</span>
                <span>{t("admin.images.col_runtime")}</span>
                <span>{t("admin.images.col_image")}</span>
                <span>{scopeNode ? t("admin.images.col_state_on", { name: scopeNode.name }) : t("admin.images.col_state")}</span>
                <span>{t("admin.images.col_auto")}</span>
                <span />
              </div>
              <div className="divide-y">
                {visible.map((img) => (
                  <GameRow
                    key={img.game}
                    img={img}
                    nodes={nodes}
                    scopeNode={scopeNode}
                    selected={selected.has(img.game)}
                    pending={build.isPending}
                    onSelect={(on) => toggleSelected(img.game, on)}
                    onAuto={(on) => toggleAuto.mutate({ game_slug: img.game, build: on })}
                    onBuild={() => requestBuild([img.game])}
                    onLog={() => scopeNode && setLogNode(scopeNode.id)}
                  />
                ))}
              </div>
            </div>
          )}
        </section>

        <p className="max-w-[900px] text-[12px] leading-[1.55] text-muted-foreground">{t("admin.images.footer_hint")}</p>
      </div>

      {selected.size > 0 && (
        <div className="pointer-events-none sticky bottom-4 z-20 mt-4 flex justify-center">
          <div className="pointer-events-auto flex flex-wrap items-center gap-2 rounded-xl border bg-popover px-3 py-2 shadow-lg">
            <span className="px-1 text-[13px]">{t("admin.images.selected", { count: selected.size })}</span>
            <Button
              size="sm"
              disabled={build.isPending || (scopeNode !== undefined && !scopeNode.can_build)}
              onClick={() => {
                requestBuild([...selected]);
                setSelected(new Set());
              }}
            >
              <Hammer className="size-3.5" />
              {scopeNode ? t("admin.images.build_selected_on", { name: scopeNode.name }) : t("admin.images.build_selected")}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setSelected(new Set())}>
              <X className="size-3.5" />
              {t("admin.images.clear_selection")}
            </Button>
          </div>
        </div>
      )}

      <BuildImagesDialog data={data} preset={preset} onOpenChange={(open) => !open && setPreset(null)} />
      <ImageBuildLogSheet data={data} nodeId={logNode} onOpenChange={(open) => !open && setLogNode(null)} />
    </PageShell>
  );
}

function SectionHeading({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="space-y-1">
      <h2 className="text-[16px] font-semibold tracking-tight">{title}</h2>
      <p className="max-w-[760px] text-[13px] text-muted-foreground">{hint}</p>
    </div>
  );
}

function ScopeTile({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "flex min-w-[200px] shrink-0 flex-col items-start gap-1.5 rounded-xl border bg-card px-3.5 py-3 text-left transition-colors",
        active ? "border-primary ring-1 ring-primary" : "hover:border-ring"
      )}
    >
      {children}
    </button>
  );
}

function NodeTile({
  node,
  data,
  active,
  onSelect,
  onLog,
}: {
  node: AdminImageNode;
  data: AdminImagesData;
  active: boolean;
  onSelect: () => void;
  onLog: () => void;
}) {
  const t = useT();
  const runtimesReady = data.runtimes.filter((rt) => {
    const tone = stateTone(rt.states[node.id]);
    return tone === "ready" || tone === "outdated";
  }).length;
  const failed =
    data.runtimes.filter((rt) => rt.states[node.id]?.status === "failed").length +
    data.images.filter((img) => img.states[node.id]?.status === "failed").length;

  return (
    <div
      className={cn(
        "relative flex min-w-[230px] shrink-0 flex-col gap-1.5 rounded-xl border bg-card px-3.5 py-3 transition-colors",
        active ? "border-primary ring-1 ring-primary" : "hover:border-ring"
      )}
    >
      <button type="button" onClick={onSelect} aria-pressed={active} className="absolute inset-0 rounded-xl" aria-label={node.name} />
      <div className="pointer-events-none relative flex items-center gap-2 pe-8">
        <span
          className={cn(
            "size-2 shrink-0 rounded-full",
            node.online ? "bg-[var(--vx-ok)]" : "bg-muted-foreground/40"
          )}
        />
        <span className="truncate text-[13.5px] font-semibold">{node.name}</span>
        {node.building && <Loader2 className="size-3.5 shrink-0 animate-spin text-[var(--vx-info)]" />}
      </div>
      <button
        type="button"
        onClick={onLog}
        title={t("admin.images.log.open")}
        aria-label={t("admin.images.log.open")}
        className="absolute top-2 right-2 flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <ScrollText className="size-3.5" />
      </button>
      <span className="pointer-events-none relative truncate text-[12px] text-muted-foreground">
        {[node.code, node.country].filter(Boolean).join(" · ") || "—"}
      </span>
      <span className="pointer-events-none relative flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px]">
        {node.can_build ? (
          <>
            <span className={runtimesReady === data.runtimes.length ? "text-[var(--vx-ok)]" : "text-muted-foreground"}>
              {t("admin.images.node_runtimes", { ready: runtimesReady, total: data.runtimes.length })}
            </span>
            {failed > 0 && (
              <span className="text-[var(--vx-danger)]">{t("admin.images.node_failed", { count: failed })}</span>
            )}
          </>
        ) : (
          <span className="text-[var(--vx-warn)]">{nodeReasonText(node)}</span>
        )}
      </span>
    </div>
  );
}

function RuntimeCard({
  runtime,
  nodes,
  scopeNode,
  pending,
  onBuild,
  onLog,
}: {
  runtime: AdminRuntimeImage;
  nodes: AdminImageNode[];
  scopeNode?: AdminImageNode;
  pending: boolean;
  onBuild: () => void;
  onLog: () => void;
}) {
  const t = useT();
  const meta = RUNTIME_META[runtime.kind] ?? { icon: "ri-box-3-line", label: runtime.kind };
  const state = scopeNode ? runtime.states[scopeNode.id] : undefined;
  const summary = summarize(runtime.states, nodes);
  const tone = stateTone(state);
  const busy = tone === "building" || tone === "queued";

  return (
    <div data-spotlight className="relative isolate flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-[20px] text-foreground/80">
          <i className={meta.icon} />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
            <span className="text-[15px] font-semibold">{meta.label}</span>
            {scopeNode && <ImageStatusBadge state={state} />}
          </div>
          <p className="mt-0.5 text-[12.5px] leading-[1.45] text-muted-foreground">
            {t(`admin.images.runtime.${runtime.kind}`)}
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-1">
        <span className="truncate font-mono text-[11.5px] text-muted-foreground" title={runtime.image}>
          {runtime.image}
        </span>
        <span className="text-[12px] text-muted-foreground">
          {t("admin.images.runtime_games", { enabled: runtime.enabled_games, total: runtime.games })}
        </span>
      </div>

      {scopeNode ? (
        <div className="flex flex-col gap-1.5 rounded-lg bg-muted/40 px-3 py-2.5">
          <span className="text-[12.5px]">{stateDetail(state)}</span>
          {state?.status === "failed" && (
            <button
              type="button"
              onClick={onLog}
              className="self-start text-[12px] font-medium text-[var(--vx-danger)] underline-offset-4 hover:underline"
            >
              {t("admin.images.show_error")}
            </button>
          )}
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          <SummaryBar summary={summary} />
          <span className="text-[12px] text-muted-foreground">{summaryText(summary)}</span>
        </div>
      )}

      <Button
        variant="outline"
        size="sm"
        className="mt-auto w-full"
        disabled={pending || busy || (scopeNode !== undefined && !scopeNode.can_build)}
        onClick={onBuild}
      >
        {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Hammer className="size-3.5" />}
        {busy
          ? t("admin.images.status.building")
          : tone === "ready" || tone === "outdated" || tone === "failed"
            ? t("admin.images.rebuild")
            : scopeNode
              ? t("admin.images.build_here")
              : t("admin.images.build_short")}
      </Button>
    </div>
  );
}

function GameRow({
  img,
  nodes,
  scopeNode,
  selected,
  pending,
  onSelect,
  onAuto,
  onBuild,
  onLog,
}: {
  img: AdminImageItem;
  nodes: AdminImageNode[];
  scopeNode?: AdminImageNode;
  selected: boolean;
  pending: boolean;
  onSelect: (on: boolean) => void;
  onAuto: (on: boolean) => void;
  onBuild: () => void;
  onLog: () => void;
}) {
  const t = useT();
  const state = scopeNode ? img.states[scopeNode.id] : undefined;
  const tone = stateTone(state);
  const busy = tone === "building" || tone === "queued";
  const summary = summarize(img.states, nodes);

  return (
    <div
      className={cn(
        "grid grid-cols-[20px_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 px-4 py-3 transition-colors lg:grid-cols-[20px_minmax(0,1.5fr)_84px_minmax(0,1.2fr)_minmax(0,1.3fr)_92px_112px] lg:gap-4",
        selected && "bg-primary/5"
      )}
    >
      <Checkbox checked={selected} aria-label={img.name} onCheckedChange={(on) => onSelect(on === true)} />

      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <span className="truncate text-[13.5px] font-medium">{img.name}</span>
          {!img.in_catalog && (
            <span className="rounded-full bg-muted px-1.5 py-px text-[10.5px] text-muted-foreground">
              {t("admin.images.not_connected")}
            </span>
          )}
        </div>
        <div className="truncate font-mono text-[11.5px] text-muted-foreground">
          {img.game}
          {img.shared_with.length > 0 && (
            <span className="font-sans">{" · "}{t("admin.images.shared_with", { games: img.shared_with.join(", ") })}</span>
          )}
        </div>
      </div>

      <div className="hidden lg:block">
        <span className="rounded-full border px-2 py-0.5 text-[11.5px] text-muted-foreground">
          {runtimeLabel(img.runtime)}
        </span>
      </div>

      <div className="col-span-3 hidden min-w-0 truncate font-mono text-[12px] text-muted-foreground lg:col-span-1 lg:block" title={`${img.repository}:${img.tag}`}>
        {img.repository}:{img.tag}
      </div>

      <div className="col-span-2 col-start-2 min-w-0 lg:col-span-1 lg:col-start-auto">
        {scopeNode ? (
          <div className="flex flex-wrap items-center gap-2">
            <ImageStatusBadge state={state} />
            {state?.status === "failed" ? (
              <button
                type="button"
                onClick={onLog}
                className="text-[12px] font-medium text-[var(--vx-danger)] underline-offset-4 hover:underline"
              >
                {t("admin.images.show_error")}
              </button>
            ) : (
              <span className="truncate text-[12px] text-muted-foreground">{stateDetail(state)}</span>
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            <SummaryBar summary={summary} className="max-w-[220px]" />
            <span className="text-[12px] text-muted-foreground">{summaryText(summary)}</span>
          </div>
        )}
      </div>

      <label className="col-start-2 flex items-center gap-2 text-[12px] text-muted-foreground lg:col-start-auto">
        <Switch
          checked={img.build_image}
          disabled={!img.in_catalog}
          aria-label={t("admin.images.col_auto")}
          onCheckedChange={onAuto}
        />
        <span className="lg:hidden">{t("admin.images.col_auto")}</span>
      </label>

      <div className="col-start-3 row-start-1 flex justify-end lg:col-start-auto lg:row-start-auto">
        <Button
          variant="outline"
          size="sm"
          disabled={pending || busy || (scopeNode !== undefined && !scopeNode.can_build)}
          onClick={onBuild}
        >
          {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Hammer className="size-3.5" />}
          <span className="max-sm:hidden">
            {busy
              ? t("admin.images.status.building")
              : tone === "ready" || tone === "outdated" || tone === "failed"
                ? t("admin.images.rebuild")
                : t("admin.images.build_short")}
          </span>
        </Button>
      </div>
    </div>
  );
}
