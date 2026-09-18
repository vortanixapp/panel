"use client";

import { Fragment, useMemo, type ReactNode } from "react";
import { EyeOff, Plus } from "lucide-react";
import { CtaSection } from "@/components/landing/sections/cta";
import { FaqSection } from "@/components/landing/sections/faq";
import { GamesSection } from "@/components/landing/sections/games";
import { HardwareSection } from "@/components/landing/sections/hardware";
import { HeroSection } from "@/components/landing/sections/hero";
import { LocationsSection } from "@/components/landing/sections/locations";
import { PanelSection } from "@/components/landing/sections/panel";
import { PricingSection } from "@/components/landing/sections/pricing";
import { StepsSection } from "@/components/landing/sections/steps";
import {
  HeadingBlock,
  HtmlBlock,
  ImageBlock,
  MediaBlock,
  SpacerBlock,
  TextBlock,
  VideoBlock,
} from "@/components/site/blocks/content-blocks";
import {
  CtaBlock,
  FaqBlock,
  FeaturesBlock,
  NoticeBlock,
  StatsBlock,
} from "@/components/site/blocks/list-blocks";
import type { BlockVariant, BlockViewProps } from "@/components/site/blocks/shared";
import { useSite } from "@/context/site-provider";
import { useViewer } from "@/hooks/use-site";
import { useT } from "@/hooks/use-translations";
import { blockMeta, isBuiltinBlock } from "@/lib/site/blocks";
import type { SiteBlock, SiteZone } from "@/lib/site/types";
import { cn } from "@/lib/utils";

const CONTENT: Record<string, (props: BlockViewProps) => ReactNode> = {
  heading: HeadingBlock,
  text: TextBlock,
  image: ImageBlock,
  media: MediaBlock,
  features: FeaturesBlock,
  cta: CtaBlock,
  faq: FaqBlock,
  stats: StatsBlock,
  notice: NoticeBlock,
  video: VideoBlock,
  html: HtmlBlock,
  spacer: SpacerBlock,
};

type BuiltinContext = {
  indexOf: (block: SiteBlock) => string;
  pricingVisible: boolean;
  loggedIn: boolean;
};

function renderBuiltin(block: SiteBlock, ctx: BuiltinContext): ReactNode {
  switch (block.type) {
    case "landing.hero":
      return <HeroSection showPricingLink={ctx.pricingVisible} />;
    case "landing.games":
      return <GamesSection index={ctx.indexOf(block)} />;
    case "landing.steps":
      return <StepsSection index={ctx.indexOf(block)} />;
    case "landing.hardware":
      return <HardwareSection index={ctx.indexOf(block)} />;
    case "landing.panel":
      return <PanelSection index={ctx.indexOf(block)} />;
    case "landing.locations":
      return <LocationsSection index={ctx.indexOf(block)} />;
    case "landing.pricing":
      return <PricingSection index={ctx.indexOf(block)} />;
    case "landing.faq":
      return <FaqSection index={ctx.indexOf(block)} />;
    case "landing.cta":
      return <CtaSection loggedIn={ctx.loggedIn} />;
    default:
      return null;
  }
}

function renderBlock(block: SiteBlock, variant: BlockVariant, ctx: BuiltinContext): ReactNode {
  if (isBuiltinBlock(block.type)) return variant === "site" ? renderBuiltin(block, ctx) : null;
  const View = CONTENT[block.type];
  return View ? <View block={block} variant={variant} /> : null;
}

function EditFrame({
  block,
  page,
  zone,
  selected,
  children,
}: {
  block: SiteBlock;
  page: string;
  zone: SiteZone;
  selected: boolean;
  children: ReactNode;
}) {
  const t = useT();
  const meta = blockMeta(block.type);
  return (
    <div
      data-vx-block={block.id}
      data-vx-type={block.type}
      data-vx-page={page}
      data-vx-zone={zone}
      data-vx-selected={selected ? "" : undefined}
      className="relative"
    >
      {block.hidden ? (
        <div
          data-vx-ui=""
          className="mx-auto my-2 flex max-w-[1240px] items-center gap-2 rounded-lg border border-dashed border-muted-foreground/40 bg-muted/30 px-4 py-3 text-sm text-muted-foreground"
        >
          <EyeOff className="size-4 shrink-0" />
          {t("template.preview.hidden_block", { name: meta ? t(meta.titleKey) : block.type })}
        </div>
      ) : (
        children
      )}
    </div>
  );
}

export function InsertPlaceholder({
  page,
  zone,
  index,
  variant,
}: {
  page: string;
  zone: SiteZone;
  index: number;
  variant: BlockVariant;
}) {
  const t = useT();
  return (
    <div
      data-vx-insert=""
      data-vx-ui=""
      data-vx-page={page}
      data-vx-zone={zone}
      data-vx-index={index}
      className={cn(
        "flex cursor-pointer items-center justify-center gap-2 rounded-xl border border-dashed border-primary/40 bg-primary/[0.03] py-4 text-sm font-medium text-primary/80 transition-colors hover:bg-primary/[0.07]",
        variant === "site" && "mx-auto my-6 w-[calc(100%-2.5rem)] max-w-[1200px]"
      )}
    >
      <Plus className="size-4" />
      {t("template.preview.add_block")}
    </div>
  );
}

export function SiteBlockList({
  blocks,
  variant,
  page,
  zone,
  className,
}: {
  blocks: SiteBlock[];
  variant: BlockVariant;
  page: string;
  zone: SiteZone;
  className?: string;
}) {
  const { editing, selected } = useSite();
  const { loggedIn } = useViewer();

  const ctx = useMemo<BuiltinContext>(() => {
    const visible = blocks.filter((block) => !block.hidden && isBuiltinBlock(block.type));
    const numbered = visible.filter((block) => block.type !== "landing.hero" && block.type !== "landing.cta");
    return {
      indexOf: (block) => String(numbered.indexOf(block) + 1).padStart(2, "0"),
      pricingVisible: visible.some((block) => block.type === "landing.pricing"),
      loggedIn,
    };
  }, [blocks, loggedIn]);

  if (editing && blocks.length === 0) {
    return (
      <div className={className}>
        <InsertPlaceholder page={page} zone={zone} index={0} variant={variant} />
      </div>
    );
  }

  const items = blocks.map((block) => {
    if (block.hidden && !editing) return null;
    const content = renderBlock(block, variant, ctx);
    if (!editing) return <Fragment key={block.id}>{content}</Fragment>;
    return (
      <EditFrame key={block.id} block={block} page={page} zone={zone} selected={selected === block.id}>
        {content}
      </EditFrame>
    );
  });

  return className ? <div className={className}>{items}</div> : <>{items}</>;
}
