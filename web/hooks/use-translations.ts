"use client";

import { useSyncExternalStore } from "react";

import {
  getTranslationOverrides,
  getTranslationsVersion,
  subscribeToTranslations,
  t,
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
  useSyncExternalStore(
    subscribeToTranslations,
    getTranslationsVersion,
    serverVersion
  );
  return t;
}
