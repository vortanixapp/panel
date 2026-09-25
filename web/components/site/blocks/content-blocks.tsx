"use client";

import { useEffect, useMemo, useRef } from "react";
import { ImageIcon, PlayCircle } from "lucide-react";
import { useSite } from "@/context/site-provider";
import { useT } from "@/hooks/use-translations";
import { sanitizeCustomHtml, sanitizeRichHtml, videoEmbedUrl } from "@/components/site/sanitize";
import {
  BlockButton,
  BlockLink,
  BlockShell,
  Field,
  boolProp,
  resolveImage,
  stringProp,
  useBlockText,
  type BlockViewProps,
} from "@/components/site/blocks/shared";
import { cn } from "@/lib/utils";

const HEADING_SIZES = {
  site: {
    md: "text-[1.6rem] sm:text-[2rem]",
    lg: "text-[2rem] sm:text-[2.6rem]",
    xl: "text-[2.4rem] sm:text-[3.4rem]",
  },
  panel: {
    md: "text-lg",
    lg: "text-xl",
    xl: "text-2xl",
  },
} as const;

type HeadingSize = keyof (typeof HEADING_SIZES)["site"];

export function HeadingBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const center = stringProp(block, "align") === "center";
  const rawSize = stringProp(block, "size");
  const size: HeadingSize = rawSize === "md" || rawSize === "xl" ? rawSize : "lg";
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant} inner={site ? "py-14 lg:py-20" : undefined}>
      <div className={cn(site ? "max-w-[860px]" : "max-w-3xl", center && "mx-auto text-center")}>
        <Field
          block={block}
          path="eyebrow"
          value={text(block.props?.eyebrow)}
          as="div"
          className={site ? "font-mono text-[12px] tracking-[0.04em] text-muted-foreground" : "text-xs font-medium tracking-wide text-muted-foreground uppercase"}
        />
        <Field
          block={block}
          path="title"
          value={text(block.props?.title)}
          as="h2"
          className={cn(
            site
              ? "vx-display mt-4 leading-[1.08] font-semibold tracking-[-0.03em] text-balance"
              : "mt-1 font-semibold tracking-tight",
            HEADING_SIZES[variant][size]
          )}
        />
        <Field
          block={block}
          path="text"
          multiline
          value={text(block.props?.text)}
          as="p"
          className={cn(
            "whitespace-pre-line text-muted-foreground",
            site ? "mt-5 text-[16px] leading-[1.6]" : "mt-2 text-sm leading-6",
            center && "mx-auto"
          )}
        />
      </div>
    </BlockShell>
  );
}

export function TextBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const t = useT();
  const { editing } = useSite();
  const raw = text(block.props?.html);
  const html = useMemo(() => (raw ? sanitizeRichHtml(raw) : ""), [raw]);
  const wide = stringProp(block, "width") === "wide";
  const center = stringProp(block, "align") === "center";
  const site = variant === "site";
  if (!html && !editing) return null;
  return (
    <BlockShell block={block} variant={variant} inner={site ? "py-12 lg:py-16" : undefined}>
      <div className={cn(!wide && (site ? "max-w-[760px]" : "max-w-3xl"), center && "mx-auto text-center", !site && "rounded-xl border bg-card p-5 sm:p-6")}>
        {html ? (
          <div className={cn("vx-article", site && "text-[16px]")} dangerouslySetInnerHTML={{ __html: html }} />
        ) : (
          <p className="text-sm text-muted-foreground opacity-60">{t("template.field.empty_text")}</p>
        )}
      </div>
    </BlockShell>
  );
}

function LiveHtml({ raw }: { raw: string }) {
  const host = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = host.current;
    if (!node) return;
    node.innerHTML = raw;
    node.querySelectorAll("script").forEach((old) => {
      const script = document.createElement("script");
      for (const attr of Array.from(old.attributes)) {
        script.setAttribute(attr.name, attr.value);
      }
      script.textContent = old.textContent;
      old.replaceWith(script);
    });
    return () => {
      node.innerHTML = "";
    };
  }, [raw]);
  return <div ref={host} className="vx-custom-html" />;
}

export function HtmlBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const t = useT();
  const { editing } = useSite();
  const raw = text(block.props?.html);
  const live = boolProp(block, "scripts") && !editing;
  const html = useMemo(() => (raw ? sanitizeCustomHtml(raw) : ""), [raw]);
  if (!html && !editing) return null;
  return (
    <BlockShell block={block} variant={variant} inner={variant === "site" ? "py-12 lg:py-16" : undefined}>
      {live && raw ? (
        <LiveHtml raw={raw} />
      ) : html ? (
        <div className="vx-custom-html" dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <p className="rounded-xl border border-dashed p-6 text-center text-sm text-muted-foreground">
          {t("template.field.empty_html")}
        </p>
      )}
    </BlockShell>
  );
}

const SPACER = { sm: "h-4", md: "h-8", lg: "h-16", xl: "h-24" } as const;

export function SpacerBlock({ block, variant }: BlockViewProps) {
  const raw = stringProp(block, "size");
  const size = raw === "sm" || raw === "lg" || raw === "xl" ? raw : "md";
  const line = boolProp(block, "line");
  return (
    <div className={cn(variant === "site" && "mx-auto max-w-[1240px] px-5 sm:px-8", "flex items-center", SPACER[size])}>
      {line ? <div className="h-px w-full bg-border" /> : null}
    </div>
  );
}

function ImagePlaceholder({ className }: { className?: string }) {
  return (
    <div className={cn("flex aspect-[16/9] w-full items-center justify-center rounded-2xl border border-dashed bg-muted/40 text-muted-foreground", className)}>
      <ImageIcon className="size-8 opacity-60" />
    </div>
  );
}

export function ImageBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const { editing } = useSite();
  const src = resolveImage(stringProp(block, "src"));
  const href = stringProp(block, "url");
  const width = stringProp(block, "width");
  const rounded = boolProp(block, "rounded");
  const site = variant === "site";
  if (!src && !editing) return null;
  const image = src ? (
    <img
      src={src}
      alt={text(block.props?.alt)}
      loading="lazy"
      className={cn("w-full object-cover", rounded ? "rounded-2xl border border-border" : "rounded-md")}
    />
  ) : (
    <ImagePlaceholder />
  );
  return (
    <BlockShell
      block={block}
      variant={variant}
      inner={cn(site && "py-12 lg:py-16", width === "full" && site && "max-w-none px-0 sm:px-0")}
    >
      <figure className={cn(width === "narrow" && "mx-auto max-w-[720px]", width !== "full" && width !== "narrow" && site && "mx-auto max-w-[1080px]")}>
        {href && !editing ? (
          <BlockLink href={href} newTab={boolProp(block, "new_tab")} className="block">
            {image}
          </BlockLink>
        ) : (
          image
        )}
        <Field
          block={block}
          path="caption"
          value={text(block.props?.caption)}
          as="figcaption"
          className={cn("mt-3 text-center text-muted-foreground", site ? "text-[14px]" : "text-xs")}
        />
      </figure>
    </BlockShell>
  );
}

export function MediaBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const src = resolveImage(stringProp(block, "src"));
  const right = stringProp(block, "side") === "right";
  const site = variant === "site";
  return (
    <BlockShell block={block} variant={variant}>
      <div
        className={cn(
          "grid items-center gap-8",
          site ? "lg:grid-cols-2 lg:gap-16" : "rounded-xl border bg-card p-5 sm:p-6 md:grid-cols-2",
        )}
      >
        <div className={cn(right && "lg:order-2 md:order-2")}>
          {src ? (
            <img
              src={src}
              alt={text(block.props?.alt)}
              loading="lazy"
              className={cn("w-full object-cover", site ? "rounded-3xl border border-border" : "rounded-lg")}
            />
          ) : (
            <ImagePlaceholder />
          )}
        </div>
        <div>
          <Field
            block={block}
            path="eyebrow"
            value={text(block.props?.eyebrow)}
            as="div"
            className={site ? "font-mono text-[12px] tracking-[0.04em] text-muted-foreground" : "text-xs font-medium tracking-wide text-muted-foreground uppercase"}
          />
          <Field
            block={block}
            path="title"
            value={text(block.props?.title)}
            as="h2"
            className={site ? "vx-display mt-4 text-[1.9rem] leading-[1.1] font-semibold tracking-[-0.03em] text-balance sm:text-[2.3rem]" : "mt-1 text-xl font-semibold tracking-tight"}
          />
          <Field
            block={block}
            path="text"
            multiline
            value={text(block.props?.text)}
            as="p"
            className={cn("whitespace-pre-line text-muted-foreground", site ? "mt-5 text-[16px] leading-[1.6]" : "mt-2 text-sm leading-6")}
          />
          <div className={site ? "mt-8" : "mt-4"}>
            <BlockButton
              block={block}
              path="button"
              label={text(block.props?.button)}
              href={stringProp(block, "url")}
              variant={variant}
              primary
            />
          </div>
        </div>
      </div>
    </BlockShell>
  );
}

export function VideoBlock({ block, variant }: BlockViewProps) {
  const text = useBlockText();
  const { editing } = useSite();
  const embed = videoEmbedUrl(stringProp(block, "url"));
  const title = text(block.props?.title);
  const site = variant === "site";
  if (!embed && !editing) return null;
  return (
    <BlockShell block={block} variant={variant} inner={site ? "py-12 lg:py-16" : undefined}>
      <figure className={cn(site && "mx-auto max-w-[1080px]")}>
        <Field
          block={block}
          path="title"
          value={title}
          as="h2"
          className={site ? "vx-display mb-6 text-[1.8rem] leading-[1.1] font-semibold tracking-[-0.03em]" : "mb-3 text-lg font-semibold"}
        />
        <div className={cn("relative aspect-video w-full overflow-hidden border border-border bg-muted", site ? "rounded-3xl" : "rounded-xl")}>
          {embed && !editing ? (
            <iframe
              src={embed}
              title={title || "video"}
              loading="lazy"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullScreen
              referrerPolicy="strict-origin-when-cross-origin"
              className="absolute inset-0 size-full"
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
              <PlayCircle className="size-12 opacity-60" />
            </div>
          )}
        </div>
        <Field
          block={block}
          path="caption"
          value={text(block.props?.caption)}
          as="figcaption"
          className={cn("mt-3 text-center text-muted-foreground", site ? "text-[14px]" : "text-xs")}
        />
      </figure>
    </BlockShell>
  );
}
