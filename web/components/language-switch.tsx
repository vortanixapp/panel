"use client";

import { Check, Languages } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLocale } from "@/context/locale-provider";
import { hasSession, updateAccount } from "@/lib/api";
import { setAccountPreferences } from "@/lib/user-preferences";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function LanguageSwitch() {
  const t = useT();
  const { locale, languages } = useLocale();

  function choose(next: string) {
    if (next === locale) return;
    setAccountPreferences({ language: next });
    if (!hasSession()) return;
    void updateAccount({ locale: next }).catch((err: unknown) =>
      toast.error(
        t("layout.language_save_failed", {
          message: err instanceof Error ? err.message : String(err),
        })
      )
    );
  }

  if (languages.length < 2) return null;

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="scale-95 rounded-full">
          <Languages className="size-[1.2rem]" />
          <span className="sr-only">{t("layout.language_aria")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {languages.map((language) => (
          <DropdownMenuItem
            key={language.code}
            onClick={() => choose(language.code)}
          >
            {language.name}
            <Check
              size={14}
              className={cn("ms-auto", locale !== language.code && "hidden")}
            />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
