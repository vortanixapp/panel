"use client";

import { useMutation } from "@tanstack/react-query";
import { Check, Lock } from "lucide-react";
import { toast } from "sonner";
import { fonts } from "@/config/fonts";
import { useFont } from "@/context/font-provider";
import { useLayout, type Collapsible } from "@/context/layout-provider";
import { useTheme } from "@/context/theme-provider";
import { DisplayForm } from "@/features/settings/display/display-form";
import { useAccountQuery, useSetAccount } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import { updateAccount } from "@/lib/api";
import { cn } from "@/lib/utils";
import { errorText, Field, SettingsSection } from "./ui";

const THEMES = [
  { id: "light", labelKey: "settings.appearance.theme_light", bg: "#f5f5f4", fg: "#1c1917", panel: "#ffffff" },
  { id: "dark", labelKey: "settings.appearance.theme_dark", bg: "#0c0c0d", fg: "#fafafa", panel: "#18181b" },
  { id: "system", labelKey: "settings.appearance.theme_system", bg: "linear-gradient(135deg,#f5f5f4 50%,#0c0c0d 50%)", fg: "#71717a", panel: "transparent" },
] as const;

const FONT_FAMILIES: Record<string, string> = {
  inter: "Inter, sans-serif",
  manrope: "Manrope, sans-serif",
  system: "system-ui, sans-serif",
};

const VARIANTS = [
  { id: "inset", labelKey: "layout.sidebar.inset" },
  { id: "floating", labelKey: "layout.sidebar.floating" },
  { id: "sidebar", labelKey: "layout.sidebar.standard" },
] as const;

const COLLAPSE: { id: Collapsible; labelKey: string }[] = [
  { id: "icon", labelKey: "settings.appearance.collapse_icon" },
  { id: "offcanvas", labelKey: "settings.appearance.collapse_hide" },
];

const START_PAGES = [
  { id: "", labelKey: "settings.appearance.start_default" },
  { id: "/dashboard", labelKey: "nav.dashboard" },
  { id: "/servers", labelKey: "nav.servers" },
  { id: "/billing", labelKey: "nav.billing" },
  { id: "/notifications", labelKey: "nav.notifications" },
  { id: "/support", labelKey: "nav.support" },
];

function OptionCard({
  active,
  disabled,
  onClick,
  children,
}: {
  active: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={active}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "relative flex flex-col gap-2 rounded-xl border p-3 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-60",
        active ? "border-primary ring-1 ring-primary" : "border-border hover:border-foreground/30"
      )}
    >
      {active && (
        <span className="absolute top-2 right-2 grid size-5 place-items-center rounded-full bg-primary text-primary-foreground">
          <Check className="size-3" />
        </span>
      )}
      {children}
    </button>
  );
}

export function AppearanceTab() {
  const t = useT();
  const account = useAccountQuery();
  const setAccount = useSetAccount();
  const { theme, setTheme, locked } = useTheme();
  const { font, setFont } = useFont();
  const { variant, setVariant, collapsible, setCollapsible } = useLayout();
  const startPage = account.data?.user.preferences?.start_page ?? "";

  const saveStart = useMutation({
    mutationFn: (page: string) => updateAccount({ preferences: { start_page: page } }),
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.appearance.start_saved"));
    },
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  return (
    <>
      <SettingsSection
        title={t("settings.appearance.theme_title")}
        description={locked ? t("settings.appearance.theme_locked") : t("settings.appearance.synced_hint")}
        action={locked ? <Lock className="size-4 text-muted-foreground" /> : undefined}
      >
        <div role="radiogroup" className="grid grid-cols-3 gap-3">
          {THEMES.map((item) => (
            <OptionCard
              key={item.id}
              active={theme === item.id}
              disabled={locked}
              onClick={() => setTheme(item.id)}
            >
              <span className="block h-16 w-full overflow-hidden rounded-lg border border-border" style={{ background: item.bg }}>
                <span className="m-2 block h-3 w-1/2 rounded" style={{ background: item.fg, opacity: 0.8 }} />
                <span className="mx-2 block h-6 rounded" style={{ background: item.panel, border: "1px solid rgba(127,127,127,.25)" }} />
              </span>
              <span className="text-[13px] font-medium">{t(item.labelKey)}</span>
            </OptionCard>
          ))}
        </div>
      </SettingsSection>

      <SettingsSection title={t("settings.appearance.font_title")} description={t("settings.appearance.font_hint")}>
        <div role="radiogroup" className="grid grid-cols-3 gap-3">
          {fonts.map((item) => (
            <OptionCard key={item} active={font === item} onClick={() => setFont(item)}>
              <span className="text-2xl leading-none" style={{ fontFamily: FONT_FAMILIES[item] }}>
                Аа
              </span>
              <span className="text-[13px] font-medium" style={{ fontFamily: FONT_FAMILIES[item] }}>
                {item === "system" ? t("settings.appearance.font_system") : item.charAt(0).toUpperCase() + item.slice(1)}
              </span>
            </OptionCard>
          ))}
        </div>
      </SettingsSection>

      <SettingsSection title={t("settings.appearance.menu_title")} description={t("settings.appearance.menu_hint")}>
        <div className="grid gap-5 md:grid-cols-2">
          <Field label={t("settings.appearance.menu_variant")}>
            <div role="radiogroup" className="grid grid-cols-3 gap-2">
              {VARIANTS.map((item) => (
                <OptionCard key={item.id} active={variant === item.id} onClick={() => setVariant(item.id)}>
                  <span className="text-[12.5px] font-medium">{t(item.labelKey)}</span>
                </OptionCard>
              ))}
            </div>
          </Field>
          <Field label={t("settings.appearance.menu_collapse")}>
            <div role="radiogroup" className="grid grid-cols-2 gap-2">
              {COLLAPSE.map((item) => (
                <OptionCard key={item.id} active={collapsible === item.id} onClick={() => setCollapsible(item.id)}>
                  <span className="text-[12.5px] font-medium">{t(item.labelKey)}</span>
                </OptionCard>
              ))}
            </div>
          </Field>
        </div>
      </SettingsSection>

      <SettingsSection title={t("settings.appearance.start_title")} description={t("settings.appearance.start_hint")}>
        <select
          aria-label={t("settings.appearance.start_title")}
          className="flex h-9 w-full max-w-sm rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
          value={startPage}
          disabled={saveStart.isPending || account.isLoading}
          onChange={(e) => saveStart.mutate(e.target.value)}
        >
          {START_PAGES.map((item) => (
            <option key={item.id} value={item.id}>
              {t(item.labelKey)}
            </option>
          ))}
        </select>
      </SettingsSection>

      {account.data?.user.staff && <DisplayForm />}
    </>
  );
}
