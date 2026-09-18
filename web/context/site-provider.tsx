"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { fetchSite } from "@/lib/site/api";
import {
  getSiteOverride,
  subscribeSiteOverride,
  type EditorMode,
} from "@/lib/site/store";
import type { SiteDocument, Viewer } from "@/lib/site/types";

type SiteContextValue = {
  document: SiteDocument;
  published: SiteDocument;
  refreshed: boolean;
  inEditor: boolean;
  editing: boolean;
  mode: EditorMode;
  selected: string | null;
  viewer: Viewer | null;
};

const EMPTY: SiteDocument = {};

const SiteContext = createContext<SiteContextValue>({
  document: EMPTY,
  published: EMPTY,
  refreshed: false,
  inEditor: false,
  editing: false,
  mode: "preview",
  selected: null,
  viewer: null,
});

function serverOverride() {
  return null;
}

export function SiteProvider({ initial, children }: { initial: SiteDocument; children: ReactNode }) {
  const [published, setPublished] = useState<SiteDocument>(initial);
  const [refreshed, setRefreshed] = useState(false);
  const override = useSyncExternalStore(subscribeSiteOverride, getSiteOverride, serverOverride);

  useEffect(() => {
    let cancelled = false;
    fetchSite()
      .then((doc) => {
        if (!cancelled) setPublished(doc);
      })
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setRefreshed(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<SiteContextValue>(
    () => ({
      document: override?.document ?? published,
      published,
      refreshed: refreshed || override !== null,
      inEditor: override !== null,
      editing: override?.mode === "edit",
      mode: override?.mode ?? "preview",
      selected: override?.selected ?? null,
      viewer: override?.viewer ?? null,
    }),
    [override, published, refreshed]
  );

  return <SiteContext.Provider value={value}>{children}</SiteContext.Provider>;
}

export function useSite(): SiteContextValue {
  return useContext(SiteContext);
}
