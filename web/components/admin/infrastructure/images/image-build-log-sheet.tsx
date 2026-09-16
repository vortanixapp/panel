"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, Hammer, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { fetchAdminImageLog, type AdminImagesData, type AdminImageState } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import {
  ImageStatusBadge,
  runtimeLabel,
  stateDetail,
  stateTone,
} from "@/components/admin/infrastructure/images/image-state";
import { useImageBuild } from "@/components/admin/infrastructure/images/build-images-dialog";

type Row = { key: string; name: string; state: AdminImageState };

const ORDER = { building: 0, queued: 1, failed: 2, outdated: 3, ready: 4, missing: 5 } as const;

export function ImageBuildLogSheet({
  data,
  nodeId,
  onOpenChange,
}: {
  data: AdminImagesData;
  nodeId: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const node = data.nodes.find((n) => n.id === nodeId);
  const build = useImageBuild();

  const log = useQuery({
    queryKey: queryKeys.adminImageLog(nodeId ?? ""),
    queryFn: () => fetchAdminImageLog(nodeId ?? ""),
    enabled: Boolean(nodeId),
    refetchInterval: (query) => (node?.building || query.state.data?.completed === false ? 2000 : false),
  });

  const rows = useMemo<Row[]>(() => {
    if (!nodeId) return [];
    const out: Row[] = [];
    for (const rt of data.runtimes) {
      const state = rt.states[nodeId];
      if (state) out.push({ key: rt.key, name: t("admin.images.runtime_name", { name: runtimeLabel(rt.kind) }), state });
    }
    for (const img of data.images) {
      const state = img.states[nodeId];
      if (state) out.push({ key: img.game, name: img.name, state });
    }
    return out.sort((a, b) => ORDER[stateTone(a.state)] - ORDER[stateTone(b.state)]);
  }, [data, nodeId, t]);

  const failedKeys = rows.filter((row) => row.state.status === "failed").map((row) => row.key);

  const logRef = useRef<HTMLPreElement>(null);
  const text = log.data?.log ?? "";
  useEffect(() => {
    const el = logRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [text]);

  return (
    <Sheet open={nodeId !== null} onOpenChange={onOpenChange}>
      <SheetContent className="w-full gap-0 p-0 sm:max-w-xl">
        <SheetHeader className="border-b px-5 py-4">
          <SheetTitle className="flex items-center gap-2 pe-8">
            {t("admin.images.log.title", { name: node?.name ?? "" })}
            {node?.building && <Loader2 className="size-4 animate-spin text-[var(--vx-info)]" />}
          </SheetTitle>
          <SheetDescription>
            {node?.building ? t("admin.images.log.building") : t("admin.images.log.idle")}
          </SheetDescription>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-5 py-4">
          <section className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-[13px] font-semibold">{t("admin.images.log.results")}</h3>
              {failedKeys.length > 0 && node?.can_build && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={build.isPending}
                  onClick={() => build.mutate({ node_ids: [node.id], images: failedKeys })}
                >
                  <Hammer className="size-3.5" />
                  {t("admin.images.log.retry_failed", { count: failedKeys.length })}
                </Button>
              )}
            </div>
            {rows.length === 0 ? (
              <p className="rounded-lg border border-dashed px-3.5 py-4 text-center text-[12.5px] text-muted-foreground">
                {t("admin.images.log.no_results")}
              </p>
            ) : (
              <div className="flex flex-col divide-y overflow-hidden rounded-lg border">
                {rows.map((row) => (
                  <div key={row.key} className="flex flex-col gap-1.5 px-3.5 py-2.5">
                    <div className="flex items-center gap-2">
                      <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{row.name}</span>
                      <ImageStatusBadge state={row.state} />
                    </div>
                    <span className="text-[12px] text-muted-foreground">{stateDetail(row.state)}</span>
                    {row.state.status === "failed" && row.state.error && (
                      <pre className="max-h-32 overflow-auto rounded-md bg-[var(--vx-danger-tint)] px-2.5 py-2 font-mono text-[11px] leading-[1.5] whitespace-pre-wrap text-[var(--vx-danger)]">
                        {row.state.error}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className="flex min-h-0 flex-1 flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-[13px] font-semibold">{t("admin.images.log.output")}</h3>
              {node && (
                <Link
                  href={`/admin/locations/${node.id}/setup`}
                  className="inline-flex items-center gap-1 text-[12.5px] font-medium text-primary transition-opacity hover:opacity-80"
                >
                  {t("admin.images.log.setup_link")}
                  <ExternalLink className="size-3" />
                </Link>
              )}
            </div>
            {log.data && log.data.component && log.data.component !== "images" && (
              <p className="text-[12px] text-muted-foreground">
                {t("admin.images.log.other_step", { step: log.data.component })}
              </p>
            )}
            <pre
              ref={logRef}
              className="min-h-[220px] flex-1 overflow-auto rounded-lg border bg-muted/30 px-3 py-2.5 font-mono text-[11.5px] leading-[1.55] whitespace-pre-wrap text-foreground/85"
            >
              {log.isLoading ? t("admin.images.log.loading") : text || t("admin.images.log.empty")}
            </pre>
          </section>
        </div>
      </SheetContent>
    </Sheet>
  );
}
