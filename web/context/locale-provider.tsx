"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";

import { setCookie } from "@/lib/cookies";
import {
  browserLocale,
  DEFAULT_LOCALE,
  LOCALE_COOKIE_MAX_AGE,
  LOCALE_COOKIE_NAME,
  normalizeLocale,
  setLocale as setModuleLocale,
  type Locale,
} from "@/lib/i18n";
import {
  ACCOUNT_PREFS_EVENT,
  getAccountPreferences,
} from "@/lib/user-preferences";

type LocaleContextValue = {
  locale: Locale;
  setLocale: (next: string | null | undefined) => void;
};

const LocaleContext = createContext<LocaleContextValue>({
  locale: DEFAULT_LOCALE,
  setLocale: () => undefined,
});

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext);
}

export function LocaleProvider({
  initialLocale,
  hasCookie,
  children,
}: {
  initialLocale: Locale;
  hasCookie: boolean;
  children: ReactNode;
}) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale);

  useEffect(() => {
    setModuleLocale(locale);
  }, [locale]);

  const setLocale = useCallback((next: string | null | undefined) => {
    const normalized = normalizeLocale(next);
    setCookie(LOCALE_COOKIE_NAME, normalized, LOCALE_COOKIE_MAX_AGE);
    setLocaleState(normalized);
  }, []);

  useEffect(() => {
    if (hasCookie) return;
    setLocale(getAccountPreferences().language || browserLocale());
  }, [hasCookie, setLocale]);

  useEffect(() => {
    const apply = () => {
      const stored = getAccountPreferences().language;
      if (stored) setLocale(stored);
    };
    window.addEventListener(ACCOUNT_PREFS_EVENT, apply);
    return () => window.removeEventListener(ACCOUNT_PREFS_EVENT, apply);
  }, [setLocale]);

  return (
    <LocaleContext.Provider value={{ locale, setLocale }}>
      {children}
    </LocaleContext.Provider>
  );
}
