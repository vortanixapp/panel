"use client";

import { useEffect, useRef, useState } from "react";
import { animate, useReducedMotionConfig } from "motion/react";
import { EASE_OUT } from "@/components/motion-root";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function useSpotlight() {
  useEffect(() => {
    if (!window.matchMedia("(hover: hover) and (pointer: fine)").matches) return;
    let frame = 0;
    let last: PointerEvent | null = null;

    const apply = () => {
      frame = 0;
      if (!last || !(last.target instanceof Element)) return;
      const target = last.target.closest<HTMLElement>("[data-spotlight]");
      if (!target) return;
      const rect = target.getBoundingClientRect();
      target.style.setProperty("--spot-x", `${last.clientX - rect.left}px`);
      target.style.setProperty("--spot-y", `${last.clientY - rect.top}px`);
    };

    const onMove = (event: PointerEvent) => {
      last = event;
      if (!frame) frame = requestAnimationFrame(apply);
    };

    document.addEventListener("pointermove", onMove, { passive: true });
    return () => {
      document.removeEventListener("pointermove", onMove);
      if (frame) cancelAnimationFrame(frame);
    };
  }, []);
}

function formatInteger(value: number) {
  return Math.round(value).toLocaleString(localeTag());
}

export function AnimatedNumber({
  value,
  format = formatInteger,
  duration = 0.9,
  className,
}: {
  value: number;
  format?: (value: number) => string;
  duration?: number;
  className?: string;
}) {
  const reduced = useReducedMotionConfig();
  const current = useRef(0);
  const formatRef = useRef(format);
  const [display, setDisplay] = useState(() => format(0));

  useEffect(() => {
    formatRef.current = format;
  });

  useEffect(() => {
    if (reduced || !Number.isFinite(value)) {
      current.current = value;
      setDisplay(formatRef.current(value));
      return;
    }
    const controls = animate(current.current, value, {
      duration,
      ease: EASE_OUT,
      onUpdate: (latest) => {
        current.current = latest;
        setDisplay(formatRef.current(latest));
      },
    });
    return () => controls.stop();
  }, [value, duration, reduced]);

  return <span className={cn("tabular-nums", className)}>{display}</span>;
}
