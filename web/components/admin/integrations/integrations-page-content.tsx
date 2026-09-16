"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { ApiKeysTab } from "@/components/admin/integrations/api-keys-tab";
import { WebhooksTab } from "@/components/admin/integrations/webhooks-tab";
import { WhmcsSection } from "@/components/admin/integrations/whmcs-section";
import { PageShell } from "@/components/layout/page-shell";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { fetchAdminAPIKeys, fetchAdminWebhooks } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const TABS = ["whmcs", "keys", "hooks"] as const;
type TabKey = (typeof TABS)[number];

function isTab(value: string | null): value is TabKey {
  return !!value && (TABS as readonly string[]).includes(value);
}

export function IntegrationsPageContent() {
  const t = useT();
  const [tab, setTab] = useState<TabKey>("whmcs");

  useEffect(() => {
    const value = new URLSearchParams(window.location.search).get("tab");
    if (isTab(value)) setTab(value);
  }, []);

  const keysQuery = useQuery({ queryKey: ["admin-api-keys"], queryFn: fetchAdminAPIKeys });
  const hooksQuery = useQuery({ queryKey: ["admin-webhooks"], queryFn: fetchAdminWebhooks });

  const activeKeys = (keysQuery.data?.keys ?? []).filter((key) => key.active).length;
  const hooks = hooksQuery.data?.webhooks ?? [];
  const failingHooks = hooks.some((hook) => hook.failed_count > 0);

  function changeTab(value: string) {
    if (!isTab(value)) return;
    setTab(value);
    const url = new URL(window.location.href);
    url.searchParams.set("tab", value);
    window.history.replaceState(window.history.state, "", url.toString());
  }

  return (
    <PageShell variant="admin">
      <div className="mb-5 space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">{t("admin.integrations.title")}</h1>
        <p className="max-w-3xl text-sm text-muted-foreground">
          {t("admin.integrations.subtitle")}
        </p>
      </div>

      <Tabs value={tab} onValueChange={changeTab}>
        <TabsList className="mb-5 h-auto flex-wrap">
          <TabsTrigger value="whmcs">{t("admin.integrations.whmcs.title")}</TabsTrigger>
          <TabsTrigger value="keys" className="gap-2">
            {t("admin.integrations.keys_title")}
            <Counter value={activeKeys} />
          </TabsTrigger>
          <TabsTrigger value="hooks" className="gap-2">
            {t("admin.integrations.hooks_title")}
            <Counter value={hooks.length} danger={failingHooks} />
          </TabsTrigger>
        </TabsList>
        <TabsContent value="whmcs">
          <WhmcsSection />
        </TabsContent>
        <TabsContent value="keys">
          <ApiKeysTab />
        </TabsContent>
        <TabsContent value="hooks">
          <WebhooksTab />
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}

function Counter({ value, danger }: { value: number; danger?: boolean }) {
  if (value === 0 && !danger) return null;
  return (
    <span
      className={cn(
        "inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full px-1.5 text-[11px] font-medium tabular-nums",
        danger
          ? "bg-destructive text-white"
          : "bg-muted-foreground/15 text-muted-foreground"
      )}
    >
      {value}
    </span>
  );
}
