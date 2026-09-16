"use client";

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Hammer, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { buildAdminImages, type AdminImagesData } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import {
  imagesErrorText,
  nodeReasonText,
  runtimeLabel,
} from "@/components/admin/infrastructure/images/image-state";

export type BuildPreset = {
  images: string[] | null;
  nodeIds: string[];
};

type Scope = "default" | "runtimes";

export function useImageBuild() {
  const t = useT();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: buildAdminImages,
    onSuccess: (res) => {
      toast.success(
        res.queued.length === 1
          ? t("admin.images.queued_one", { name: res.queued[0].name, count: res.images })
          : t("admin.images.queued_many", { nodes: res.queued.length, count: res.images })
      );
      if (res.skipped.length > 0) {
        toast.warning(
          t("admin.images.skipped", {
            names: res.skipped.map((s) => s.name || s.node_id).join(", "),
          })
        );
      }
      void qc.invalidateQueries({ queryKey: queryKeys.adminImages });
    },
    onError: (error) => toast.error(imagesErrorText(error)),
  });
}

export function BuildImagesDialog({
  data,
  preset,
  onOpenChange,
}: {
  data: AdminImagesData;
  preset: BuildPreset | null;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={preset !== null} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-32px)] overflow-y-auto sm:max-w-[520px]">
        {preset && (
          <BuildImagesForm
            key={`${(preset.images ?? []).join(",")}|${preset.nodeIds.join(",")}`}
            data={data}
            preset={preset}
            onDone={() => onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function BuildImagesForm({
  data,
  preset,
  onDone,
}: {
  data: AdminImagesData;
  preset: BuildPreset;
  onDone: () => void;
}) {
  const t = useT();
  const build = useImageBuild();
  const buildable = useMemo(() => data.nodes.filter((node) => node.can_build), [data.nodes]);
  const [scope, setScope] = useState<Scope>("default");
  const [nodes, setNodes] = useState<Set<string>>(() => {
    const wanted = preset.nodeIds.length > 0 ? preset.nodeIds : buildable.map((node) => node.id);
    return new Set(wanted.filter((id) => buildable.some((node) => node.id === id)));
  });

  const marked = data.images.filter((img) => img.build_image).length;
  const names = useMemo(() => {
    if (!preset.images) return [];
    return preset.images.map((key) => {
      const runtime = data.runtimes.find((rt) => rt.key === key);
      if (runtime) return t("admin.images.runtime_name", { name: runtimeLabel(runtime.kind) });
      return data.images.find((img) => img.game === key)?.name ?? key;
    });
  }, [preset.images, data, t]);

  const toggleNode = (id: string, on: boolean) => {
    setNodes((prev) => {
      const next = new Set(prev);
      if (on) next.add(id);
      else next.delete(id);
      return next;
    });
  };
  const allSelected = buildable.length > 0 && buildable.every((node) => nodes.has(node.id));

  const submit = () => {
    const images = preset.images ?? (scope === "runtimes" ? data.runtimes.map((rt) => rt.key) : undefined);
    build.mutate(
      { node_ids: [...nodes], images },
      { onSuccess: onDone }
    );
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("admin.images.dialog.title")}</DialogTitle>
        <DialogDescription>{t("admin.images.dialog.description")}</DialogDescription>
      </DialogHeader>

      <div className="flex flex-col gap-5">
        <section className="flex flex-col gap-2">
          <h3 className="text-[13px] font-semibold">{t("admin.images.dialog.what")}</h3>
          {preset.images ? (
            <div className="flex flex-wrap gap-1.5">
              {names.map((name) => (
                <span key={name} className="rounded-full border bg-muted/40 px-2.5 py-1 text-[12.5px]">
                  {name}
                </span>
              ))}
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {(
                [
                  {
                    id: "default",
                    title: t("admin.images.dialog.default_title"),
                    text: t("admin.images.dialog.default_text", {
                      runtimes: data.runtimes.length,
                      games: marked,
                    }),
                  },
                  {
                    id: "runtimes",
                    title: t("admin.images.dialog.runtimes_title"),
                    text: t("admin.images.dialog.runtimes_text", { runtimes: data.runtimes.length }),
                  },
                ] as const
              ).map((option) => (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => setScope(option.id)}
                  aria-pressed={scope === option.id}
                  className={cn(
                    "flex items-start gap-3 rounded-lg border px-3.5 py-3 text-left transition-colors",
                    scope === option.id ? "border-primary bg-primary/5" : "hover:bg-accent/60"
                  )}
                >
                  <span
                    className={cn(
                      "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border",
                      scope === option.id ? "border-primary" : "border-input"
                    )}
                  >
                    {scope === option.id && <span className="size-2 rounded-full bg-primary" />}
                  </span>
                  <span className="min-w-0">
                    <span className="block text-[13.5px] font-medium">{option.title}</span>
                    <span className="block text-[12.5px] text-muted-foreground">{option.text}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-2">
            <h3 className="text-[13px] font-semibold">{t("admin.images.dialog.where")}</h3>
            {buildable.length > 1 && (
              <button
                type="button"
                onClick={() =>
                  setNodes(allSelected ? new Set() : new Set(buildable.map((node) => node.id)))
                }
                className="text-[12.5px] font-medium text-primary transition-opacity hover:opacity-80"
              >
                {allSelected ? t("admin.images.dialog.select_none") : t("admin.images.dialog.select_all")}
              </button>
            )}
          </div>
          {data.nodes.length === 0 ? (
            <p className="rounded-lg border border-dashed px-3.5 py-4 text-center text-[12.5px] text-muted-foreground">
              {t("admin.images.no_nodes")}
            </p>
          ) : (
            <div className="flex flex-col divide-y overflow-hidden rounded-lg border">
              {data.nodes.map((node) => (
                <label
                  key={node.id}
                  className={cn(
                    "flex items-center gap-3 px-3.5 py-2.5",
                    node.can_build ? "cursor-pointer hover:bg-accent/40" : "opacity-60"
                  )}
                >
                  <Checkbox
                    checked={nodes.has(node.id)}
                    disabled={!node.can_build}
                    onCheckedChange={(on) => toggleNode(node.id, on === true)}
                  />
                  <span
                    className={cn(
                      "size-2 shrink-0 rounded-full",
                      node.online ? "bg-[var(--vx-ok)]" : "bg-muted-foreground/40"
                    )}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[13.5px] font-medium">{node.name}</span>
                    <span className="block truncate text-[12px] text-muted-foreground">
                      {node.can_build
                        ? [node.code, node.country].filter(Boolean).join(" · ") ||
                          t("admin.images.dialog.ready_to_build")
                        : nodeReasonText(node)}
                    </span>
                  </span>
                  {node.building && (
                    <span className="inline-flex items-center gap-1 text-[11.5px] text-[var(--vx-info)]">
                      <Loader2 className="size-3 animate-spin" />
                      {t("admin.images.node_building")}
                    </span>
                  )}
                </label>
              ))}
            </div>
          )}
        </section>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onDone}>
          {t("common.cancel")}
        </Button>
        <Button onClick={submit} disabled={nodes.size === 0 || build.isPending}>
          {build.isPending ? <Loader2 className="size-4 animate-spin" /> : <Hammer className="size-4" />}
          {t("admin.images.dialog.submit", { count: nodes.size })}
        </Button>
      </DialogFooter>
    </>
  );
}
