"use client";

import {
  CheckCircle2,
  Circle,
  Clock3,
  Loader2,
  RefreshCw,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import type {
  AdminImageNode,
  AdminImageState,
  AdminImageStates,
  AdminImagesData,
} from "@/lib/api";
import { timeAgo } from "@/components/user/account/shared";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export type ImageTone = "ready" | "outdated" | "building" | "queued" | "failed" | "missing";

export const RUNTIME_META: Record<string, { icon: string; label: string }> = {
  steam: { icon: "ri-steam-line", label: "Steam" },
  srcds: { icon: "ri-crosshair-2-line", label: "Source" },
  java: { icon: "ri-cup-line", label: "Java" },
  native: { icon: "ri-terminal-box-line", label: "Native" },
};

export function runtimeLabel(kind: string): string {
  return RUNTIME_META[kind]?.label ?? kind;
}

export function stateTone(state?: AdminImageState): ImageTone {
  if (!state) return "missing";
  if (state.status === "ready") return state.outdated ? "outdated" : "ready";
  return state.status;
}

const TONE_BADGE: Record<ImageTone, string> = {
  ready: "bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]",
  outdated: "bg-[var(--vx-warn-tint)] text-[var(--vx-warn)]",
  building: "bg-[var(--vx-info-tint)] text-[var(--vx-info)]",
  queued: "bg-muted text-muted-foreground",
  failed: "bg-[var(--vx-danger-tint)] text-[var(--vx-danger)]",
  missing: "bg-muted text-muted-foreground",
};

const TONE_BAR: Record<ImageTone, string> = {
  ready: "bg-[var(--vx-ok)]",
  outdated: "bg-[var(--vx-warn)]",
  building: "bg-[var(--vx-info)]",
  queued: "bg-[var(--vx-info-tint)]",
  failed: "bg-[var(--vx-danger)]",
  missing: "bg-muted-foreground/15",
};

const TONE_ICON: Record<ImageTone, LucideIcon> = {
  ready: CheckCircle2,
  outdated: RefreshCw,
  building: Loader2,
  queued: Clock3,
  failed: XCircle,
  missing: Circle,
};

export function ImageStatusBadge({ state, className }: { state?: AdminImageState; className?: string }) {
  const tone = stateTone(state);
  const Icon = TONE_ICON[tone];
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11.5px] font-medium whitespace-nowrap",
        TONE_BADGE[tone],
        className
      )}
    >
      <Icon className={cn("size-3", tone === "building" && "animate-spin")} />
      {t(`admin.images.status.${tone}`)}
    </span>
  );
}

export type ImageSummary = Record<ImageTone, number> & { total: number };

export function summarize(states: AdminImageStates, nodes: AdminImageNode[]): ImageSummary {
  const out: ImageSummary = { ready: 0, outdated: 0, building: 0, queued: 0, failed: 0, missing: 0, total: 0 };
  for (const node of nodes) {
    const state = states[node.id];
    if (!node.active && !state) continue;
    out[stateTone(state)] += 1;
    out.total += 1;
  }
  return out;
}

const BAR_ORDER: ImageTone[] = ["ready", "outdated", "building", "queued", "failed", "missing"];

export function SummaryBar({ summary, className }: { summary: ImageSummary; className?: string }) {
  if (summary.total === 0) {
    return <div className={cn("h-1.5 rounded-full bg-muted", className)} />;
  }
  return (
    <div className={cn("flex h-1.5 gap-0.5 overflow-hidden rounded-full", className)}>
      {BAR_ORDER.filter((tone) => summary[tone] > 0).map((tone) => (
        <span
          key={tone}
          className={cn("h-full rounded-full", TONE_BAR[tone], tone === "building" && "animate-pulse")}
          style={{ flexGrow: summary[tone], flexBasis: 0 }}
        />
      ))}
    </div>
  );
}

export function summaryText(summary: ImageSummary): string {
  if (summary.total === 0) return t("admin.images.summary.no_nodes");
  const active = summary.building + summary.queued;
  if (active > 0) return t("admin.images.summary.building", { count: active, total: summary.total });
  const ready = summary.ready + summary.outdated;
  if (summary.failed > 0) {
    return t("admin.images.summary.failed", { failed: summary.failed, ready, total: summary.total });
  }
  if (ready === 0) return t("admin.images.summary.none");
  if (summary.outdated > 0 && summary.ready === 0) {
    return t("admin.images.summary.outdated_only", { outdated: summary.outdated, total: summary.total });
  }
  if (summary.outdated > 0) {
    return t("admin.images.summary.outdated", {
      ready: summary.ready,
      total: summary.total,
      outdated: summary.outdated,
    });
  }
  return t("admin.images.summary.ready", { ready, total: summary.total });
}

export function stateDetail(state?: AdminImageState): string {
  if (!state) return t("admin.images.detail.missing");
  switch (state.status) {
    case "ready":
      if (state.outdated) {
        return t("admin.images.detail.outdated", {
          ago: state.built_at ? timeAgo(state.built_at) : "—",
          ref: state.recipe_ref ?? "",
        });
      }
      return state.built_at
        ? t("admin.images.detail.built", { ago: timeAgo(state.built_at) })
        : t("admin.images.detail.ready");
    case "building":
      return state.started_at
        ? t("admin.images.detail.building", { ago: timeAgo(state.started_at) })
        : t("admin.images.status.building");
    case "queued":
      return t("admin.images.detail.queued");
    case "failed":
      if (state.interrupted) return t("admin.images.detail.interrupted");
      return state.built_at
        ? t("admin.images.detail.failed_previous", { ago: timeAgo(state.built_at) })
        : t("admin.images.detail.failed");
  }
}

export function hasActivity(data?: AdminImagesData): boolean {
  if (!data) return false;
  if (data.nodes.some((node) => node.building)) return true;
  const busy = (states: AdminImageStates) =>
    Object.values(states).some((s) => s.status === "queued" || s.status === "building");
  return data.runtimes.some((rt) => busy(rt.states)) || data.images.some((img) => busy(img.states));
}

export function nodeReasonText(node: AdminImageNode): string {
  return node.reason ? t(`admin.images.reason.${node.reason}`) : "";
}

export function imagesErrorText(error: unknown): string {
  const message = error instanceof Error ? error.message : "";
  if (message === "no_nodes" || message === "unknown_image") {
    return t(`admin.images.error.${message}`);
  }
  return message || t("admin.images.error.build");
}
