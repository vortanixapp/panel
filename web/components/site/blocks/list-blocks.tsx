"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, CheckCircle2, ChevronDown, Info, OctagonAlert, X } from "lucide-react";
import { useSite } from "@/context/site-provider";
import { useT } from "@/hooks/use-translations";
import {
  BlockButton,
  BlockLink,
  BlockShell,
  Field,
  boolProp,
  listProp,
  numberProp,
  stringProp,
  useBlockText,
  type BlockViewProps,
} from "@/components/site/blocks/shared";
import { siteIcon } from "@/lib/site/icons";
import { cn } from "@/lib/utils";

function BlockIntro({ block, variant, center }: BlockViewProps & { center?: boolean }) {
  const text = useBlockText();
  const site = variant === "site";
  const title = text(block.props?.title);
  const body = text(block.props?.text);
  const { editing } = useSite();
  if (!title && !body && !editing) return null;
  return (
    <div className={cn(site ? "mb-12 max-w-[760px]" : "mb-4 max-w-3xl", center && "mx-auto text-center")}>
      <Field
        block={block}
        path="title"
        value={title}
        as="h2"
        className={site ? "vx-display text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]" : "text-lg font-semibold tracking-tight"}
      />
      <Field
        block={block}
        path="text"
        multiline
        value={body}
        as="p"
        className={cn("whitespace-pre-line text-muted-foreground", site ? "mt-5 text-[16px] leading-[1.6]" : "mt-1 text-sm")}
      />
    </div>
  );
}

const COLUMNS = {
  2: "sm:grid-cols-2",
  3: "sm:grid-cols-2 lg:grid-cols-3",
  4: "sm:grid-cols-2 lg:grid-cols-4",
} as const;

export function FeaturesBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const { editing } = useSite();
  const items = listProp(block, "items");
  const raw = numberProp(block, "columns", 3);
  const columns: keyof typeof COLUMNS = raw === 2 || raw === 4 ? raw : 3;
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant}>
      <BlockIntro block={block} variant={variant} />
      <div className={cn("grid gap-4", COLUMNS[columns])}>
        {items.map((item, i) => {
          const Icon = siteIcon(typeof item.icon === "string" ? item.icon : undefined);
          const href = typeof item.url === "string" ? item.url : "";
          const card = (
            <div
              className={cn(
                "flex h-full flex-col border bg-card transition-colors",
                site ? "rounded-3xl p-7" : "rounded-xl p-5",
                href && "hover:border-foreground/30"
              )}
            >
              {Icon ? (
                <span className={cn("flex items-center justify-center rounded-xl bg-primary/10 text-primary", site ? "size-11" : "size-9")}>
                  <Icon className={site ? "size-5" : "size-4"} />
                </span>
              ) : null}
              <Field
                block={block}
                path={`items.${i}.title`}
                value={text(item.title)}
                as="h3"
                className={cn("font-semibold tracking-tight", site ? "mt-5 text-[18px]" : "mt-3 text-[15px]", !Icon && "mt-0")}
              />
              <Field
                block={block}
                path={`items.${i}.text`}
                multiline
                value={text(item.text)}
                as="p"
                className={cn("whitespace-pre-line text-muted-foreground", site ? "mt-2 text-[15px] leading-[1.6]" : "mt-1 text-sm")}
              />
            </div>
          );
          return href && !editing ? (
            <BlockLink key={i} href={href} className="block">
              {card}
            </BlockLink>
          ) : (
            <div key={i}>{card}</div>
          );
        })}
      </div>
    </BlockShell>
  );
}

export function CtaBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const muted = stringProp(block, "tone") === "muted";
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant} className={site ? "border-b-0" : undefined} inner={site ? "py-10 lg:py-14" : undefined}>
      <div
        className={cn(
          "relative overflow-hidden",
          site ? "rounded-[2rem] px-7 py-14 sm:px-12 lg:px-16 lg:py-20" : "rounded-xl p-6",
          muted ? "border bg-muted/60 text-foreground" : "bg-primary text-primary-foreground"
        )}
      >
        <Field
          block={block}
          path="title"
          value={text(block.props?.title)}
          as="h2"
          className={site ? "vx-display max-w-[820px] text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.8rem]" : "text-xl font-semibold tracking-tight"}
        />
        <Field
          block={block}
          path="text"
          multiline
          value={text(block.props?.text)}
          as="p"
          className={cn("max-w-[640px] whitespace-pre-line", muted ? "text-muted-foreground" : "opacity-80", site ? "mt-5 text-[16px] leading-[1.6]" : "mt-2 text-sm")}
        />
        <div className={cn("flex flex-wrap gap-3", site ? "mt-9" : "mt-5")}>
          <BlockButton
            block={block}
            path="button"
            label={text(block.props?.button)}
            href={stringProp(block, "url")}
            variant={variant}
            primary
            onDark={!muted}
          />
          <BlockButton
            block={block}
            path="button2"
            label={text(block.props?.button2)}
            href={stringProp(block, "url2")}
            variant={variant}
            onDark={!muted}
          />
        </div>
      </div>
    </BlockShell>
  );
}

export function FaqBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const { editing } = useSite();
  const items = listProp(block, "items");
  const [open, setOpen] = useState<number | null>(editing ? null : 0);
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant}>
      <div className={cn(site && "grid gap-10 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] lg:gap-16")}>
        <BlockIntro block={block} variant={variant} />
        <div className={cn("border-t border-border", !site && "rounded-xl border bg-card px-5")}>
          {items.map((item, i) => {
            const isOpen = editing || open === i;
            return (
              <div key={i} className="border-b border-border last:border-b-0">
                <button
                  type="button"
                  onClick={() => setOpen(open === i ? null : i)}
                  aria-expanded={isOpen}
                  className={cn("flex w-full items-start gap-4 text-left", site ? "py-6" : "py-4")}
                >
                  <Field
                    block={block}
                    path={`items.${i}.q`}
                    value={text(item.q)}
                    className={cn("flex-1 font-medium tracking-[-0.01em]", site ? "text-[18px] leading-[1.4]" : "text-[15px]")}
                  />
                  <ChevronDown className={cn("mt-1 size-4 shrink-0 text-muted-foreground transition-transform duration-300", isOpen && "rotate-180")} />
                </button>
                {isOpen ? (
                  <Field
                    block={block}
                    path={`items.${i}.a`}
                    multiline
                    value={text(item.a)}
                    as="p"
                    className={cn("whitespace-pre-line text-muted-foreground", site ? "pb-6 text-[16px] leading-[1.65]" : "pb-4 text-sm")}
                  />
                ) : null}
              </div>
            );
          })}
        </div>
      </div>
    </BlockShell>
  );
}

export function StatsBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const items = listProp(block, "items");
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant} inner={site ? "py-14 lg:py-20" : undefined}>
      <Field
        block={block}
        path="title"
        value={text(block.props?.title)}
        as="h2"
        className={site ? "vx-display mb-10 text-[1.8rem] leading-[1.1] font-semibold tracking-[-0.03em]" : "mb-3 text-lg font-semibold"}
      />
      <dl className={cn("grid grid-cols-2 gap-px overflow-hidden border bg-border", site ? "rounded-3xl lg:grid-cols-4" : "rounded-xl md:grid-cols-4")}>
        {items.map((item, i) => (
          <div key={i} className={cn("bg-card", site ? "p-7" : "p-4")}>
            <Field
              block={block}
              path={`items.${i}.value`}
              value={text(item.value)}
              as="dd"
              className={site ? "vx-display text-[2.2rem] leading-none font-semibold tracking-[-0.03em]" : "text-2xl font-semibold"}
            />
            <Field
              block={block}
              path={`items.${i}.label`}
              value={text(item.label)}
              as="dt"
              className={cn("text-muted-foreground", site ? "mt-3 text-[14px]" : "mt-1 text-xs")}
            />
          </div>
        ))}
      </dl>
    </BlockShell>
  );
}

const NOTICE_TONES = {
  info: { icon: Info, box: "bg-[var(--vx-info-tint)]", accent: "text-[var(--vx-info)]" },
  success: { icon: CheckCircle2, box: "bg-[var(--vx-ok-tint)]", accent: "text-[var(--vx-ok)]" },
  warning: { icon: AlertTriangle, box: "bg-[var(--vx-warn-tint)]", accent: "text-[var(--vx-warn)]" },
  danger: { icon: OctagonAlert, box: "bg-[var(--vx-danger-tint)]", accent: "text-[var(--vx-danger)]" },
} as const;

const DISMISS_KEY = "vx-dismissed-notices";

function readDismissed(): string[] {
  try {
    const raw = localStorage.getItem(DISMISS_KEY);
    const parsed = raw ? (JSON.parse(raw) as unknown) : [];
    return Array.isArray(parsed) ? parsed.filter((v): v is string => typeof v === "string") : [];
  } catch {
    return [];
  }
}

function writeDismissed(ids: string[]) {
  try {
    localStorage.setItem(DISMISS_KEY, JSON.stringify(ids.slice(-100)));
  } catch {
    return;
  }
}

export function NoticeBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const t = useT();
  const { editing } = useSite();
  const raw = stringProp(block, "tone");
  const tone = NOTICE_TONES[raw as keyof typeof NOTICE_TONES] ?? NOTICE_TONES.info;
  const dismissible = boolProp(block, "dismissible");
  const [dismissed, setDismissed] = useState(false);
  const site = variant === "site";

  useEffect(() => {
    if (dismissible && !editing) setDismissed(readDismissed().includes(block.id));
  }, [block.id, dismissible, editing]);

  if (dismissed && !editing) return null;
  const Icon = tone.icon;
  return (
    <div className={cn(site && "mx-auto max-w-[1240px] px-5 py-6 sm:px-8")}>
      <div className={cn("flex items-start gap-3 rounded-xl border border-transparent p-4", tone.box)}>
        <Icon className={cn("mt-0.5 size-5 shrink-0", tone.accent)} />
        <div className="min-w-0 flex-1">
          <Field block={block} path="title" value={text(block.props?.title)} as="div" className="text-sm font-semibold" />
          <Field
            block={block}
            path="text"
            multiline
            value={text(block.props?.text)}
            as="p"
            className="mt-0.5 text-sm whitespace-pre-line text-muted-foreground"
          />
          {stringProp(block, "url") || editing ? (
            <div className="mt-2">
              {editing ? (
                <Field block={block} path="button" value={text(block.props?.button)} className={cn("text-sm font-medium", tone.accent)} />
              ) : (
                <BlockLink href={stringProp(block, "url")} className={cn("text-sm font-medium underline-offset-4 hover:underline", tone.accent)}>
                  {text(block.props?.button) || t("template.block.notice.more")}
                </BlockLink>
              )}
            </div>
          ) : null}
        </div>
        {dismissible ? (
          <button
            type="button"
            aria-label={t("common.close")}
            onClick={() => {
              if (editing) return;
              writeDismissed([...readDismissed(), block.id]);
              setDismissed(true);
            }}
            className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-background/60 hover:text-foreground"
          >
            <X className="size-4" />
          </button>
        ) : null}
      </div>
    </div>
  );
}
