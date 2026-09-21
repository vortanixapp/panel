"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Btn, EmptyState, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import {
  applyAgentCleanup,
  previewAgentCleanup,
  type CleanupItem,
  type CleanupResult,
} from "@/lib/api";
import { formatBytes } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

const ORDER = [
  "dangling_images",
  "agent_images",
  "game_images",
  "helper_containers",
  "abandoned_containers",
  "temp_files",
  "build_cache",
  "foreign_images",
];

export function CleanupDialog({
  id,
  open,
  onOpenChange,
}: {
  id: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const [items, setItems] = useState<CleanupItem[] | null>(null);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [result, setResult] = useState<CleanupResult | null>(null);

  const preview = useNodeTask<{ items: CleanupItem[] }>(id, {
    silent: true,
    onDone: (task) => {
      const list = task.result?.items ?? [];
      setItems(list);
      setPicked(new Set(list.filter((i) => i.selected).map((i) => i.id)));
    },
  });
  const apply = useNodeTask<{ result: CleanupResult }>(id, {
    invalidate: [queryKeys.agentDisk(id), queryKeys.agentContainers(id), queryKeys.agents],
    successText: (task) =>
      t("admin.agents.cleanup.freed", { value: formatBytes(task.result?.result.freed_bytes ?? 0) }),
    failureText: t("admin.agents.cleanup.failed"),
    onDone: (task) => setResult(task.result?.result ?? null),
  });
  const { run: runPreview } = preview;

  useEffect(() => {
    if (!open) return;
    setItems(null);
    setResult(null);
    void runPreview(() => previewAgentCleanup(id));
  }, [open, id, runPreview]);

  const groups = useMemo(() => {
    const map = new Map<string, CleanupItem[]>();
    for (const it of items ?? []) {
      const list = map.get(it.category) ?? [];
      list.push(it);
      map.set(it.category, list);
    }
    return ORDER.filter((c) => map.has(c)).map((c) => ({ category: c, items: map.get(c)! }));
  }, [items]);

  const total = (items ?? []).filter((i) => picked.has(i.id)).reduce((s, i) => s + i.size_bytes, 0);

  function toggle(idList: string[], on: boolean) {
    setPicked((prev) => {
      const next = new Set(prev);
      for (const x of idList) {
        if (on) next.add(x);
        else next.delete(x);
      }
      return next;
    });
  }

  const failedPreview = preview.task?.status === "failed" || preview.task?.status === "expired";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{t("admin.agents.cleanup.title")}</DialogTitle>
          <DialogDescription asChild>
            <div className="text-[13px] leading-[1.55] text-[var(--vx-muted)]">{t("admin.agents.cleanup.body")}</div>
          </DialogDescription>
        </DialogHeader>

        {result ? (
          <div className="grid gap-3 text-[12.5px]">
            <div className="text-[15px] font-medium">
              {t("admin.agents.cleanup.freed", { value: formatBytes(result.freed_bytes) })}
            </div>
            <div className={VX_MUTED}>{t("admin.agents.cleanup.removed_count", { count: result.removed.length })}</div>
            {result.skipped.length > 0 && (
              <div className="grid gap-1">
                <div className={VX_FAINT}>{t("admin.agents.cleanup.skipped")}</div>
                {result.skipped.map((s) => (
                  <div key={s.id} className="flex justify-between gap-3 text-[12px]">
                    <span className="truncate font-mono">{s.name || s.id}</span>
                    <span className="text-[var(--vx-warn)]">{s.reason}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : failedPreview ? (
          <div className="text-[12.5px] text-[var(--vx-danger)]">
            {preview.task?.error || t("admin.agents.cleanup.preview_failed")}
          </div>
        ) : items == null ? (
          <div className={cn("py-8 text-center text-[12.5px]", VX_MUTED)}>{t("admin.agents.cleanup.scanning")}</div>
        ) : groups.length === 0 ? (
          <EmptyState>{t("admin.agents.cleanup.nothing")}</EmptyState>
        ) : (
          <div className="grid max-h-[50vh] gap-3 overflow-auto pr-1">
            {groups.map((g) => {
              const ids = g.items.map((i) => i.id);
              const allOn = ids.every((x) => picked.has(x));
              const size = g.items.reduce((s, i) => s + i.size_bytes, 0);
              return (
                <div key={g.category} className="rounded-[10px] border border-[var(--vx-border)]">
                  <label className="flex cursor-pointer items-center justify-between gap-3 border-b border-[var(--vx-border)] px-3 py-2">
                    <span className="inline-flex items-center gap-2 text-[12.5px] font-medium">
                      <input type="checkbox" checked={allOn} onChange={(e) => toggle(ids, e.target.checked)} />
                      {t(`admin.agents.cleanup.cat.${g.category}`)}
                      <span className={cn("font-mono text-[11px]", VX_FAINT)}>{g.items.length}</span>
                    </span>
                    <span className="font-mono text-[12px]">{formatBytes(size)}</span>
                  </label>
                  {g.category === "foreign_images" && (
                    <div className="px-3 pt-2 text-[11.5px] text-[var(--vx-warn)]">{t("admin.agents.cleanup.foreign_hint")}</div>
                  )}
                  <div className="px-3 py-1.5">
                    {g.items.map((it) => (
                      <label key={it.id} className="flex cursor-pointer items-center justify-between gap-3 py-1 text-[12px]">
                        <span className="inline-flex min-w-0 items-center gap-2">
                          <input type="checkbox" checked={picked.has(it.id)} onChange={(e) => toggle([it.id], e.target.checked)} />
                          <span className="truncate font-mono">{it.name}</span>
                          {it.running && <span className="text-[var(--vx-warn)]">{t("admin.agents.cleanup.running")}</span>}
                        </span>
                        <span className={cn("shrink-0 font-mono", VX_FAINT)}>{formatBytes(it.size_bytes)}</span>
                      </label>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <DialogFooter className="gap-2">
          <Btn tone="ghost" onClick={() => onOpenChange(false)}>
            {result ? t("common.close") : t("common.cancel")}
          </Btn>
          {!result && (
            <Btn
              tone="danger"
              disabled={apply.busy || picked.size === 0 || items == null}
              onClick={() => void apply.run(() => applyAgentCleanup(id, [...picked]))}
            >
              {apply.busy
                ? t("admin.agents.cleanup.applying")
                : t("admin.agents.cleanup.apply", { value: formatBytes(total) })}
            </Btn>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
