"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useT } from "@/hooks/use-translations";
import { useAccountQuery } from "@/hooks/use-account";
import { cn } from "@/lib/utils";
import { AppearanceForm } from "@/features/settings/appearance/appearance-form";
import { DisplayForm } from "@/features/settings/display/display-form";
import { NotificationsForm } from "@/features/settings/notifications/notifications-form";
import { ProfileTab } from "@/features/settings/account/profile-tab";
import { ContactsTab } from "@/features/settings/account/contacts-tab";
import { SecurityTab } from "@/features/settings/account/security-tab";
import { DevicesTab } from "@/features/settings/account/devices-tab";
import { DataTab } from "@/features/settings/account/data-tab";

type TabDef = { id: string; labelKey: string; icon: string };

const TABS: TabDef[] = [
  { id: "profile", labelKey: "settings.tab.profile", icon: "ri-user-3-line" },
  { id: "contacts", labelKey: "settings.tab.contacts", icon: "ri-contacts-book-2-line" },
  { id: "security", labelKey: "settings.tab.security", icon: "ri-shield-keyhole-line" },
  { id: "sessions", labelKey: "settings.tab.sessions", icon: "ri-device-line" },
  { id: "appearance", labelKey: "settings.tab.appearance", icon: "ri-palette-line" },
  { id: "notifications", labelKey: "settings.tab.notifications", icon: "ri-notification-3-line" },
  { id: "data", labelKey: "settings.tab.data", icon: "ri-folder-shield-2-line" },
];

const ALIASES: Record<string, { tab: string; anchor?: string }> = {
  account: { tab: "profile" },
  "2fa": { tab: "security", anchor: "twofa" },
  password: { tab: "security", anchor: "password" },
  devices: { tab: "sessions" },
};

function resolveTab(raw: string | null, tabs: TabDef[]): { tab: string; anchor?: string } {
  if (raw && ALIASES[raw]) return ALIASES[raw];
  if (raw && tabs.some((t) => t.id === raw)) return { tab: raw };
  return { tab: tabs[0].id };
}

export function AccountSettings() {
  const t = useT();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { data: account } = useAccountQuery();
  const tabs = TABS;
  const { tab, anchor } = resolveTab(searchParams.get("tab"), tabs);
  const listRef = useRef<HTMLDivElement | null>(null);
  const [edges, setEdges] = useState({ left: false, right: false });

  const select = useCallback(
    (id: string) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set("tab", id);
      router.replace(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  const updateEdges = useCallback(() => {
    const el = listRef.current;
    if (!el) return;
    setEdges({
      left: el.scrollLeft > 4,
      right: el.scrollLeft + el.clientWidth < el.scrollWidth - 4,
    });
  }, []);

  useEffect(() => {
    const el = listRef.current;
    if (!el) return;
    const active = el.querySelector<HTMLElement>(`[data-tab="${tab}"]`);
    active?.scrollIntoView({ block: "nearest", inline: "nearest" });
    updateEdges();
  }, [tab, updateEdges]);

  useEffect(() => {
    window.addEventListener("resize", updateEdges);
    return () => window.removeEventListener("resize", updateEdges);
  }, [updateEdges]);

  useEffect(() => {
    if (!anchor) return;
    const timer = window.setTimeout(() => {
      document.getElementById(anchor)?.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 250);
    return () => window.clearTimeout(timer);
  }, [anchor, account]);

  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    const index = tabs.findIndex((item) => item.id === tab);
    let next = -1;
    if (e.key === "ArrowRight") next = (index + 1) % tabs.length;
    if (e.key === "ArrowLeft") next = (index - 1 + tabs.length) % tabs.length;
    if (e.key === "Home") next = 0;
    if (e.key === "End") next = tabs.length - 1;
    if (next < 0) return;
    e.preventDefault();
    select(tabs[next].id);
    listRef.current?.querySelector<HTMLElement>(`[data-tab="${tabs[next].id}"]`)?.focus();
  };

  return (
    <div className="space-y-[22px]">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1.5">
          <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">{t("common.settings")}</h1>
          <p className="text-[13.5px] text-muted-foreground">{t("settings.page.subtitle")}</p>
        </div>
        <p className="flex items-center gap-2 text-[12.5px] text-muted-foreground">
          <i className="ri-information-line" />
          {t("settings.page.save_hint")}
        </p>
      </div>

      <div className="sticky top-0 z-20 -mx-1 bg-background/85 px-1 py-1.5 backdrop-blur supports-[backdrop-filter]:bg-background/70">
        <div className="relative">
          <div
            ref={listRef}
            role="tablist"
            aria-label={t("common.settings")}
            onKeyDown={onKeyDown}
            onScroll={updateEdges}
            className="flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          >
            {tabs.map((item) => {
              const active = item.id === tab;
              return (
                <button
                  key={item.id}
                  type="button"
                  role="tab"
                  data-tab={item.id}
                  id={`settings-tab-${item.id}`}
                  aria-selected={active}
                  aria-controls={`settings-panel-${item.id}`}
                  tabIndex={active ? 0 : -1}
                  onClick={() => select(item.id)}
                  className={cn(
                    "flex h-9 shrink-0 items-center gap-[7px] rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
                    active ? "bg-muted font-medium text-foreground" : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <i className={item.icon} />
                  {t(item.labelKey)}
                </button>
              );
            })}
          </div>
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-px left-px w-8 rounded-l-xl bg-gradient-to-r from-card to-transparent transition-opacity",
              edges.left ? "opacity-100" : "opacity-0"
            )}
          />
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-px right-px w-8 rounded-r-xl bg-gradient-to-l from-card to-transparent transition-opacity",
              edges.right ? "opacity-100" : "opacity-0"
            )}
          />
        </div>
      </div>

      <div role="tabpanel" id={`settings-panel-${tab}`} aria-labelledby={`settings-tab-${tab}`} className="space-y-4">
        {tab === "profile" && <ProfileTab />}
        {tab === "contacts" && <ContactsTab />}
        {tab === "security" && <SecurityTab />}
        {tab === "sessions" && <DevicesTab />}
        {tab === "appearance" && (
          <>
            <AppearanceForm />
            <DisplayForm />
          </>
        )}
        {tab === "notifications" && <NotificationsForm />}
        {tab === "data" && <DataTab />}
      </div>
    </div>
  );
}
