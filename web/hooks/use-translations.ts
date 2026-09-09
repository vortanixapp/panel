"use client";

import { useMemo, useSyncExternalStore } from "react";

import { useLocale } from "@/context/locale-provider";
import {
  getTranslationOverrides,
  getTranslationsVersion,
  localeTagOf,
  subscribeToTranslations,
  translateWith,
  type TranslateFn,
} from "@/lib/i18n";

export function useTranslations(): Record<string, string> {
  return useSyncExternalStore(
    subscribeToTranslations,
    getTranslationOverrides,
    () => ({})
  );
}

function serverVersion(): number {
  return 0;
}

export function useT(): TranslateFn {
  const { locale } = useLocale();
  const version = useSyncExternalStore(
    subscribeToTranslations,
    getTranslationsVersion,
    serverVersion
  );
  return useMemo(() => translateWith(locale), [locale, version]);
}

export function useLocaleTag(): string {
  const { locale } = useLocale();
  return localeTagOf(locale);
}
