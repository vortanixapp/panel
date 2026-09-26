"use client";

import { Check, Minus } from "lucide-react";

import { VX_CARD, VX_FAINT, VX_MONO_LABEL, VX_MUTED } from "@/components/vx/panel-ui";
import { cn } from "@/lib/utils";

export function formatBytes(n: number) {
  if (!n) return "0";
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GB`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(n / 1024))} KB`;
}

export function SectionLabel({ children }: { children: React.ReactNode }) {
  return <div className={cn(VX_MONO_LABEL, "tracking-[0.09em]")}>{children}</div>;
}

export function MarkedItem({ on, children }: { on: boolean; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2.5 py-[7px] text-[12.5px] leading-[1.5]">
      <span
        className={cn(
          "mt-[1px] inline-flex size-[17px] shrink-0 items-center justify-center rounded-full",
          on
            ? "bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]"
            : cn("bg-[var(--vx-inset)]", VX_FAINT)
        )}
      >
        {on ? <Check className="size-[11px]" /> : <Minus className="size-[11px]" />}
      </span>
      <span className={on ? undefined : VX_MUTED}>{children}</span>
    </div>
  );
}

export function ChoiceCard({
  active,
  icon,
  title,
  desc,
  disabled,
  onSelect,
}: {
  active: boolean;
  icon: React.ReactNode;
  title: string;
  desc: string;
  disabled?: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onSelect}
      className={cn(
        "group flex w-full items-start gap-3 rounded-[12px] p-3.5 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-55",
        VX_CARD,
        active
          ? "border-[var(--vx-fg-strong)] bg-[var(--vx-elevated)] shadow-[inset_0_0_0_1px_var(--vx-fg-strong)]"
          : "hover:border-[var(--vx-border-hover)]"
      )}
    >
      <span
        className={cn(
          "inline-flex size-[30px] shrink-0 items-center justify-center rounded-[9px] transition-colors",
          active
            ? "bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]"
            : cn("bg-[var(--vx-inset)]", VX_MUTED)
        )}
      >
        {icon}
      </span>
      <span className="grid gap-1">
        <span className="text-[12.5px] font-medium text-[var(--vx-fg)]">{title}</span>
        <span className={cn("text-[11.5px] leading-[1.5]", VX_FAINT)}>{desc}</span>
      </span>
    </button>
  );
}

export function StatTile({
  label,
  value,
  sub,
  icon,
}: {
  label: string;
  value: React.ReactNode;
  sub: React.ReactNode;
  icon: React.ReactNode;
}) {
  return (
    <div data-spotlight className={cn("relative isolate rounded-[14px] px-[18px] py-4", VX_CARD)}>
      <div className="flex items-start justify-between gap-2">
        <SectionLabel>{label}</SectionLabel>
        <span className={cn("inline-flex size-[22px] items-center justify-center rounded-[7px] bg-[var(--vx-inset)]", VX_MUTED)}>
          {icon}
        </span>
      </div>
      <div className="mt-2 font-mono text-[22px] leading-none font-medium tracking-[-0.02em] text-[var(--vx-fg)]">
        {value}
      </div>
      <div className={cn("mt-1.5 text-[11.5px]", VX_FAINT)}>{sub}</div>
    </div>
  );
}

export function StatusChip({ status, label }: { status: string; label: string }) {
  const tone =
    status === "completed"
      ? "bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]"
      : status === "failed"
        ? "bg-[rgba(224,122,122,0.12)] text-[var(--vx-danger)]"
        : status === "cancelled"
          ? cn("bg-[var(--vx-inset)]", VX_MUTED)
          : "bg-[rgba(232,160,60,0.12)] text-[var(--vx-warn)]";
  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-[3px] text-[11px] font-medium", tone)}>
      {label}
    </span>
  );
}
