"use client";

import Link from "next/link";
import type { ComponentPropsWithRef, MouseEvent, ReactNode, Ref } from "react";
import { m } from "motion/react";
import { ArrowRight, ArrowUpRight, ChevronDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { landingFontVariables } from "@/components/landing/fonts";
import { EASE_OUT } from "@/components/landing/motion";
import { isExternalUrl, type ResolvedItem } from "@/lib/site/menu";
import { siteIcon } from "@/lib/site/icons";
import { cn } from "@/lib/utils";

export type NavigateFn = (url: string) => void;

type SiteLinkProps = Omit<ComponentPropsWithRef<"a">, "href" | "children"> & {
  item: ResolvedItem;
  onNavigate: NavigateFn;
  children: ReactNode;
};

export function SiteLink({ item, onNavigate, children, onClick, ref, ...rest }: SiteLinkProps) {
  const handleClick = (event: MouseEvent<HTMLElement>) => {
    onClick?.(event as MouseEvent<HTMLAnchorElement>);
    if (!event.defaultPrevented) onNavigate(item.url);
  };
  if (item.url.startsWith("#")) {
    return (
      <button
        {...(rest as ComponentPropsWithRef<"button">)}
        ref={ref as Ref<HTMLButtonElement>}
        type="button"
        onClick={handleClick}
      >
        {children}
      </button>
    );
  }
  const external = item.newTab || isExternalUrl(item.url);
  return (
    <Link
      {...rest}
      ref={ref}
      href={item.url}
      {...(external ? { target: "_blank", rel: "noopener noreferrer" } : {})}
      onClick={handleClick}
    >
      {children}
    </Link>
  );
}

function flatten(items: ResolvedItem[]): ResolvedItem[] {
  return items.flatMap((item) => (item.url ? [item, ...flatten(item.items)] : flatten(item.items)));
}

function ItemBadge({ badge }: { badge?: string }) {
  if (!badge) return null;
  return (
    <span className="relative ml-1.5 rounded-full bg-primary/15 px-1.5 py-px text-[10.5px] font-semibold text-primary">
      {badge}
    </span>
  );
}

export function SiteHeaderNav({
  items,
  pathname,
  hovered,
  onHover,
  onNavigate,
}: {
  items: ResolvedItem[];
  pathname: string;
  hovered: string | null;
  onHover: (id: string | null) => void;
  onNavigate: NavigateFn;
}) {
  return (
    <nav className="hidden flex-1 items-center gap-0.5 lg:flex" onMouseLeave={() => onHover(null)}>
      {items.map((item) => {
        const active = !item.url.startsWith("#") && pathname === item.url;
        const className = cn(
          "relative rounded-full px-3.5 py-2 text-[14px] font-medium whitespace-nowrap transition-colors",
          active || hovered === item.id ? "text-foreground" : "text-muted-foreground"
        );
        const pill =
          hovered === item.id ? (
            <m.span
              layoutId="landing-nav-hover"
              className="absolute inset-0 rounded-full bg-accent"
              transition={{ type: "spring", stiffness: 500, damping: 38 }}
            />
          ) : null;
        if (item.items.length > 0) {
          const children = flatten(item.items);
          return (
            <DropdownMenu key={item.id} modal={false}>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  onMouseEnter={() => onHover(item.id)}
                  onFocus={() => onHover(item.id)}
                  className={cn(className, "group/nav")}
                >
                  {pill}
                  <span className="relative inline-flex items-center gap-1">
                    {item.title}
                    <ItemBadge badge={item.badge} />
                    <ChevronDown className="size-3.5 opacity-60 transition-transform duration-300 group-data-[state=open]/nav:rotate-180" />
                  </span>
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="start"
                sideOffset={10}
                className={cn(landingFontVariables, "font-landing min-w-[240px] rounded-2xl p-1.5")}
              >
                {children.map((child) => {
                  const Icon = siteIcon(child.icon);
                  return (
                    <DropdownMenuItem key={child.id} asChild className="rounded-xl px-3 py-2.5">
                      <SiteLink item={child} onNavigate={onNavigate} className="flex w-full items-center gap-2.5 text-[14px]">
                        {Icon ? <Icon className="size-4 text-muted-foreground" /> : null}
                        <span className="flex-1 text-left">{child.title}</span>
                        <ItemBadge badge={child.badge} />
                      </SiteLink>
                    </DropdownMenuItem>
                  );
                })}
              </DropdownMenuContent>
            </DropdownMenu>
          );
        }
        return (
          <SiteLink
            key={item.id}
            item={item}
            onNavigate={onNavigate}
            onMouseEnter={() => onHover(item.id)}
            onFocus={() => onHover(item.id)}
            className={className}
          >
            {pill}
            <span className="relative">{item.title}</span>
            <ItemBadge badge={item.badge} />
          </SiteLink>
        );
      })}
    </nav>
  );
}

const MOBILE_ITEM = {
  hidden: { opacity: 0, y: 18 },
  shown: { opacity: 1, y: 0, transition: { duration: 0.5, ease: EASE_OUT } },
};

export function SiteMobileNav({ items, onNavigate }: { items: ResolvedItem[]; onNavigate: NavigateFn }) {
  const rowClass =
    "vx-display flex w-full items-center justify-between border-b border-border py-4 text-left text-[1.7rem] font-semibold tracking-[-0.03em]";
  return (
    <>
      {items.map((item) => (
        <m.div key={item.id} variants={MOBILE_ITEM}>
          {item.items.length > 0 ? (
            <div className="border-b border-border py-4">
              <div className="vx-display text-[1.7rem] font-semibold tracking-[-0.03em]">
                {item.title}
                <ItemBadge badge={item.badge} />
              </div>
              <div className="mt-3 flex flex-col gap-1">
                {flatten(item.items).map((child) => (
                  <SiteLink
                    key={child.id}
                    item={child}
                    onNavigate={onNavigate}
                    className="flex items-center justify-between py-1.5 text-left text-[17px] text-muted-foreground"
                  >
                    {child.title}
                    <ArrowRight className="size-4" />
                  </SiteLink>
                ))}
              </div>
            </div>
          ) : (
            <SiteLink item={item} onNavigate={onNavigate} className={rowClass}>
              <span>
                {item.title}
                <ItemBadge badge={item.badge} />
              </span>
              <ArrowRight className="size-5 text-muted-foreground" />
            </SiteLink>
          )}
        </m.div>
      ))}
    </>
  );
}

export function SiteFooterColumns({ items, onNavigate }: { items: ResolvedItem[]; onNavigate: NavigateFn }) {
  const columns: { id: string; title: string; links: ResolvedItem[] }[] = [];
  const loose: ResolvedItem[] = [];
  for (const item of items) {
    if (item.kind === "group" || (!item.url && item.items.length > 0)) {
      const links = flatten(item.items);
      if (links.length > 0) columns.push({ id: item.id, title: item.title, links });
    } else {
      loose.push(item, ...flatten(item.items));
    }
  }
  if (loose.length > 0) columns.push({ id: "loose", title: "", links: loose });

  return (
    <div className="grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-4">
      {columns.map((col) => (
        <div key={col.id}>
          {col.title ? <div className="font-mono text-[12px] text-muted-foreground">{col.title}</div> : null}
          <ul className={cn("flex flex-col gap-2.5", col.title && "mt-4")}>
            {col.links.map((link) => (
              <li key={link.id}>
                <SiteLink
                  item={link}
                  onNavigate={onNavigate}
                  className="group inline-flex items-center gap-1 text-left text-[14.5px] text-foreground/80 transition-colors hover:text-foreground"
                >
                  {link.title}
                  <ItemBadge badge={link.badge} />
                  {link.url.startsWith("#") ? null : (
                    <ArrowUpRight className="size-3.5 -translate-x-1 opacity-0 transition-[opacity,transform] duration-300 group-hover:translate-x-0 group-hover:opacity-100" />
                  )}
                </SiteLink>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
