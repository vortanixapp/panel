"use client";

import Link from "next/link";
import { useLayoutEffect, useRef, useState } from "react";
import { m } from "motion/react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

export type ResponsiveTab = {
  key: string;
  label: string;
  href: string;
  disabled?: boolean;
  disabledHint?: string;
};

const TAB =
  "h-[30px] shrink-0 rounded-[8px] border px-3 text-[12.5px] font-medium whitespace-nowrap transition-colors";
const GAP = 4;

function TabItem({
  tab,
  active,
  pillId,
}: {
  tab: ResponsiveTab;
  active: boolean;
  pillId?: string;
}) {
  if (tab.disabled) {
    return (
      <span
        title={tab.disabledHint}
        className={cn(
          TAB,
          "inline-flex grow cursor-not-allowed items-center justify-center border-transparent text-[var(--vx-border-hover)]"
        )}
      >
        {tab.label}
      </span>
    );
  }
  return (
    <Link
      href={tab.href}
      aria-current={active ? "page" : undefined}
      className={cn(
        TAB,
        "relative inline-flex grow items-center justify-center",
        active
          ? cn(
              "text-[var(--vx-fg-strong)]",
              !pillId && "border-[var(--vx-border-strong)] bg-[var(--vx-tint)]"
            )
          : "border-transparent text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
      )}
    >
      {active && pillId && (
        <m.span
          layoutId={pillId}
          className="absolute -inset-px rounded-[8px] border border-[var(--vx-border-strong)] bg-[var(--vx-tint)]"
          transition={{ type: "spring", stiffness: 480, damping: 38 }}
        />
      )}
      <span className="relative">{tab.label}</span>
    </Link>
  );
}

export function ResponsiveTabs({
  tabs,
  activeKey,
  pillId,
}: {
  tabs: ResponsiveTab[];
  activeKey: string;
  pillId?: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [compact, setCompact] = useState(false);
  const signature = tabs.map((tab) => `${tab.key}:${tab.label}`).join("|");

  useLayoutEffect(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure) return;
    const recalc = () => {
      const widths = Array.from(measure.children).map((el) => (el as HTMLElement).offsetWidth);
      const total = widths.reduce((sum, w) => sum + w, 0) + GAP * Math.max(0, widths.length - 1);
      setCompact(total > container.clientWidth);
    };
    recalc();
    const observer = new ResizeObserver(recalc);
    observer.observe(container);
    return () => observer.disconnect();
  }, [signature]);

  const current = tabs.find((tab) => tab.key === activeKey) ?? tabs[0];

  return (
    <div ref={containerRef} className="relative">
      {compact ? (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="flex h-[36px] w-full items-center justify-between gap-2 rounded-[9px] border border-[var(--vx-border-strong)] bg-[var(--vx-tint)] px-3.5 text-[13px] font-medium text-[var(--vx-fg-strong)] transition-colors"
            >
              <span className="truncate">{current?.label}</span>
              <i className="ri-arrow-down-s-line text-[17px] text-[var(--vx-muted)]" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            collisionPadding={8}
            className="max-h-(--radix-dropdown-menu-content-available-height) w-(--radix-dropdown-menu-trigger-width) overflow-y-auto overscroll-contain border-[var(--vx-border-2)] bg-[var(--vx-elevated)]"
          >
            {tabs.map((tab) =>
              tab.disabled ? (
                <DropdownMenuItem key={tab.key} disabled title={tab.disabledHint}>
                  {tab.label}
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem key={tab.key} asChild>
                  <Link
                    href={tab.href}
                    className={cn(
                      "flex cursor-pointer items-center justify-between py-2 text-[13px]",
                      tab.key === activeKey && "font-medium text-[var(--vx-fg-strong)]"
                    )}
                  >
                    {tab.label}
                    {tab.key === activeKey && <i className="ri-check-line text-[15px]" />}
                  </Link>
                </DropdownMenuItem>
              )
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
        <div className="flex items-center gap-1">
          {tabs.map((tab) => (
            <TabItem key={tab.key} tab={tab} active={tab.key === activeKey} pillId={pillId} />
          ))}
        </div>
      )}

      <div
        ref={measureRef}
        aria-hidden
        className="pointer-events-none invisible absolute top-0 left-0 flex h-0 gap-1 overflow-hidden"
      >
        {tabs.map((tab) => (
          <span key={tab.key} className={cn(TAB, "inline-flex items-center")}>
            {tab.label}
          </span>
        ))}
      </div>
    </div>
  );
}
