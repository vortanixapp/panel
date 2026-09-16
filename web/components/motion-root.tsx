"use client";

import type { ReactNode } from "react";
import { LazyMotion, MotionConfig, domMax, type Transition } from "motion/react";

export const EASE_OUT: Transition["ease"] = [0.22, 1, 0.36, 1];

export function MotionRoot({ children }: { children: ReactNode }) {
  return (
    <LazyMotion features={domMax} strict>
      <MotionConfig reducedMotion="user" transition={{ duration: 0.6, ease: EASE_OUT }}>
        {children}
      </MotionConfig>
    </LazyMotion>
  );
}
