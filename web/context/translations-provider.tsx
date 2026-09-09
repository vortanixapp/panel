"use client";

import { useEffect, type ReactNode } from "react";

import { fetchTranslations, getAccessToken } from "@/lib/api";
import { browserLocale, setLocale, setTranslationOverrides } from "@/lib/i18n";
import {
  ACCOUNT_PREFS_EVENT,
  getAccountPreferences,
} from "@/lib/user-preferences";

export function TranslationsProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    const applyLocale = () =>
      setLocale(getAccountPreferences().language || browserLocale());
    applyLocale();
    window.addEventListener(ACCOUNT_PREFS_EVENT, applyLocale);
    return () => window.removeEventListener(ACCOUNT_PREFS_EVENT, applyLocale);
  }, []);

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
