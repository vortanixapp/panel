"use client";

import Link from "next/link";
import { m, useMotionTemplate, useMotionValue, useReducedMotionConfig, useSpring } from "motion/react";
import { ArrowRight } from "lucide-react";
import { EASE_OUT, Magnetic, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";

const DOTS =
  "radial-gradient(circle at center, currentColor 1.1px, transparent 1.3px)";

export function CtaSection({ loggedIn }: { loggedIn: boolean }) {
  const t = useT();
  const reduced = useReducedMotionConfig();
  const pointerX = useMotionValue(-600);
  const pointerY = useMotionValue(-600);
  const x = useSpring(pointerX, { stiffness: 180, damping: 26, mass: 0.6 });
  const y = useSpring(pointerY, { stiffness: 180, damping: 26, mass: 0.6 });
  const mask = useMotionTemplate`radial-gradient(260px circle at ${x}px ${y}px, black 0%, transparent 75%)`;

  return (
    <section className="px-3 py-3 sm:px-4 sm:py-4">
      <m.div
        initial={{ opacity: 0, scale: 0.97 }}
        whileInView={{ opacity: 1, scale: 1 }}
        viewport={{ once: true, margin: "0px 0px -10% 0px" }}
        transition={{ duration: 0.9, ease: EASE_OUT }}
        data-reveal
        onPointerMove={(event) => {
          if (reduced || event.pointerType !== "mouse") return;
          const rect = event.currentTarget.getBoundingClientRect();
          pointerX.set(event.clientX - rect.left);
          pointerY.set(event.clientY - rect.top);
        }}
        onPointerLeave={() => {
          pointerX.set(-600);
          pointerY.set(-600);
        }}
        className="relative overflow-hidden rounded-[28px] bg-foreground text-background"
      >
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 opacity-[0.13]"
          style={{ backgroundImage: DOTS, backgroundSize: "22px 22px" }}
        />
        <m.div
          aria-hidden
          className="pointer-events-none absolute inset-0 opacity-70"
          style={{
            backgroundImage: DOTS,
            backgroundSize: "22px 22px",
            maskImage: mask,
            WebkitMaskImage: mask,
          }}
        />

        <div className="relative mx-auto flex max-w-[1240px] flex-col items-start gap-10 px-6 py-16 sm:px-12 sm:py-24 lg:flex-row lg:items-end lg:justify-between lg:py-28">
          <div className="max-w-[760px]">
            <h2 className="vx-display text-[2.3rem] leading-[1.02] font-semibold tracking-[-0.035em] text-balance sm:text-[3.4rem] lg:text-[4.2rem]">
              <WordsReveal text={t("landing.cta.title")} />
            </h2>
            <p className="mt-6 max-w-[540px] text-[17px] leading-[1.6] text-background/70">{t("landing.cta.text")}</p>
          </div>

          <div className="flex shrink-0 flex-col items-start gap-5">
            <Magnetic strength={0.3}>
              <Link
                href={loggedIn ? "/dashboard" : "/register"}
                className="group inline-flex items-center gap-4 rounded-full bg-background py-4 pr-4 pl-7 text-[16px] font-semibold text-foreground transition-transform active:scale-[0.97]"
              >
                {loggedIn ? t("landing.header.to_panel") : t("landing.cta.button")}
                <span className="flex size-9 items-center justify-center rounded-full bg-foreground text-background transition-transform duration-300 group-hover:translate-x-1">
                  <ArrowRight className="size-4" />
                </span>
              </Link>
            </Magnetic>
            {!loggedIn && (
              <Link
                href="/login"
                className="text-[14px] text-background/65 underline-offset-4 transition-colors hover:text-background hover:underline"
              >
                {t("landing.cta.login")}
              </Link>
            )}
          </div>
        </div>
      </m.div>
    </section>
  );
}
