type BaseNavItem = {
  id?: string;
  title: string;
  badge?: string;
  icon?: React.ElementType;
};

type NavLink = BaseNavItem & {
  url: string;
  newTab?: boolean;
  items?: never;
};

type NavCollapsible = BaseNavItem & {
  items: (BaseNavItem & { url: string; newTab?: boolean })[];
  url?: never;
};

type NavItem = NavCollapsible | NavLink;

type NavGroup = {
  id?: string;
  title: string;
  items: NavItem[];
};

export type { NavGroup, NavItem, NavCollapsible, NavLink };
