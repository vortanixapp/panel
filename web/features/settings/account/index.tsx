"use client";

import { PageShell } from "@/components/layout/page-shell";
import { AccountSettings } from "./account-settings";

export function SettingsAccount() {
  return (
    <PageShell variant="user">
      <AccountSettings />
    </PageShell>
  );
}
