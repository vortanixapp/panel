export type PageGroup = "site" | "auth" | "cabinet" | "admin";

export type PageVariant = "site" | "panel";

export type RegistryPage = {
  key: string;
  path: string;
  group: PageGroup;
  titleKey: string;
  variant: PageVariant;
};

export const REGISTRY_PAGES: RegistryPage[] = [
  { key: "/", path: "/", group: "site", titleKey: "template.route.landing", variant: "site" },
  { key: "/login", path: "/login", group: "auth", titleKey: "template.route.login", variant: "site" },
  { key: "/register", path: "/register", group: "auth", titleKey: "template.route.register", variant: "site" },
  { key: "/forgot-password", path: "/forgot-password", group: "auth", titleKey: "template.route.forgot", variant: "site" },
  { key: "/dashboard", path: "/dashboard", group: "cabinet", titleKey: "template.route.dashboard", variant: "panel" },
  { key: "/servers", path: "/servers", group: "cabinet", titleKey: "template.route.servers", variant: "panel" },
  { key: "/rent-server", path: "/rent-server", group: "cabinet", titleKey: "template.route.rent_server", variant: "panel" },
  { key: "/hosting/my", path: "/hosting/my", group: "cabinet", titleKey: "template.route.hosting_my", variant: "panel" },
  { key: "/hosting/rent", path: "/hosting/rent", group: "cabinet", titleKey: "template.route.hosting_rent", variant: "panel" },
  { key: "/billing", path: "/billing", group: "cabinet", titleKey: "template.route.billing", variant: "panel" },
  { key: "/daily-bonus", path: "/daily-bonus", group: "cabinet", titleKey: "template.route.daily_bonus", variant: "panel" },
  { key: "/monitoring", path: "/monitoring", group: "cabinet", titleKey: "template.route.monitoring", variant: "panel" },
  { key: "/news", path: "/news", group: "cabinet", titleKey: "template.route.news", variant: "panel" },
  { key: "/support", path: "/support", group: "cabinet", titleKey: "template.route.support", variant: "panel" },
  { key: "/support/kb", path: "/support/kb", group: "cabinet", titleKey: "template.route.kb", variant: "panel" },
  { key: "/notifications", path: "/notifications", group: "cabinet", titleKey: "template.route.notifications", variant: "panel" },
  { key: "/settings", path: "/settings", group: "cabinet", titleKey: "template.route.settings", variant: "panel" },
  { key: "/activity", path: "/activity", group: "cabinet", titleKey: "template.route.activity", variant: "panel" },
  { key: "/games", path: "/games", group: "cabinet", titleKey: "template.route.games", variant: "panel" },
  { key: "/features", path: "/features", group: "cabinet", titleKey: "template.route.features", variant: "panel" },
  { key: "/admin/dashboard", path: "/admin/dashboard", group: "admin", titleKey: "template.route.admin_dashboard", variant: "panel" },
  { key: "/admin/servers", path: "/admin/servers", group: "admin", titleKey: "template.route.admin_servers", variant: "panel" },
  { key: "/admin/users", path: "/admin/users", group: "admin", titleKey: "template.route.admin_users", variant: "panel" },
];

const DYNAMIC_ROUTES = [
  "/admin/servers/[id]",
  "/admin/users/[id]",
  "/admin/locations/[id]",
  "/support/kb/[slug]",
  "/monitoring/public/[id]",
  "/servers/[id]",
  "/hosting/[id]",
  "/news/[slug]",
  "/support/[id]",
  "/monitoring/[id]",
  "/games/[slug]",
  "/legal/[kind]",
].map((pattern) => ({ pattern, parts: pattern.split("/").filter(Boolean) }));

const STATIC_KEYS = new Set(REGISTRY_PAGES.map((page) => page.key));

const CUSTOM_PREFIX = "/p/";

export function normalizePath(pathname: string): string {
  const clean = pathname.split(/[?#]/)[0].replace(/\/+$/, "");
  return clean || "/";
}

export function pageKeyOf(pathname: string): string {
  const path = normalizePath(pathname);
  if (STATIC_KEYS.has(path)) return path;
  const parts = path.split("/").filter(Boolean);
  for (const route of DYNAMIC_ROUTES) {
    if (parts.length < route.parts.length) continue;
    const matches = route.parts.every((part, i) => part.startsWith("[") || part === parts[i]);
    if (matches) return route.pattern;
  }
  return path;
}

export function customSlugOf(pathname: string): string | null {
  const path = normalizePath(pathname);
  if (!path.startsWith(CUSTOM_PREFIX)) return null;
  const slug = path.slice(CUSTOM_PREFIX.length);
  return slug && !slug.includes("/") ? slug : null;
}

export function customPagePath(slug: string): string {
  return `${CUSTOM_PREFIX}${slug}`;
}

export function registryPage(key: string): RegistryPage | undefined {
  return REGISTRY_PAGES.find((page) => page.key === key);
}

export function pageVariantOf(pathname: string): PageVariant {
  const key = pageKeyOf(pathname);
  const known = registryPage(key);
  if (known) return known.variant;
  const path = normalizePath(pathname);
  if (path === "/" || path.startsWith("/legal/") || path.startsWith(CUSTOM_PREFIX)) return "site";
  if (["/login", "/register", "/forgot-password", "/reset-password", "/two-factor-challenge"].includes(path)) {
    return "site";
  }
  return "panel";
}
