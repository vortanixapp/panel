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

// На сервере версия всегда нулевая: язык берётся из localStorage уже после
// монтирования, иначе разметка сервера и клиента разъезжались бы при гидрации.
function serverVersion(): number {
  return 0;
}

/**
 * Возвращает t(), перерисовывая компонент при смене языка или приходе
 * переопределений с сервера. Сама функция стабильна — перерисовку вызывает
 * изменившийся номер версии.
 */
export function useT(): TranslateFn {
  useSyncExternalStore(
    subscribeToTranslations,
    getTranslationsVersion,
    serverVersion
  );
  return t;
}
