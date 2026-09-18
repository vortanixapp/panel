export type LText = Record<string, string>;

export type MenuName = "site_header" | "site_footer" | "user_sidebar" | "admin_sidebar";

export const MENU_NAMES: MenuName[] = ["site_header", "site_footer", "user_sidebar", "admin_sidebar"];

export type ItemAudience = "" | "guests" | "users" | "staff";

export type SiteItem = {
  id: string;
  ref?: string;
  kind?: "link" | "group";
  label?: LText;
  icon?: string;
  url?: string;
  new_tab?: boolean;
  badge?: LText;
  audience?: ItemAudience;
  roles?: string[];
  hidden?: boolean;
  items?: SiteItem[];
};

export type SiteMenu = {
  items: SiteItem[];
};

export type BlockProps = Record<string, unknown>;

export type SiteBlock = {
  id: string;
  type: string;
  hidden?: boolean;
  props?: BlockProps;
};

export type SitePage = {
  blocks?: SiteBlock[];
  top?: SiteBlock[];
  bottom?: SiteBlock[];
  hidden?: string[];
};

export type PageAudience = "" | "guests" | "users";

export type CustomPage = {
  id: string;
  slug: string;
  title?: LText;
  description?: LText;
  layout?: "site" | "panel";
  audience?: PageAudience;
  hidden?: boolean;
  blocks: SiteBlock[];
};

export type SiteDocument = {
  menus?: Partial<Record<MenuName, SiteMenu>>;
  pages?: Record<string, SitePage>;
  custom_pages?: CustomPage[];
  texts?: Record<string, Record<string, string>>;
};

export type SiteZone = "blocks" | "top" | "bottom";

export type Viewer = {
  loggedIn: boolean;
  role: string;
};
