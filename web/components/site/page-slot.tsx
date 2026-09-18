"use client";

import { SiteBlockList } from "@/components/site/block-renderer";
import type { BlockVariant } from "@/components/site/blocks/shared";
import { useSite } from "@/context/site-provider";
import { usePageKey, useSitePage } from "@/hooks/use-site";
import { cn } from "@/lib/utils";

export function PageSlot({
  position,
  variant = "panel",
  className,
}: {
  position: "top" | "bottom";
  variant?: BlockVariant;
  className?: string;
}) {
  const key = usePageKey();
  const page = useSitePage(key);
  const { editing } = useSite();
  const blocks = (position === "top" ? page?.top : page?.bottom) ?? [];
  const visible = blocks.filter((block) => !block.hidden);
  if (!editing && visible.length === 0) return null;
  return (
    <SiteBlockList
      blocks={blocks}
      variant={variant}
      page={key}
      zone={position}
      className={cn("flex flex-col gap-4", position === "top" ? "mb-6" : "mt-6", className)}
    />
  );
}
