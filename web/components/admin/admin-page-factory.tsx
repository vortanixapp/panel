import { AdminListPage } from "@/components/admin/admin-list-page";

const pages = [
  ["plugins", "Плагины", "plugins"],
  ["maps", "Карты", "maps"],
  ["billing", "Биллинг", "payments"],
  ["support", "Поддержка", "tickets"],
  ["news", "Новости", "news"],
  ["hosting/servers", "Хостинг: серверы", "servers"],
  ["hosting/plans", "Хостинг: тарифы", "plans"],
  ["hosting/accounts", "Хостинг: аккаунты", "accounts"],
  ["promotions", "Акции", "promotions"],
  ["mailings", "Рассылки", "mailings"],
] as const;

export function createAdminListPage(path: string, title: string, dataKey: string) {
  return function Page() {
    return <AdminListPage title={title} path={`/${path}`} dataKey={dataKey} />;
  };
}
