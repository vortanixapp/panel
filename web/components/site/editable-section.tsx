"use client";

import type { ReactElement } from "react";
import { Slot } from "@radix-ui/react-slot";
import { useSite } from "@/context/site-provider";
import { usePageKey, useSitePage } from "@/hooks/use-site";

export function useSectionHidden(id: string): boolean {
  const page = useSitePage();
  const { editing } = useSite();
  return !editing && (page?.hidden?.includes(id) ?? false);
}

export function EditableSection({ id, children }: { id: string; children: ReactElement }) {
  const key = usePageKey();
  const page = useSitePage(key);
  const { editing } = useSite();
  const hidden = page?.hidden?.includes(id) ?? false;
  if (hidden && !editing) return null;
  if (!editing) return children;
  return (
    <Slot
      data-vx-section={id}
      data-vx-page={key}
      data-vx-hidden={hidden ? "" : undefined}
      className={hidden ? "opacity-35 grayscale" : undefined}
    >
      {children}
    </Slot>
  );
}
