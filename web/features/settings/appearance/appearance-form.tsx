"use client";

import { useState } from "react";
import { toast } from "sonner";
import { fonts } from "@/config/fonts";
import { cn } from "@/lib/utils";
import { useFont } from "@/context/font-provider";
import { useTheme } from "@/context/theme-provider";
import { Button } from "@/components/ui/button";
import { useT } from "@/hooks/use-translations";

type ThemeOption = "light" | "dark" | "system";

const THEMES: { id: ThemeOption; labelKey: string; swatch: string }[] = [
  { id: "light", labelKey: "settings.appearance.theme_light", swatch: "bg-neutral-100" },
  { id: "dark", labelKey: "settings.appearance.theme_dark", swatch: "bg-neutral-900" },
  {
    id: "system",
    labelKey: "settings.appearance.theme_system",
    swatch: "bg-[linear-gradient(90deg,var(--color-neutral-100)_50%,var(--color-neutral-900)_50%)]",
  },
];

const FONT_LABELS: Record<string, string> = {
  inter: "Inter",
  manrope: "Manrope",
};

export function AppearanceForm() {
  const t = useT();
  const { font, setFont } = useFont();
  const { theme, setTheme } = useTheme();

  const [selectedTheme, setSelectedTheme] = useState<ThemeOption>(
    (theme as ThemeOption) ?? "system"
  );
  const [selectedFont, setSelectedFont] =
    useState<(typeof fonts)[number]>(font);

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (selectedFont !== font) setFont(selectedFont);
    if (selectedTheme !== theme) setTheme(selectedTheme);
    toast.success(t("settings.appearance.saved"));
  }

  return (
    <form onSubmit={onSubmit} className="flex max-w-[720px] flex-col gap-4">
      <section className="flex flex-col gap-4.5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
        <div className="space-y-1">
          <div className="text-[15px] leading-none font-semibold">
            {t("settings.appearance.theme_title")}
          </div>
          <div className="text-xs text-muted-foreground">
            {t("settings.appearance.theme_hint")}
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-3">
          {THEMES.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setSelectedTheme(item.id)}
              className={cn(
                "flex flex-col items-start gap-3 rounded-xl border bg-muted/30 p-4 text-left transition-colors",
                selectedTheme === item.id
                  ? "border-foreground/40 text-foreground"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <span
                className={cn("h-11 w-full rounded-lg border", item.swatch)}
              />
              <span className="text-[13px]">{t(item.labelKey)}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-4.5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
        <div className="text-[15px] leading-none font-semibold">
          {t("settings.appearance.font_title")}
        </div>
        <div className="flex flex-col gap-1 rounded-xl border bg-muted/30 p-1">
          {fonts.map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => setSelectedFont(item)}
              className={cn(
                "h-[34px] rounded-lg px-3 text-left text-[13px] transition-colors",
                selectedFont === item
                  ? "bg-background font-medium shadow-xs"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {item === "system"
                ? t("settings.appearance.font_system")
                : FONT_LABELS[item] ?? item}
            </button>
          ))}
        </div>
      </section>

      <Button type="submit" className="h-[38px] w-fit px-5 text-[13px]">
        {t("common.save")}
      </Button>
    </form>
  );
}
