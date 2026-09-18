"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";

import { fetchI18n } from "@/lib/api";
import { setCookie } from "@/lib/cookies";
import {
  browserLocale,
  fallbackI18n,
  getLocaleOverride,
  i18nState,
  LOCALE_COOKIE_MAX_AGE,
  LOCALE_COOKIE_NAME,
  normalizeLocaleCode,
  parseI18nPayload,
  resolveLocale,
  setI18nState,
  subscribeLocaleOverride,
  type I18nPayload,
  type I18nState,
} from "@/lib/i18n";
import {
  ACCOUNT_PREFS_EVENT,
  getAccountPreferences,
} from "@/lib/user-preferences";

type LocaleContextValue = I18nState & {
  setLocale: (next: string | null | undefined) => void;
  reload: () => Promise<void>;
};

const LocaleContext = createContext<LocaleContextValue>({
  ...i18nState(fallbackI18n()),
  setLocale: () => undefined,
  reload: async () => undefined,
});

function noOverride(): I18nState | null {
  return null;
}

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext);
}

export function LocaleProvider({
  initial,
  hasCookie,
  children,
}: {
  initial: I18nPayload;
  hasCookie: boolean;
  children: ReactNode;
}) {
  const [state, setState] = useState<I18nState>(() => i18nState(initial));
  const stateRef = useRef(state);
  const requestRef = useRef(0);

  useEffect(() => {
    setI18nState(stateRef.current);
  }, []);

  const apply = useCallback((next: I18nState) => {
    stateRef.current = next;
    setI18nState(next);
    setState(next);
    setCookie(LOCALE_COOKIE_NAME, next.locale, LOCALE_COOKIE_MAX_AGE);
    document.documentElement.lang = next.locale;
  }, []);

  const load = useCallback(
    async (requested: string) => {
      const request = ++requestRef.current;
      let payload: I18nPayload;
      try {
        payload = parseI18nPayload(await fetchI18n(requested), requested);
      } catch {
        const current = stateRef.current;
        const locale = resolveLocale(
          requested,
          current.languages,
          current.defaultLocale
        );
        payload = {
          default_locale: current.defaultLocale,
          languages: current.languages,
          locale,
          base:
            current.languages.find((l) => l.code === locale)?.base ??
            current.base,
          messages: locale === current.locale ? current.messages : {},
        };
      }
      if (request !== requestRef.current) return;
      apply(i18nState(payload));
    },
    [apply]
  );

  const setLocale = useCallback(
    (next: string | null | undefined) => {
      const code = normalizeLocaleCode(next);
      if (!code) return;
      if (code === stateRef.current.locale) {
        requestRef.current += 1;
        setCookie(LOCALE_COOKIE_NAME, code, LOCALE_COOKIE_MAX_AGE);
        return;
      }
      void load(code);
    },
    [load]
  );

  const reload = useCallback(() => load(stateRef.current.locale), [load]);

  useEffect(() => {
    if (hasCookie) return;
    const current = stateRef.current;
    setLocale(
      getAccountPreferences().language ||
        browserLocale(current.languages, current.defaultLocale)
    );
  }, [hasCookie, setLocale]);

  useEffect(() => {
    const onPrefs = () => {
      const stored = getAccountPreferences().language;
      if (stored) setLocale(stored);
    };
    window.addEventListener(ACCOUNT_PREFS_EVENT, onPrefs);
    return () => window.removeEventListener(ACCOUNT_PREFS_EVENT, onPrefs);
  }, [setLocale]);

  const override = useSyncExternalStore(subscribeLocaleOverride, getLocaleOverride, noOverride);

  useEffect(() => {
    setI18nState(override ?? stateRef.current);
  }, [override]);

  const value = useMemo(
    () => ({ ...(override ?? state), setLocale, reload }),
    [override, state, setLocale, reload]
  );

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  );
}
