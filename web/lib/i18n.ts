import { EN } from "@/lib/locales/en";
import { RU } from "@/lib/locales/ru";

export type Locale = "ru" | "en";

export const DEFAULT_LOCALE: Locale = "ru";
export const LOCALE_COOKIE_NAME = "vortanix-locale";
export const LOCALE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;
export const SUPPORTED_LOCALES: Locale[] = ["en", "ru"];

const CATALOGS: Record<Locale, Record<string, string>> = { ru: RU, en: EN };

const EN_COVERS_RU: Record<keyof typeof RU, string> = EN;
const RU_COVERS_EN: Record<keyof typeof EN, string> = RU;
void EN_COVERS_RU;
void RU_COVERS_EN;

export type OverridableString = {
  key: string;
  default: string;
  groupKey: string;
};

const OVERRIDABLE_KEYS: { key: string; groupKey: string }[] = [
  { key: "nav.dashboard", groupKey: "admin.language.group.user_menu" },
  { key: "nav.servers", groupKey: "admin.language.group.user_menu" },
  { key: "nav.monitoring", groupKey: "admin.language.group.user_menu" },
  { key: "nav.news", groupKey: "admin.language.group.user_menu" },
  { key: "nav.rent_server", groupKey: "admin.language.group.user_menu" },
  { key: "nav.hosting_my", groupKey: "admin.language.group.user_menu" },
  { key: "nav.hosting_rent", groupKey: "admin.language.group.user_menu" },
  { key: "nav.billing", groupKey: "admin.language.group.user_menu" },
  { key: "nav.support", groupKey: "admin.language.group.user_menu" },
  { key: "nav.notifications", groupKey: "admin.language.group.user_menu" },
  { key: "nav.activity", groupKey: "admin.language.group.user_menu" },
  { key: "nav.daily_bonus", groupKey: "admin.language.group.user_menu" },
  { key: "nav.kb", groupKey: "admin.language.group.user_menu" },

  { key: "nav.admin.dashboard", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.analytics", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.users", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.servers", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.support", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.kb", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.locations", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.images", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.games", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.tariffs", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.jobs", groupKey: "admin.language.group.admin_menu" },
  {
    key: "nav.admin.hosting_servers",
    groupKey: "admin.language.group.admin_menu",
  },
  {
    key: "nav.admin.hosting_plans",
    groupKey: "admin.language.group.admin_menu",
  },
  {
    key: "nav.admin.hosting_accounts",
    groupKey: "admin.language.group.admin_menu",
  },
  { key: "nav.admin.plugins", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.maps", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.news", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.promo", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.mailings", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.settings", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.appearance", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.billing", groupKey: "admin.language.group.admin_menu" },
  {
    key: "nav.admin.payment_providers",
    groupKey: "admin.language.group.admin_menu",
  },
  { key: "nav.admin.groups", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.language", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.activity", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.logs", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.security", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.integrations", groupKey: "admin.language.group.admin_menu" },
  { key: "nav.admin.bug_report", groupKey: "admin.language.group.admin_menu" },

  { key: "nav.settings", groupKey: "admin.language.group.settings_menu" },
  {
    key: "nav.settings.profile",
    groupKey: "admin.language.group.settings_menu",
  },
  {
    key: "nav.settings.account",
    groupKey: "admin.language.group.settings_menu",
  },
  {
    key: "nav.settings.appearance",
    groupKey: "admin.language.group.settings_menu",
  },
  {
    key: "nav.settings.notifications",
    groupKey: "admin.language.group.settings_menu",
  },

  { key: "nav.group.panel", groupKey: "admin.language.group.menu_sections" },
  { key: "nav.group.general", groupKey: "admin.language.group.menu_sections" },
  {
    key: "nav.group.infrastructure",
    groupKey: "admin.language.group.menu_sections",
  },
  { key: "nav.group.hosting", groupKey: "admin.language.group.menu_sections" },
  { key: "nav.group.catalog", groupKey: "admin.language.group.menu_sections" },
  { key: "nav.group.content", groupKey: "admin.language.group.menu_sections" },
  { key: "nav.group.account", groupKey: "admin.language.group.menu_sections" },
  { key: "nav.group.settings", groupKey: "admin.language.group.menu_sections" },

  { key: "server.tab.main", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.console", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.logs", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.metrics", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.ftp", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.mysql", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.cron", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.firewall", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.ports", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.settings", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.tariff", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.plugins", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.maps", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.copies", groupKey: "admin.language.group.server_tabs" },
  { key: "server.tab.friends", groupKey: "admin.language.group.server_tabs" },
];

export const OVERRIDABLE_STRINGS: OverridableString[] = OVERRIDABLE_KEYS.map(
  ({ key, groupKey }) => ({
    key,
    groupKey,
    default: EN[key as keyof typeof EN] ?? key,
  })
);

let overrides: Record<string, string> = {};
let locale: Locale = DEFAULT_LOCALE;

let version = 0;
const listeners = new Set<() => void>();

function notify() {
  version += 1;
  listeners.forEach((fn) => fn());
}

export function setTranslationOverrides(next: Record<string, string>) {
  overrides = next ?? {};
  notify();
}

export function subscribeToTranslations(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function getTranslationOverrides(): Record<string, string> {
  return overrides;
}

export function getTranslationsVersion(): number {
  return version;
}

export function normalizeLocale(value: string | null | undefined): Locale {
  return SUPPORTED_LOCALES.includes(value as Locale)
    ? (value as Locale)
    : DEFAULT_LOCALE;
}

export function browserLocale(): Locale {
  if (typeof navigator === "undefined") return DEFAULT_LOCALE;
  for (const candidate of navigator.languages ?? [navigator.language]) {
    const base = candidate?.split("-")[0];
    if (SUPPORTED_LOCALES.includes(base as Locale)) return base as Locale;
  }
  return DEFAULT_LOCALE;
}

export function getLocale(): Locale {
  return locale;
}

export function setLocale(next: string | null | undefined) {
  const normalized = normalizeLocale(next);
  if (normalized === locale) return;
  locale = normalized;
  notify();
}

const LOCALE_TAGS: Record<Locale, string> = {
  ru: "ru-RU",
  en: "en-US",
};

export function localeTag(): string {
  return LOCALE_TAGS[locale] ?? LOCALE_TAGS[DEFAULT_LOCALE];
}

export type TranslateParams = Record<string, string | number>;

function interpolate(template: string, params: TranslateParams): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined || value === null ? match : String(value);
  });
}

export function translateWith(target: Locale): TranslateFn {
  return (key: string, params?: TranslateParams): string => {
    const override = overrides[key];
    let value: string | undefined;
    if (typeof override === "string" && override.trim() !== "") {
      value = override;
    } else {
      value = CATALOGS[target][key] ?? CATALOGS[DEFAULT_LOCALE][key];
    }
    if (value === undefined) return key;
    return params ? interpolate(value, params) : value;
  };
}

export function localeTagOf(target: Locale): string {
  return LOCALE_TAGS[target] ?? LOCALE_TAGS[DEFAULT_LOCALE];
}

export function t(key: string, params?: TranslateParams): string {
  return translateWith(locale)(key, params);
}

export type TranslateFn = (key: string, params?: TranslateParams) => string;
