import {
  Activity,
  Bell,
  BookOpen,
  ClipboardList,
  CreditCard,
  Gift,
  Globe,
  Headphones,
  Home,
  Newspaper,
  Palette,
  PlusCircle,
  Server,
  Settings,
  UserCog,
  Wrench,
} from "lucide-react";
import type { NavGroup } from "../types";
import { t } from "@/lib/i18n";

export function userSidebarNavGroups(): NavGroup[] {
  return [
  {
    title: t("nav.group.panel"),
    items: [
      {
        title: t("nav.dashboard"),
        url: "/dashboard",
        icon: Home,
      },
      {
        title: t("nav.activity"),
        url: "/activity",
        icon: ClipboardList,
      },
      {
        title: t("nav.monitoring"),
        url: "/monitoring",
        icon: Activity,
      },
      {
        title: t("nav.servers"),
        url: "/servers",
        icon: Server,
      },
      {
        title: t("nav.news"),
        url: "/news",
        icon: Newspaper,
      },
      {
        title: t("nav.rent_server"),
        url: "/rent-server",
        icon: PlusCircle,
        badge: "New",
      },
    ],
  },
  {
    title: t("nav.group.hosting"),
    items: [
      {
        title: t("nav.hosting_my"),
        url: "/hosting/my",
        icon: Globe,
      },
      {
        title: t("nav.hosting_rent"),
        url: "/hosting/rent",
        icon: PlusCircle,
        badge: "New",
      },
    ],
  },
  {
    title: t("nav.group.account"),
    items: [
      {
        title: t("nav.billing"),
        url: "/billing",
        icon: CreditCard,
      },
      {
        title: t("nav.daily_bonus"),
        url: "/daily-bonus",
        icon: Gift,
      },
      {
        title: t("nav.notifications"),
        url: "/notifications",
        icon: Bell,
      },
      {
        title: t("nav.support"),
        url: "/support",
        icon: Headphones,
      },
      {
        title: t("nav.kb"),
        url: "/support/kb",
        icon: BookOpen,
      },
    ],
  },
  {
    title: t("nav.group.settings"),
    items: [
      {
        title: t("nav.settings"),
        icon: Settings,
        items: [
          {
            title: t("nav.settings.profile"),
            url: "/settings",
            icon: UserCog,
          },
          {
            title: t("nav.settings.account"),
            url: "/settings/account",
            icon: Wrench,
          },
          {
            title: t("nav.settings.appearance"),
            url: "/settings?tab=appearance",
            icon: Palette,
          },
          {
            title: t("nav.settings.notifications"),
            url: "/settings?tab=notifications",
            icon: Bell,
          },
        ],
      },
    ],
  },
  ];
}
