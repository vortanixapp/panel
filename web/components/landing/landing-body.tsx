"use client";

import { useEffect, useState } from "react";
import { CtaSection } from "@/components/landing/sections/cta";
import { FaqSection } from "@/components/landing/sections/faq";
import { GamesSection } from "@/components/landing/sections/games";
import { HardwareSection } from "@/components/landing/sections/hardware";
import { HeroSection } from "@/components/landing/sections/hero";
import { LocationsSection } from "@/components/landing/sections/locations";
import { PanelSection } from "@/components/landing/sections/panel";
import { PricingSection } from "@/components/landing/sections/pricing";
import { StepsSection } from "@/components/landing/sections/steps";
import { useBrand } from "@/context/brand-provider";
import { getAccessToken } from "@/lib/api";

const ORDER = ["hero", "games", "steps", "hardware", "panel", "locations", "pricing", "faq", "cta"] as const;

type Block = (typeof ORDER)[number];

export function LandingBody() {
  const { templateBlocks: blocks } = useBrand();
  const [loggedIn, setLoggedIn] = useState(false);

  useEffect(() => {
    setLoggedIn(!!getAccessToken());
  }, []);

  const visible = ORDER.filter((name) => blocks[name] !== false);
  const numbered: Block[] = visible.filter((name) => name !== "hero" && name !== "cta");
  const indexOf = (name: Block) => String(numbered.indexOf(name) + 1).padStart(2, "0");

  const render = (name: Block) => {
    switch (name) {
      case "hero":
        return <HeroSection key={name} showPricingLink={blocks.pricing !== false} />;
      case "games":
        return <GamesSection key={name} index={indexOf(name)} />;
      case "steps":
        return <StepsSection key={name} index={indexOf(name)} />;
      case "hardware":
        return <HardwareSection key={name} index={indexOf(name)} />;
      case "panel":
        return <PanelSection key={name} index={indexOf(name)} />;
      case "locations":
        return <LocationsSection key={name} index={indexOf(name)} />;
      case "pricing":
        return <PricingSection key={name} index={indexOf(name)} />;
      case "faq":
        return <FaqSection key={name} index={indexOf(name)} />;
      case "cta":
        return <CtaSection key={name} loggedIn={loggedIn} />;
    }
  };

  return (
    <>
      <noscript>
        <style>{"[data-reveal]{opacity:1!important;transform:none!important;filter:none!important}"}</style>
      </noscript>
      {visible.map(render)}
    </>
  );
}
