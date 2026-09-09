import {
  BookOpen,
  Bug,
  ChartLine,
  ClipboardList,
  CloudDownload,
  CreditCard,
  Database,
  FileImage,
  Gift,
  Globe,
  Headphones,
  Key,
  Languages,
  LayoutDashboard,
  ListChecks,
  Mail,
  Map,
  MapPin,
  Monitor,
  Newspaper,
  Palette,
  Plug,
  Server,
  Settings,
  Shield,
  ShieldAlert,
  Tag,
  Terminal,
  Users,
  Wallet,
  Webhook,
} from "lucide-react";
import type { NavGroup } from "../types";
import type { TranslateFn } from "@/lib/i18n";

export function adminSidebarNavGroups(t: TranslateFn): NavGroup[] {
  return [
  {
    title: t("nav.group.general"),
    items: [
      {
        title: t("nav.admin.dashboard"),
        url: "/admin/dashboard",
        icon: LayoutDashboard,
      },
      {
        title: t("nav.admin.analytics"),
        url: "/admin/analytics",
        icon: ChartLine,
      },
      {
        title: t("nav.admin.users"),
        url: "/admin/users",
        icon: Users,
      },
      {
        title: t("nav.admin.servers"),
        url: "/admin/servers",
        icon: Server,
      },
      {
        title: t("nav.admin.support"),
        url: "/admin/support",
        icon: Headphones,
      },
      {
        title: t("nav.admin.kb"),
        url: "/admin/support/kb",
        icon: BookOpen,
      },
    ],
  },
  {
    title: t("nav.group.infrastructure"),
    items: [
      {
        title: t("nav.admin.locations"),
        url: "/admin/locations",
        icon: MapPin,
      },
      {
        title: t("nav.admin.images"),
        url: "/admin/images",
        icon: FileImage,
      },
      {
        title: "MySQL",
        url: "/admin/mysql",
        icon: Database,
      },
      {
        title: t("nav.admin.games"),
        url: "/admin/games",
        icon: Monitor,
      },
      {
        title: t("nav.admin.tariffs"),
        url: "/admin/tariffs",
        icon: Tag,
      },
      {
        title: "Vortanix Agent",
        url: "/admin/daemons",
        icon: Terminal,
      },
      {
        title: t("nav.admin.jobs"),
        url: "/admin/jobs",
        icon: ListChecks,
      },
    ],
  },
  {
    title: t("nav.group.hosting"),
    items: [
      {
        title: t("nav.admin.hosting_servers"),
        url: "/admin/hosting/servers",
        icon: Server,
      },
      {
        title: t("nav.admin.hosting_plans"),
        url: "/admin/hosting/plans",
        icon: Tag,
      },
      {
        title: t("nav.admin.hosting_accounts"),
        url: "/admin/hosting/accounts",
        icon: Globe,
      },
    ],
  },
  {
    title: t("nav.group.catalog"),
    items: [
      {
        title: t("nav.admin.plugins"),
        url: "/admin/plugins",
        icon: Plug,
      },
      {
        title: t("nav.admin.maps"),
        url: "/admin/maps",
        icon: Map,
      },
    ],
  },
  {
    title: t("nav.group.content"),
    items: [
      {
        title: t("nav.admin.news"),
        url: "/admin/news",
        icon: Newspaper,
      },
      {
        title: t("nav.admin.promo"),
        url: "/admin/promo",
        icon: Gift,
      },
      {
        title: t("nav.admin.mailings"),
        url: "/admin/mailings",
        icon: Mail,
      },
    ],
  },
  {
    title: t("nav.group.settings"),
    items: [
      {
        title: t("nav.admin.settings"),
        url: "/admin/settings",
        icon: Settings,
      },
      {
        title: t("nav.admin.appearance"),
        url: "/admin/settings/appearance",
        icon: Palette,
      },
      {
        title: t("nav.admin.billing"),
        url: "/admin/billing",
        icon: CreditCard,
      },
      {
        title: t("nav.admin.payment_providers"),
        url: "/admin/payment-providers",
        icon: Wallet,
      },
      {
        title: t("nav.admin.groups"),
        url: "/admin/groups",
        icon: Shield,
      },
      {
        title: t("nav.admin.language"),
        url: "/admin/language",
        icon: Languages,
      },
      {
        title: t("nav.admin.activity"),
        url: "/activity",
        icon: ClipboardList,
      },
      {
        title: t("nav.admin.logs"),
        url: "/admin/logs",
        icon: FileImage,
      },
      {
        title: t("nav.admin.security"),
        url: "/admin/security",
        icon: ShieldAlert,
      },
      {
        title: t("nav.admin.integrations"),
        url: "/admin/integrations",
        icon: Webhook,
      },
      {
        title: t("nav.admin.updates"),
        url: "/admin/updates",
        icon: CloudDownload,
      },
      {
        title: t("nav.admin.bug_report"),
        url: "/admin/bug-report",
        icon: Bug,
      },
    ],
  },
  ];
}
