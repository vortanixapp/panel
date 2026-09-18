import type { MenuName, SiteBlock } from "@/lib/site/types";

export type BuiltinItem = {
  ref: string;
  kind?: "link" | "group";
  labelKey?: string;
  labelText?: string;
  url?: string;
  icon?: string;
  badge?: string;
  items?: BuiltinItem[];
};

const USER_SIDEBAR: BuiltinItem[] = [
  {
    ref: "user.group.panel",
    kind: "group",
    labelKey: "nav.group.panel",
    items: [
      { ref: "user.dashboard", labelKey: "nav.dashboard", url: "/dashboard", icon: "home" },
      { ref: "user.activity", labelKey: "nav.activity", url: "/activity", icon: "clipboard-list" },
      { ref: "user.monitoring", labelKey: "nav.monitoring", url: "/monitoring", icon: "activity" },
      { ref: "user.servers", labelKey: "nav.servers", url: "/servers", icon: "server" },
      { ref: "user.news", labelKey: "nav.news", url: "/news", icon: "newspaper" },
      { ref: "user.rent_server", labelKey: "nav.rent_server", url: "/rent-server", icon: "circle-plus", badge: "New" },
    ],
  },
  {
    ref: "user.group.hosting",
    kind: "group",
    labelKey: "nav.group.hosting",
    items: [
      { ref: "user.hosting_my", labelKey: "nav.hosting_my", url: "/hosting/my", icon: "globe" },
      { ref: "user.hosting_rent", labelKey: "nav.hosting_rent", url: "/hosting/rent", icon: "circle-plus", badge: "New" },
    ],
  },
  {
    ref: "user.group.account",
    kind: "group",
    labelKey: "nav.group.account",
    items: [
      { ref: "user.billing", labelKey: "nav.billing", url: "/billing", icon: "credit-card" },
      { ref: "user.daily_bonus", labelKey: "nav.daily_bonus", url: "/daily-bonus", icon: "gift" },
      { ref: "user.notifications", labelKey: "nav.notifications", url: "/notifications", icon: "bell" },
      { ref: "user.support", labelKey: "nav.support", url: "/support", icon: "headphones" },
      { ref: "user.kb", labelKey: "nav.kb", url: "/support/kb", icon: "book-open" },
    ],
  },
  {
    ref: "user.group.settings",
    kind: "group",
    labelKey: "nav.group.settings",
    items: [
      {
        ref: "user.settings",
        labelKey: "nav.settings",
        icon: "settings",
        items: [
          { ref: "user.settings.profile", labelKey: "nav.settings.profile", url: "/settings", icon: "user-cog" },
          { ref: "user.settings.account", labelKey: "nav.settings.account", url: "/settings/account", icon: "wrench" },
          { ref: "user.settings.appearance", labelKey: "nav.settings.appearance", url: "/settings?tab=appearance", icon: "palette" },
          { ref: "user.settings.notifications", labelKey: "nav.settings.notifications", url: "/settings?tab=notifications", icon: "bell" },
        ],
      },
    ],
  },
];

const ADMIN_SIDEBAR: BuiltinItem[] = [
  {
    ref: "admin.group.general",
    kind: "group",
    labelKey: "nav.group.general",
    items: [
      { ref: "admin.dashboard", labelKey: "nav.admin.dashboard", url: "/admin/dashboard", icon: "layout-dashboard" },
      { ref: "admin.analytics", labelKey: "nav.admin.analytics", url: "/admin/analytics", icon: "chart-line" },
      { ref: "admin.users", labelKey: "nav.admin.users", url: "/admin/users", icon: "users" },
      { ref: "admin.servers", labelKey: "nav.admin.servers", url: "/admin/servers", icon: "server" },
      { ref: "admin.support", labelKey: "nav.admin.support", url: "/admin/support", icon: "headphones" },
      { ref: "admin.kb", labelKey: "nav.admin.kb", url: "/admin/support/kb", icon: "book-open" },
    ],
  },
  {
    ref: "admin.group.infrastructure",
    kind: "group",
    labelKey: "nav.group.infrastructure",
    items: [
      { ref: "admin.locations", labelKey: "nav.admin.locations", url: "/admin/locations", icon: "map-pin" },
      { ref: "admin.images", labelKey: "nav.admin.images", url: "/admin/images", icon: "file-image" },
      { ref: "admin.mysql", labelText: "MySQL", url: "/admin/mysql", icon: "database" },
      { ref: "admin.games", labelKey: "nav.admin.games", url: "/admin/games", icon: "monitor" },
      { ref: "admin.tariffs", labelKey: "nav.admin.tariffs", url: "/admin/tariffs", icon: "tag" },
      { ref: "admin.daemons", labelText: "Vortanix Agent", url: "/admin/daemons", icon: "terminal" },
      { ref: "admin.jobs", labelKey: "nav.admin.jobs", url: "/admin/jobs", icon: "list-checks" },
    ],
  },
  {
    ref: "admin.group.hosting",
    kind: "group",
    labelKey: "nav.group.hosting",
    items: [
      { ref: "admin.hosting_servers", labelKey: "nav.admin.hosting_servers", url: "/admin/hosting/servers", icon: "server" },
      { ref: "admin.hosting_plans", labelKey: "nav.admin.hosting_plans", url: "/admin/hosting/plans", icon: "tag" },
      { ref: "admin.hosting_accounts", labelKey: "nav.admin.hosting_accounts", url: "/admin/hosting/accounts", icon: "globe" },
    ],
  },
  {
    ref: "admin.group.catalog",
    kind: "group",
    labelKey: "nav.group.catalog",
    items: [
      { ref: "admin.plugins", labelKey: "nav.admin.plugins", url: "/admin/plugins", icon: "plug" },
      { ref: "admin.maps", labelKey: "nav.admin.maps", url: "/admin/maps", icon: "map" },
    ],
  },
  {
    ref: "admin.group.content",
    kind: "group",
    labelKey: "nav.group.content",
    items: [
      { ref: "admin.news", labelKey: "nav.admin.news", url: "/admin/news", icon: "newspaper" },
      { ref: "admin.promo", labelKey: "nav.admin.promo", url: "/admin/promo", icon: "gift" },
      { ref: "admin.mailings", labelKey: "nav.admin.mailings", url: "/admin/mailings", icon: "mail" },
    ],
  },
  {
    ref: "admin.group.settings",
    kind: "group",
    labelKey: "nav.group.settings",
    items: [
      { ref: "admin.settings", labelKey: "nav.admin.settings", url: "/admin/settings", icon: "settings" },
      { ref: "admin.appearance", labelKey: "nav.admin.appearance", url: "/admin/settings/appearance", icon: "palette" },
      { ref: "admin.template", labelKey: "nav.admin.template", url: "/admin/template", icon: "layout-grid" },
      { ref: "admin.billing", labelKey: "nav.admin.billing", url: "/admin/billing", icon: "credit-card" },
      { ref: "admin.accounting", labelKey: "nav.admin.accounting", url: "/admin/accounting", icon: "calculator" },
      { ref: "admin.payment_providers", labelKey: "nav.admin.payment_providers", url: "/admin/payment-providers", icon: "wallet" },
      { ref: "admin.groups", labelKey: "nav.admin.groups", url: "/admin/groups", icon: "shield" },
      { ref: "admin.language", labelKey: "nav.admin.language", url: "/admin/language", icon: "languages" },
      { ref: "admin.activity", labelKey: "nav.admin.activity", url: "/activity", icon: "clipboard-list" },
      { ref: "admin.logs", labelKey: "nav.admin.logs", url: "/admin/logs", icon: "file-text" },
      { ref: "admin.security", labelKey: "nav.admin.security", url: "/admin/security", icon: "shield-alert" },
      { ref: "admin.legal", labelKey: "nav.admin.legal", url: "/admin/legal", icon: "scale" },
      { ref: "admin.abuse", labelKey: "nav.admin.abuse", url: "/admin/abuse", icon: "siren" },
      { ref: "admin.integrations", labelKey: "nav.admin.integrations", url: "/admin/integrations", icon: "webhook" },
      { ref: "admin.updates", labelKey: "nav.admin.updates", url: "/admin/updates", icon: "cloud-download" },
      { ref: "admin.bug_report", labelKey: "nav.admin.bug_report", url: "/admin/bug-report", icon: "bug" },
    ],
  },
];

const SITE_HEADER: BuiltinItem[] = [
  { ref: "site.pricing", labelKey: "landing.nav.pricing", url: "#pricing" },
  { ref: "site.games", labelKey: "landing.nav.games", url: "/games" },
  { ref: "site.features", labelKey: "landing.nav.features", url: "/features" },
  { ref: "site.faq", labelKey: "landing.nav.faq", url: "#faq" },
];

const SITE_FOOTER: BuiltinItem[] = [
  {
    ref: "footer.product",
    kind: "group",
    labelKey: "landing.footer.product",
    items: [
      { ref: "footer.pricing", labelKey: "landing.nav.pricing", url: "#pricing" },
      { ref: "footer.games", labelKey: "landing.nav.games", url: "/games" },
      { ref: "footer.features", labelKey: "landing.nav.features", url: "/features" },
      { ref: "footer.status", labelKey: "landing.footer.status", url: "/status" },
    ],
  },
  {
    ref: "footer.company",
    kind: "group",
    labelKey: "landing.footer.company",
    items: [
      { ref: "footer.about", labelKey: "landing.footer.about", url: "/about" },
      { ref: "footer.blog", labelKey: "landing.footer.blog", url: "/blog" },
      { ref: "footer.support", labelKey: "landing.footer.support", url: "/support" },
    ],
  },
  {
    ref: "footer.resources",
    kind: "group",
    labelKey: "landing.footer.resources",
    items: [
      { ref: "footer.kb", labelKey: "landing.footer.kb", url: "/kb" },
      { ref: "footer.faq", labelKey: "landing.nav.faq", url: "#faq" },
      { ref: "footer.api", labelKey: "landing.footer.api", url: "/kb" },
    ],
  },
  {
    ref: "footer.account",
    kind: "group",
    labelKey: "landing.footer.account",
    items: [
      { ref: "footer.sign_in", labelKey: "landing.footer.sign_in", url: "/login" },
      { ref: "footer.register", labelKey: "landing.footer.register", url: "/register" },
    ],
  },
];

export const DEFAULT_MENUS: Record<MenuName, BuiltinItem[]> = {
  site_header: SITE_HEADER,
  site_footer: SITE_FOOTER,
  user_sidebar: USER_SIDEBAR,
  admin_sidebar: ADMIN_SIDEBAR,
};

export const LANDING_SECTIONS = [
  "hero",
  "games",
  "steps",
  "hardware",
  "panel",
  "locations",
  "pricing",
  "faq",
  "cta",
] as const;

export type LandingSection = (typeof LANDING_SECTIONS)[number];

export const SECTION_ANCHORS: Record<string, LandingSection> = {
  "#pricing": "pricing",
  "#faq": "faq",
  "#games": "games",
  "#hardware": "hardware",
};

export function defaultLandingBlocks(): SiteBlock[] {
  return LANDING_SECTIONS.map((name) => ({ id: name, type: `landing.${name}` }));
}
