"use client";

import { PageShell } from "@/components/layout/page-shell";
import { AccountSettings } from "./account-settings";

// Раздел рисует свой заголовок сам. Обёртка ContentSection добавляла второй,
// «Аккаунт», и вместе с ним съедала PageShell: страница оставалась без шапки
// с поиском и переключателями, а лишний заголовок вылезал на её место.
export function SettingsAccount() {
  return (
    <PageShell variant="user">
      <AccountSettings />
    </PageShell>
  );
}
