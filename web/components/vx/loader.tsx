"use client";

import { useEffect, useRef, useState } from "react";
import { useIsFetching, useIsMutating } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function VxSpinner({
  size = 20,
  className,
  label,
}: {
  size?: number;
  className?: string;
  label?: string;
}) {
  const t = useT();
  const text = label ?? t("layout.loader.default");
  const stroke = Math.max(1.5, size / 11);
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className={cn("vx-spin", className)}
      role="status"
      aria-label={text}
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="currentColor"
        strokeWidth={stroke}
        opacity={0.18}
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="currentColor"
        strokeWidth={stroke}
        strokeLinecap="round"
        strokeDasharray={`${c * 0.28} ${c}`}
      />
    </svg>
  );
}

function VxMark({ size = 34 }: { size?: number }) {
  return (
    <span
      aria-hidden
      className="vx-breathe block bg-[var(--vx-fg-strong)]"
      style={{
        width: size,
        height: size,
        clipPath: "polygon(50% 0,100% 28%,100% 72%,50% 100%,0 72%,0 28%)",
      }}
    />
  );
}

export function VxPageLoader({
  label,
  hint,
  delayMs = 250,
  className,
}: {
  label?: string;
  hint?: string;
  delayMs?: number;
  className?: string;
}) {
  const t = useT();
  const text = label ?? t("layout.loader.panel");
  const [visible, setVisible] = useState(delayMs === 0);

  useEffect(() => {
    if (delayMs === 0) return;
    const timer = setTimeout(() => setVisible(true), delayMs);
    return () => clearTimeout(timer);
  }, [delayMs]);

  return (
    <div
      className={cn(
        "flex min-h-screen flex-col items-center justify-center gap-5 p-6",
        "transition-opacity duration-300",
        visible ? "opacity-100" : "opacity-0",
        className
      )}
      role="status"
      aria-live="polite"
      aria-busy
    >
      <span className="relative flex h-[72px] w-[72px] items-center justify-center">
        <VxSpinner size={72} className="absolute inset-0 text-[var(--vx-fg-strong)]" label={text} />
        <VxMark />
      </span>
      <span className="flex flex-col items-center gap-1.5 text-center">
        <span className="text-[13.5px] font-medium text-foreground">{text}</span>
        {hint && <span className="text-[12px] text-muted-foreground">{hint}</span>}
      </span>
    </div>
  );
}

export function VxInlineLoader({
  label,
  className,
}: {
  label?: string;
  className?: string;
}) {
  const t = useT();
  const text = label ?? t("common.loading");
  return (
    <div
      className={cn("flex items-center justify-center gap-2.5 py-10", className)}
      role="status"
      aria-live="polite"
    >
      <VxSpinner size={16} className="text-muted-foreground" />
      <span className="text-[12.5px] text-muted-foreground">{text}</span>
    </div>
  );
}

export function VxRouteProgress() {
  const fetching = useIsFetching();
  const mutating = useIsMutating();
  const active = fetching + mutating > 0;

  const [visible, setVisible] = useState(false);
  const shownAtRef = useRef(0);

  useEffect(() => {
    let showTimer: ReturnType<typeof setTimeout> | undefined;
    let hideTimer: ReturnType<typeof setTimeout> | undefined;

    if (active) {
      showTimer = setTimeout(() => {
        shownAtRef.current = Date.now();
        setVisible(true);
      }, 300);
    } else {
      const shownFor = Date.now() - shownAtRef.current;
      const wait = shownAtRef.current > 0 ? Math.max(0, 400 - shownFor) : 0;
      hideTimer = setTimeout(() => setVisible(false), wait);
    }

    return () => {
      if (showTimer) clearTimeout(showTimer);
      if (hideTimer) clearTimeout(hideTimer);
    };
  }, [active]);

  return (
    <div
      aria-hidden
      className={cn(
        "pointer-events-none fixed inset-x-0 top-0 z-[100] h-0.5 overflow-hidden transition-opacity duration-200",
        visible ? "opacity-100" : "opacity-0"
      )}
    >
      <div className="vx-indeterminate h-full w-full bg-[var(--vx-fg-strong)]" />
    </div>
  );
}
