"use client";

import { useEffect, type ReactNode } from "react";

import { fetchTranslations, getAccessToken } from "@/lib/api";
import { setTranslationOverrides } from "@/lib/i18n";

export function TranslationsProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    if (!getAccessToken()) return;
    let cancelled = false;
    void fetchTranslations()
      .then((res) => {
        if (!cancelled) setTranslationOverrides(res.messages ?? {});
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  return <>{children}</>;
}
