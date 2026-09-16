"use client";

import Link from "next/link";
import { useId, useState } from "react";
import { AnimatePresence, m } from "motion/react";
import { ArrowUpRight, Plus } from "lucide-react";
import { landingFaq } from "@/components/landing/landing-content";
import { EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export function FaqSection({ index }: { index: string }) {
  const t = useT();
  const { links } = useBrand();
  const items = landingFaq(t);
  const [open, setOpen] = useState<number | null>(0);
  const baseId = useId();
  const contact = links.support || links.telegram || "/support";
  const external = /^https?:\/\//.test(contact);

  return (
    <section id="faq" className="scroll-mt-16 border-b border-border">
      <div className="mx-auto grid max-w-[1240px] gap-12 px-5 py-20 sm:px-8 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] lg:gap-16 lg:py-28">
        <div>
          <div className="lg:sticky lg:top-28">
            <SectionLabel index={index}>{t("landing.faq.label")}</SectionLabel>
            <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
              <WordsReveal text={t("landing.faq.title")} />
            </h2>
            <Reveal>
              <p className="mt-6 max-w-[420px] text-[16px] leading-[1.6] text-muted-foreground">{t("landing.faq.text")}</p>
              <Link
                href={contact}
                {...(external ? { target: "_blank", rel: "noopener noreferrer" } : {})}
                className="group mt-8 inline-flex items-center gap-1.5 text-[15px] font-medium"
              >
                {t("landing.faq.contact")}
                <ArrowUpRight className="size-4 transition-transform duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
              </Link>
            </Reveal>
          </div>
        </div>

        <div className="border-t border-border">
          {items.map((item, i) => {
            const isOpen = open === i;
            const panelId = `${baseId}-panel-${i}`;
            const buttonId = `${baseId}-button-${i}`;
            return (
              <Reveal key={item.q} delay={i * 0.05} y={16} className="border-b border-border">
                <h3>
                  <button
                    id={buttonId}
                    type="button"
                    aria-expanded={isOpen}
                    aria-controls={panelId}
                    onClick={() => setOpen(isOpen ? null : i)}
                    className="group flex w-full items-start gap-5 py-6 text-left"
                  >
                    <span className="w-6 shrink-0 pt-1 font-mono text-[12px] text-muted-foreground">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span
                      className={cn(
                        "flex-1 text-[17px] leading-[1.4] font-medium tracking-[-0.01em] transition-colors sm:text-[19px]",
                        isOpen ? "text-foreground" : "text-foreground/80 group-hover:text-foreground"
                      )}
                    >
                      {item.q}
                    </span>
                    <m.span
                      animate={{ rotate: isOpen ? 45 : 0 }}
                      transition={{ duration: 0.35, ease: EASE_OUT }}
                      className={cn(
                        "mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full border transition-colors",
                        isOpen ? "border-foreground bg-foreground text-background" : "border-border"
                      )}
                    >
                      <Plus className="size-3.5" />
                    </m.span>
                  </button>
                </h3>
                <AnimatePresence initial={false}>
                  {isOpen && (
                    <m.div
                      id={panelId}
                      role="region"
                      aria-labelledby={buttonId}
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.45, ease: EASE_OUT }}
                      className="overflow-hidden"
                    >
                      <m.p
                        initial={{ y: -8 }}
                        animate={{ y: 0 }}
                        exit={{ y: -8 }}
                        transition={{ duration: 0.45, ease: EASE_OUT }}
                        className="max-w-[620px] pb-7 pl-11 text-[15.5px] leading-[1.65] text-muted-foreground"
                      >
                        {item.a}
                      </m.p>
                    </m.div>
                  )}
                </AnimatePresence>
              </Reveal>
            );
          })}
        </div>
      </div>
    </section>
  );
}
