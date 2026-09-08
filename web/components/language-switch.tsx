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
import { updateAccount } from "@/lib/api";
import { SUPPORTED_LOCALES, getLocale, type Locale } from "@/lib/i18n";
import { setAccountPreferences } from "@/lib/user-preferences";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

// Языки подписаны на самих себе: «Русский», а не «Russian». Человек, попавший
// на чужой язык интерфейса, ищет в списке знакомое слово, а перевод названий
// сделал бы этот список нечитаемым ровно в тот момент, когда он нужен.
const LOCALE_NAMES: Record<Locale, string> = {
  ru: "Русский",
  en: "English",
};

export function LanguageSwitch() {
  // useT подписывает компонент на смену языка: без подписки галочка осталась бы
  // на прежнем пункте до следующей перерисовки шапки.
  const t = useT();
  const locale = getLocale();

  function choose(next: Locale) {
    if (next === locale) return;
    // Локальная настройка переключает интерфейс сразу, запрос сохраняет выбор
    // для других устройств. Ждать ответ сервера, чтобы сменить язык, незачем.
    setAccountPreferences({ language: next });
    void updateAccount({ locale: next }).catch((err: unknown) =>
      toast.error(
        t("layout.language_save_failed", {
          message: err instanceof Error ? err.message : String(err),
        })
      )
    );
  }

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="scale-95 rounded-full">
          <Languages className="size-[1.2rem]" />
          <span className="sr-only">{t("layout.language_aria")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {SUPPORTED_LOCALES.map((code) => (
          <DropdownMenuItem key={code} onClick={() => choose(code)}>
            {LOCALE_NAMES[code]}
            <Check size={14} className={cn("ms-auto", locale !== code && "hidden")} />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
