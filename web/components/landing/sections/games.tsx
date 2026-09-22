"use client";

import Image from "next/image";
import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { FEATURED_GAMES, MORE_GAMES } from "@/components/landing/landing-content";
import { Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const LAYOUT = [
  "sm:col-span-2 lg:col-span-2 lg:row-span-2",
  "",
  "",
  "",
  "",
  "",
];

export function GamesSection({ index }: { index: string }) {
  const t = useT();
  const price = (value: number) =>
    t("landing.unit.from_rub_month", { price: value.toLocaleString(localeTag()) });

  return (
    <section id="games" className="scroll-mt-16 border-b border-border">
      <div className="mx-auto max-w-[1240px] px-5 py-20 sm:px-8 lg:py-28">
        <div className="grid gap-8 lg:grid-cols-[1fr_minmax(0,420px)] lg:items-end">
          <div>
            <SectionLabel index={index}>{t("landing.games.label")}</SectionLabel>
            <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
              <WordsReveal text={t("landing.games.title")} />
            </h2>
          </div>
          <Reveal className="flex flex-col gap-4 lg:pb-2">
            <p className="text-[16px] leading-[1.6] text-muted-foreground">
              {t("landing.games.subtitle")}
            </p>
            <Link
              href="/rent-server"
              className="group inline-flex w-fit items-center gap-1.5 text-[15px] font-medium"
            >
              {t("landing.games.all_link")}
              <ArrowUpRight className="size-4 transition-transform duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
            </Link>
          </Reveal>
        </div>

        <div className="mt-12 grid auto-rows-[220px] gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURED_GAMES.map((game, i) => (
            <Reveal key={game.name} delay={i * 0.06} className={cn("min-h-0", LAYOUT[i])}>
              <Link
                href="/rent-server"
                className="group relative flex h-full overflow-hidden rounded-2xl border border-border bg-card"
              >
                <Image
                  src={game.image}
                  alt={game.name}
                  fill
                  unoptimized
                  sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
                  className="object-cover transition-transform duration-[900ms] ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-[1.06]"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/75 via-black/15 to-transparent" />
                <div className="relative mt-auto flex w-full items-end justify-between gap-4 p-5 text-white">
                  <div className="min-w-0">
                    <div
                      className={cn(
                        "vx-display font-semibold tracking-[-0.02em]",
                        i === 0 ? "text-[1.7rem] sm:text-[2.1rem]" : "text-[1.15rem]"
                      )}
                    >
                      {game.name}
                    </div>
                    <div className="mt-1 truncate font-mono text-[12px] text-white/65">{game.note}</div>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-2">
                    <span className="flex size-9 translate-y-2 items-center justify-center rounded-full bg-white text-black opacity-0 transition-[opacity,transform] duration-300 group-hover:translate-y-0 group-hover:opacity-100">
                      <ArrowUpRight className="size-4" />
                    </span>
                    <span className="rounded-full bg-black/45 px-2.5 py-1 font-mono text-[12px] whitespace-nowrap text-white backdrop-blur-sm">
                      {price(game.from)}
                    </span>
                  </div>
                </div>
              </Link>
            </Reveal>
          ))}
        </div>

        <Reveal className="mt-3">
          <div className="no-scrollbar flex overflow-x-auto rounded-2xl border border-border">
            {MORE_GAMES.map((game) => (
              <Link
                key={game.name}
                href="/rent-server"
                className="group flex min-w-[180px] flex-1 flex-col gap-1 border-r border-border px-5 py-4 transition-colors last:border-r-0 hover:bg-accent"
              >
                <span className="text-[14.5px] font-medium whitespace-nowrap">{game.name}</span>
                <span className="font-mono text-[12px] whitespace-nowrap text-muted-foreground">
                  {price(game.from)}
                </span>
              </Link>
            ))}
          </div>
        </Reveal>
      </div>
    </section>
  );
}
