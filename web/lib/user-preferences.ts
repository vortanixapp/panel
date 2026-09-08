const ACCOUNT_KEY = "vortanix_account_prefs";
const DISPLAY_KEY = "vortanix_display_prefs";

export const ACCOUNT_PREFS_EVENT = "vortanix-account-prefs";

export type AccountPreferences = {
  language: string;
};

export type DisplayPreferences = {
  items: string[];
};

const DEFAULT_ACCOUNT: AccountPreferences = { language: "ru" };
const DEFAULT_DISPLAY: DisplayPreferences = {
  items: ["dashboard", "locations", "servers", "users"],
};

function normalizeDisplayItems(items: string[]): string[] {
  const normalized = items.map((id) => (id === "nodes" ? "locations" : id));
  return [...new Set(normalized)];
}

function read<T>(key: string, fallback: T): T {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return fallback;
    return { ...fallback, ...JSON.parse(raw) } as T;
  } catch {
    return fallback;
  }
}

function write<T>(key: string, value: T) {
  if (typeof window === "undefined") return;
  localStorage.setItem(key, JSON.stringify(value));
}

export function getAccountPreferences(): AccountPreferences {
  return read(ACCOUNT_KEY, DEFAULT_ACCOUNT);
}

export function setAccountPreferences(prefs: AccountPreferences) {
  write(ACCOUNT_KEY, prefs);
  // Событие слушает TranslationsProvider: без него смена языка применилась бы
  // только после перезагрузки страницы.
  window.dispatchEvent(new Event(ACCOUNT_PREFS_EVENT));
}

export function getDisplayPreferences(): DisplayPreferences {
  const prefs = read(DISPLAY_KEY, DEFAULT_DISPLAY);
  const items = normalizeDisplayItems(prefs.items ?? DEFAULT_DISPLAY.items);
  if (items.join(",") !== (prefs.items ?? []).join(",")) {
    const migrated = { items };
    write(DISPLAY_KEY, migrated);
    return migrated;
  }
  return { items };
}

export function setDisplayPreferences(prefs: DisplayPreferences) {
  write(DISPLAY_KEY, prefs);
  window.dispatchEvent(new Event("vortanix-display-prefs"));
}

// Подписи хранятся ключом, а не текстом: список читается на модульном уровне,
// и готовая строка застыла бы на языке, выбранном при загрузке страницы.
export const DISPLAY_NAV_ITEMS = [
  { id: "dashboard", labelKey: "common.overview", urls: ["/dashboard", "/admin/dashboard"] },
  { id: "locations", labelKey: "common.locations", urls: ["/admin/locations", "/admin/nodes"] },
  { id: "servers", labelKey: "common.servers", urls: ["/servers", "/admin/servers"] },
  { id: "users", labelKey: "common.users", urls: ["/admin/users"] },
] as const;

export function isNavUrlVisible(url: string, prefs = getDisplayPreferences()): boolean {
  const item = DISPLAY_NAV_ITEMS.find((i) =>
    (i.urls as readonly string[]).includes(url)
  );
  if (!item) return true;
  return (
    prefs.items.includes(item.id) ||
    (item.id === "locations" && prefs.items.includes("nodes"))
  );
}
