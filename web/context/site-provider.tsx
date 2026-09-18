"use client";

import {
  createContext,
  useContext,
  useMemo,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import {
  getSiteOverride,
  subscribeSiteOverride,
  type EditorMode,
} from "@/lib/site/store";
import type { SiteDocument, Viewer } from "@/lib/site/types";

type SiteContextValue = {
  document: SiteDocument;
  session: Viewer;
  inEditor: boolean;
  editing: boolean;
  mode: EditorMode;
  selected: string | null;
  viewer: Viewer | null;
};

const EMPTY: SiteDocument = {};
const GUEST: Viewer = { loggedIn: false, role: "" };

const SiteContext = createContext<SiteContextValue>({
  document: EMPTY,
  session: GUEST,
  inEditor: false,
  editing: false,
  mode: "preview",
  selected: null,
  viewer: null,
});

function serverOverride() {
  return null;
}

export function SiteProvider({
  initial,
  session,
  children,
}: {
  initial: SiteDocument;
  session: Viewer;
  children: ReactNode;
}) {
  const override = useSyncExternalStore(subscribeSiteOverride, getSiteOverride, serverOverride);

  const value = useMemo<SiteContextValue>(
    () => ({
      document: override?.document ?? initial,
      session,
      inEditor: override !== null,
      editing: override?.mode === "edit",
      mode: override?.mode ?? "preview",
      selected: override?.selected ?? null,
      viewer: override?.viewer ?? null,
    }),
    [override, initial, session]
  );

  return <SiteContext.Provider value={value}>{children}</SiteContext.Provider>;
}

export function useSite(): SiteContextValue {
  return useContext(SiteContext);
}
