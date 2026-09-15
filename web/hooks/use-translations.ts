"use client";

import { useMemo, useSyncExternalStore } from "react";

import { useLocale } from "@/context/locale-provider";
import {
  getTranslationMessages,
  localeTagOf,
  subscribeToTranslations,
  translator,
  type TranslateFn,
} from "@/lib/i18n";

const EMPTY_MESSAGES: Record<string, string> = {};

function serverMessages(): Record<string, string> {
  return EMPTY_MESSAGES;
}

export function useTranslations(): Record<string, string> {
  return useSyncExternalStore(
    subscribeToTranslations,
    getTranslationMessages,
    serverMessages
  );
}

export function useT(): TranslateFn {
  const { base, messages } = useLocale();
  return useMemo(() => translator({ base, messages }), [base, messages]);
}

export function useLocaleTag(): string {
  const { locale, base } = useLocale();
  return localeTagOf(locale, base);
}
