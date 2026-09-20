"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  animate,
  m,
  useInView,
  useMotionValue,
  useReducedMotionConfig,
  useSpring,
} from "motion/react";
import { EASE_OUT } from "@/components/motion-root";
import { useSite } from "@/context/site-provider";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export { EASE_OUT, MotionRoot } from "@/components/motion-root";

export function Reveal({
  children,
  className,
  delay = 0,
  y = 28,
  as = "div",
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
  y?: number;
  as?: "div" | "li" | "section" | "article";
}) {
  const Component = m[as];
  return (
    <Component
      data-reveal
      className={className}
      initial={{ opacity: 0, y }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "0px 0px -12% 0px" }}
      transition={{ duration: 0.8, ease: EASE_OUT, delay }}
    >
      {children}
    </Component>
  );
}

export function WordsReveal({
  text,
  className,
  delay = 0,
  immediate = false,
}: {
  text: string;
  className?: string;
  delay?: number;
  immediate?: boolean;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const { inEditor } = useSite();
  const inView = useInView(ref, { once: true, margin: "0px 0px -10% 0px" });
  const words = text.split(/\s+/).filter(Boolean);
  const state = { hidden: { y: "110%" }, shown: { y: "0%" } };
  return (
    <m.span
      ref={ref}
      className={cn("inline", className)}
      initial={immediate || inEditor ? "shown" : "hidden"}
      animate={immediate || inEditor || inView ? "shown" : "hidden"}
      transition={{ staggerChildren: 0.045, delayChildren: delay }}
      aria-label={text}
    >
      {words.map((word, index) => (
        <span
          key={index}
          aria-hidden
          className="inline-block overflow-hidden pb-[0.12em] align-top [margin-bottom:-0.12em]"
        >
          <m.span
            data-reveal
            className="inline-block will-change-transform"
            variants={state}
            transition={{ duration: 0.9, ease: EASE_OUT }}
          >
            {word}
            {index < words.length - 1 ? " " : ""}
          </m.span>
        </span>
      ))}
    </m.span>
  );
}

export function Magnetic({
  children,
  className,
  strength = 0.25,
}: {
  children: ReactNode;
  className?: string;
  strength?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotionConfig();
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const springX = useSpring(x, { stiffness: 260, damping: 18, mass: 0.4 });
  const springY = useSpring(y, { stiffness: 260, damping: 18, mass: 0.4 });

  return (
    <m.div
      ref={ref}
      className={cn("inline-block", className)}
      style={{ x: springX, y: springY }}
      onPointerMove={(event) => {
        if (reduced || event.pointerType !== "mouse" || !ref.current) return;
        const rect = ref.current.getBoundingClientRect();
        x.set((event.clientX - rect.left - rect.width / 2) * strength);
        y.set((event.clientY - rect.top - rect.height / 2) * strength);
      }}
      onPointerLeave={() => {
        x.set(0);
        y.set(0);
      }}
    >
      {children}
    </m.div>
  );
}

function formatNumber(value: number, decimals: number) {
  return value.toLocaleString(localeTag(), {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

export function CountUp({
  value,
  decimals = 0,
  duration = 1.4,
  className,
}: {
  value: number;
  decimals?: number;
  duration?: number;
  className?: string;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true, margin: "0px 0px -10% 0px" });
  const reduced = useReducedMotionConfig();
  const [display, setDisplay] = useState(() => formatNumber(value, decimals));

  useEffect(() => {
    if (!inView) return;
    if (reduced) {
      setDisplay(formatNumber(value, decimals));
      return;
    }
    const controls = animate(0, value, {
      duration,
      ease: EASE_OUT,
      onUpdate: (latest) => setDisplay(formatNumber(latest, decimals)),
    });
    return () => controls.stop();
  }, [inView, value, decimals, duration, reduced]);

  return (
    <span ref={ref} className={cn("tabular-nums", className)}>
      {display}
    </span>
  );
}

export function SectionLabel({
  index,
  children,
  className,
}: {
  index: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-3 font-mono text-[12px] tracking-[0.04em] text-muted-foreground",
        className
      )}
    >
      <span className="text-foreground">{index}</span>
      <span className="h-px w-8 bg-border" />
      <span>{children}</span>
    </div>
  );
}
