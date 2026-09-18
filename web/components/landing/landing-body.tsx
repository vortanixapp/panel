"use client";

import { useMemo } from "react";
import { SiteBlockList } from "@/components/site/block-renderer";
import { useSite } from "@/context/site-provider";
import { defaultLandingBlocks } from "@/lib/site/defaults";

export function LandingBody() {
  const { document } = useSite();
  const configured = document.pages?.["/"]?.blocks;
  const blocks = useMemo(() => configured ?? defaultLandingBlocks(), [configured]);

  return (
    <>
      <noscript>
        <style>{"[data-reveal]{opacity:1!important;transform:none!important;filter:none!important}"}</style>
      </noscript>
      <SiteBlockList blocks={blocks} variant="site" page="/" zone="blocks" />
    </>
  );
}
