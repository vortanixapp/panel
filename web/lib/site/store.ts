import type { SiteDocument, Viewer } from "@/lib/site/types";

export type EditorMode = "edit" | "preview";

export type SiteOverride = {
  document: SiteDocument;
  mode: EditorMode;
  viewer: Viewer | null;
  selected: string | null;
};

let current: SiteOverride | null = null;
const listeners = new Set<() => void>();

export function getSiteOverride(): SiteOverride | null {
  return current;
}

export function setSiteOverride(next: SiteOverride | null) {
  current = next;
  listeners.forEach((fn) => fn());
}

export function subscribeSiteOverride(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}
