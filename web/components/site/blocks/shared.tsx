"use client";

import Link from "next/link";
import type { ElementType, ReactNode } from "react";
import { ArrowRight } from "lucide-react";
import { useSite } from "@/context/site-provider";
import { useSiteText } from "@/hooks/use-site";
import { useT } from "@/hooks/use-translations";
import { brandingUploadUrl } from "@/lib/api";
import { isExternalUrl } from "@/lib/site/menu";
import type { LText, SiteBlock } from "@/lib/site/types";
import { cn } from "@/lib/utils";

export type BlockVariant = "site" | "panel";

export type BlockViewProps = {
  block: SiteBlock;
  variant: BlockVariant;
};

export function isLText(value: unknown): value is LText {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

export function stringProp(block: SiteBlock, name: string): string {
  const value = block.props?.[name];
  return typeof value === "string" ? value : "";
}

export function boolProp(block: SiteBlock, name: string): boolean {
  return block.props?.[name] === true;
}

export function numberProp(block: SiteBlock, name: string, fallback: number): number {
  const value = block.props?.[name];
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

export function listProp(block: SiteBlock, name: string): Record<string, unknown>[] {
  const value = block.props?.[name];
  if (!Array.isArray(value)) return [];
  return value.map((item) => (item && typeof item === "object" && !Array.isArray(item) ? (item as Record<string, unknown>) : {}));
}

export function useBlockText() {
  const text = useSiteText();
  return (value: unknown) => text(isLText(value) ? value : undefined);
}

export function resolveImage(src: string): string {
  if (!src) return "";
  if (src.startsWith("branding/")) return brandingUploadUrl(src);
  return src;
}

export function Field({
  block,
  path,
  value,
  as,
  className,
  multiline,
  placeholder,
}: {
  block: SiteBlock;
  path: string;
  value: string;
  as?: ElementType;
  className?: string;
  multiline?: boolean;
  placeholder?: string;
}) {
  const { editing } = useSite();
  const t = useT();
  if (!value && !editing) return null;
  const Tag = as ?? "span";
  const marks = editing
    ? { "data-vx-field": `${block.id}|${path}`, ...(multiline ? { "data-vx-multiline": "" } : {}) }
    : {};
  return (
    <Tag className={cn(className, !value && "opacity-45")} {...marks}>
      {value || placeholder || t("template.field.empty")}
    </Tag>
  );
}

export function BlockShell({
  variant,
  className,
  inner,
  children,
}: {
  variant: BlockVariant;
  className?: string;
  inner?: string;
  children: ReactNode;
}) {
  if (variant === "panel") return <div className={className}>{children}</div>;
  return (
    <section className={cn("border-b border-border", className)}>
      <div className={cn("mx-auto max-w-[1240px] px-5 py-16 sm:px-8 lg:py-24", inner)}>{children}</div>
    </section>
  );
}

export function BlockLink({
  href,
  newTab,
  className,
  children,
}: {
  href: string;
  newTab?: boolean;
  className?: string;
  children: ReactNode;
}) {
  const external = newTab || isExternalUrl(href);
  return (
    <Link
      href={href || "#"}
      {...(external ? { target: "_blank", rel: "noopener noreferrer" } : {})}
      className={className}
    >
      {children}
    </Link>
  );
}

export function BlockButton({
  block,
  path,
  label,
  href,
  variant,
  primary,
  onDark,
}: {
  block: SiteBlock;
  path: string;
  label: string;
  href: string;
  variant: BlockVariant;
  primary?: boolean;
  onDark?: boolean;
}) {
  const { editing } = useSite();
  if (!label && !editing) return null;
  if (!href && !editing) return null;
  const className =
    variant === "site"
      ? cn(
          "group inline-flex items-center gap-2 rounded-full text-[15px] font-semibold transition-transform active:scale-[0.97]",
          primary
            ? onDark
              ? "bg-background py-2.5 pr-2.5 pl-5 text-foreground"
              : "bg-primary py-2.5 pr-2.5 pl-5 text-primary-foreground"
            : onDark
              ? "border border-current/30 px-5 py-2.5"
              : "border border-border px-5 py-2.5 hover:border-foreground"
        )
      : cn(
          "inline-flex h-9 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors",
          primary
            ? onDark
              ? "bg-background text-foreground hover:bg-background/90"
              : "bg-primary text-primary-foreground hover:bg-primary/90"
            : "border border-border bg-background hover:bg-accent"
        );
  const content = (
    <>
      <Field block={block} path={path} value={label} />
      {variant === "site" && primary ? (
        <span className="flex size-7 items-center justify-center rounded-full bg-current/15 transition-transform duration-300 group-hover:translate-x-0.5">
          <ArrowRight className="size-3.5" />
        </span>
      ) : null}
    </>
  );
  if (editing) return <span className={className}>{content}</span>;
  return (
    <BlockLink href={href} className={className}>
      {content}
    </BlockLink>
  );
}
