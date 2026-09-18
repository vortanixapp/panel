import { EN } from "@/lib/locales/en";
import { RU } from "@/lib/locales/ru";

export type Locale = string;
export type BaseLocale = "ru" | "en";

export type LanguageInfo = {
  code: string;
  name: string;
  base: BaseLocale;
};

export type I18nPayload = {
  default_locale: string;
  languages: LanguageInfo[];
  locale: string;
  base: BaseLocale;
  messages: Record<string, string>;
};

export type I18nState = {
  locale: string;
  base: BaseLocale;
  messages: Record<string, string>;
  languages: LanguageInfo[];
  defaultLocale: string;
};

export const DEFAULT_LOCALE: BaseLocale = "ru";
export const BASE_LOCALES: BaseLocale[] = ["ru", "en"];
export const LOCALE_COOKIE_NAME = "vortanix-locale";
export const LOCALE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;
export const BUILTIN_LANGUAGES: LanguageInfo[] = [
  { code: "ru", name: "Русский", base: "ru" },
  { code: "en", name: "English", base: "en" },
];

export const CATALOGS: Record<BaseLocale, Record<string, string>> = {
  ru: RU,
  en: EN,
};
export const CATALOG_KEYS: string[] = Object.keys(RU).sort();

const EN_COVERS_RU: Record<keyof typeof RU, string> = EN;
const RU_COVERS_EN: Record<keyof typeof EN, string> = RU;
void EN_COVERS_RU;
void RU_COVERS_EN;

const LOCALE_CODE = /^[a-z]{2,3}(-[a-z0-9]{2,8})?$/;

export function isBaseLocale(value: unknown): value is BaseLocale {
  return value === "ru" || value === "en";
}

export function normalizeLocaleCode(value: string | null | undefined): string {
  return (value ?? "").trim().toLowerCase().replace(/_/g, "-");
}

export function isLocaleCode(value: string): boolean {
  return LOCALE_CODE.test(value);
}

export function catalogText(base: BaseLocale, key: string): string | undefined {
  return CATALOGS[base][key] ?? CATALOGS[DEFAULT_LOCALE][key];
}

export function resolveLocale(
  value: string | null | undefined,
  languages: LanguageInfo[],
  fallback: string
): string {
  const code = normalizeLocaleCode(value);
  if (languages.some((l) => l.code === code)) return code;
  if (languages.some((l) => l.code === fallback)) return fallback;
  return languages[0]?.code ?? DEFAULT_LOCALE;
}

export function fallbackI18n(requested?: string | null): I18nPayload {
  const code = normalizeLocaleCode(requested);
  const locale = isBaseLocale(code) ? code : DEFAULT_LOCALE;
  return {
    default_locale: DEFAULT_LOCALE,
    languages: BUILTIN_LANGUAGES,
    locale,
    base: locale,
    messages: {},
  };
}

export function parseI18nPayload(
  raw: unknown,
  requested?: string | null
): I18nPayload {
  const fallback = fallbackI18n(requested);
  if (!raw || typeof raw !== "object") return fallback;
  const data = raw as Record<string, unknown>;

  const languages = Array.isArray(data.languages)
    ? data.languages.flatMap((item): LanguageInfo[] => {
        if (!item || typeof item !== "object") return [];
        const { code, name, base } = item as Record<string, unknown>;
        if (typeof code !== "string" || !isLocaleCode(code)) return [];
        if (!isBaseLocale(base)) return [];
        const label = typeof name === "string" && name.trim() ? name : code;
        return [{ code, name: label, base }];
      })
    : [];
  if (languages.length === 0) return fallback;

  const messages: Record<string, string> = {};
  if (data.messages && typeof data.messages === "object") {
    for (const [key, value] of Object.entries(
      data.messages as Record<string, unknown>
    )) {
      if (typeof value === "string" && value.trim() !== "") {
        messages[key] = value;
      }
    }
  }

  const defaultLocale = resolveLocale(
    String(data.default_locale ?? ""),
    languages,
    languages[0].code
  );
  const locale = resolveLocale(String(data.locale ?? ""), languages, defaultLocale);
  const base = languages.find((l) => l.code === locale)?.base ?? DEFAULT_LOCALE;
  return { default_locale: defaultLocale, languages, locale, base, messages };
}

export function i18nState(payload: I18nPayload): I18nState {
  return {
    locale: payload.locale,
    base: payload.base,
    messages: payload.messages,
    languages: payload.languages,
    defaultLocale: payload.default_locale,
  };
}

let state: I18nState = i18nState(fallbackI18n());
let version = 0;
const listeners = new Set<() => void>();

export function setI18nState(next: I18nState) {
  if (typeof window === "undefined" || next === state) return;
  state = next;
  version += 1;
  listeners.forEach((fn) => fn());
}

export function subscribeToTranslations(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function getTranslationsVersion(): number {
  return version;
}

export function getTranslationMessages(): Record<string, string> {
  return state.messages;
}

export function getLocale(): string {
  return state.locale;
}

export function getBaseLocale(): BaseLocale {
  return state.base;
}

export function pickLocale(
  candidates: readonly string[],
  languages: LanguageInfo[],
  fallback: string
): string {
  const codes = new Set(languages.map((l) => l.code));
  for (const candidate of candidates) {
    const tag = normalizeLocaleCode(candidate);
    if (codes.has(tag)) return tag;
    const primary = tag.split("-")[0];
    if (codes.has(primary)) return primary;
  }
  return fallback;
}

export function browserLocale(
  languages: LanguageInfo[] = state.languages,
  fallback: string = state.defaultLocale
): string {
  if (typeof navigator === "undefined") return fallback;
  return pickLocale(navigator.languages ?? [navigator.language], languages, fallback);
}

export function acceptedLocales(header: string | null | undefined): string[] {
  return (header ?? "")
    .split(",")
    .map((part, index) => {
      const [tag, ...params] = part.split(";").map((item) => item.trim());
      const weight = params.find((item) => item.startsWith("q="));
      return { tag, q: weight ? Number(weight.slice(2)) : 1, index };
    })
    .filter((item) => item.tag && item.tag !== "*" && item.q > 0)
    .sort((a, b) => b.q - a.q || a.index - b.index)
    .map((item) => item.tag);
}

const LOCALE_TAGS: Record<string, string> = {
  ru: "ru-RU",
  en: "en-US",
};

export function localeTagOf(code: string, base: BaseLocale = DEFAULT_LOCALE): string {
  if (LOCALE_TAGS[code]) return LOCALE_TAGS[code];
  try {
    return Intl.getCanonicalLocales(code)[0] ?? LOCALE_TAGS[base];
  } catch {
    return LOCALE_TAGS[base];
  }
}

export function localeTag(): string {
  return localeTagOf(state.locale, state.base);
}

export function dateLocaleTag(): string {
  if (state.locale === "en") return "en-GB";
  if (state.locale === "ru") return "ru";
  return localeTag();
}

export function builtinLanguageName(base: BaseLocale): string {
  return BUILTIN_LANGUAGES.find((l) => l.code === base)?.name ?? base;
}

export type TranslateParams = Record<string, string | number>;
export type TranslateFn = (key: string, params?: TranslateParams) => string;

function interpolate(template: string, params: TranslateParams): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined || value === null ? match : String(value);
  });
}

type MarkerEncoder = (key: string, text: string) => string;

let markerEncoder: MarkerEncoder | null = null;

export function setMarkerEncoder(encoder: MarkerEncoder | null) {
  markerEncoder = encoder;
}

export function rawMessage(
  source: { base: BaseLocale; messages: Record<string, string> },
  key: string
): string | undefined {
  const custom = source.messages[key];
  return typeof custom === "string" && custom.trim() !== ""
    ? custom
    : catalogText(source.base, key);
}

export function translator(source: {
  base: BaseLocale;
  messages: Record<string, string>;
}): TranslateFn {
  return (key: string, params?: TranslateParams): string => {
    const value = rawMessage(source, key);
    if (value === undefined) return key;
    const text = params ? interpolate(value, params) : value;
    return markerEncoder ? markerEncoder(key, text) : text;
  };
}

let localeOverride: I18nState | null = null;
const overrideListeners = new Set<() => void>();

export function setLocaleOverride(next: I18nState | null) {
  localeOverride = next;
  overrideListeners.forEach((fn) => fn());
}

export function getLocaleOverride(): I18nState | null {
  return localeOverride;
}

export function subscribeLocaleOverride(fn: () => void): () => void {
  overrideListeners.add(fn);
  return () => {
    overrideListeners.delete(fn);
  };
}

export function t(key: string, params?: TranslateParams): string {
  return translator(state)(key, params);
}
