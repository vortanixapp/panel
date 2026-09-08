"use client";

import { useEffect } from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { CaretSortIcon, CheckIcon } from "@radix-ui/react-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { PasswordInput } from "@/components/password-input";
import { useChangePassword } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";
import {
  getAccountPreferences,
  setAccountPreferences,
} from "@/lib/user-preferences";

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const languages = [
  { labelKey: "settings.account.language_ru", value: "ru" },
  { labelKey: "settings.account.language_en", value: "en" },
] as const;

// Схема собирается внутри компонента: сообщения об ошибках должны меняться
// вместе с языком, а модульная константа заморозила бы их на первом рендере.
const buildSchema = (t: TranslateFn) =>
  z
    .object({
      language: z.string().min(1, t("settings.account.language_select")),
      currentPassword: z.string().optional(),
      newPassword: z.string().optional(),
      confirmPassword: z.string().optional(),
    })
    .superRefine((data, ctx) => {
      const changing =
        !!data.currentPassword || !!data.newPassword || !!data.confirmPassword;
      if (!changing) return;
      if (!data.currentPassword) {
        ctx.addIssue({
          code: "custom",
          message: t("settings.account.password_current_required"),
          path: ["currentPassword"],
        });
      }
      if (!data.newPassword || data.newPassword.length < 8) {
        ctx.addIssue({
          code: "custom",
          message: t("settings.account.password_min"),
          path: ["newPassword"],
        });
      }
      if (data.newPassword !== data.confirmPassword) {
        ctx.addIssue({
          code: "custom",
          message: t("settings.account.password_mismatch"),
          path: ["confirmPassword"],
        });
      }
    });

type AccountFormValues = z.infer<ReturnType<typeof buildSchema>>;

export function AccountForm() {
  const t = useT();
  const changePassword = useChangePassword();
  const form = useForm<AccountFormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: {
      language: "ru",
      currentPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  useEffect(() => {
    const prefs = getAccountPreferences();
    form.reset({
      language: prefs.language,
      currentPassword: "",
      newPassword: "",
      confirmPassword: "",
    });
  }, [form]);

  async function onSubmit(data: AccountFormValues) {
    setAccountPreferences({ language: data.language });

    const changing =
      data.currentPassword || data.newPassword || data.confirmPassword;
    if (changing) {
      try {
        await changePassword.mutateAsync({
          currentPassword: data.currentPassword ?? "",
          newPassword: data.newPassword ?? "",
        });
        form.setValue("currentPassword", "");
        form.setValue("newPassword", "");
        form.setValue("confirmPassword", "");
        toast.success(t("settings.account.password_updated"));
      } catch (err) {
        toast.error(
          err instanceof Error
            ? err.message
            : t("settings.account.password_change_failed")
        );
        return;
      }
    }

    toast.success(t("settings.account.saved"));
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
        <FormField
          control={form.control}
          name="language"
          render={({ field }) => (
            <FormItem className="flex flex-col">
              <FormLabel>{t("settings.account.language_label")}</FormLabel>
              <Popover>
                <PopoverTrigger asChild>
                  <FormControl>
                    <Button
                      variant="outline"
                      role="combobox"
                      className={cn(
                        "w-50 justify-between",
                        !field.value && "text-muted-foreground"
                      )}
                    >
                      {field.value
                        ? t(
                            languages.find((l) => l.value === field.value)
                              ?.labelKey ?? "settings.account.language_select"
                          )
                        : t("settings.account.language_select")}
                      <CaretSortIcon className="ms-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                  </FormControl>
                </PopoverTrigger>
                <PopoverContent className="w-50 p-0">
                  <Command>
                    <CommandInput placeholder={t("common.search_placeholder")} />
                    <CommandEmpty>{t("common.not_found")}</CommandEmpty>
                    <CommandGroup>
                      <CommandList>
                        {languages.map((language) => (
                          <CommandItem
                            value={t(language.labelKey)}
                            key={language.value}
                            onSelect={() => form.setValue("language", language.value)}
                          >
                            <CheckIcon
                              className={cn(
                                "size-4",
                                language.value === field.value
                                  ? "opacity-100"
                                  : "opacity-0"
                              )}
                            />
                            {t(language.labelKey)}
                          </CommandItem>
                        ))}
                      </CommandList>
                    </CommandGroup>
                  </Command>
                </PopoverContent>
              </Popover>
              <FormDescription>{t("settings.account.language_hint")}</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="space-y-4 rounded-lg border p-4">
          <div>
            <h4 className="text-sm font-medium">
              {t("settings.account.password_section")}
            </h4>
            <p className="text-sm text-muted-foreground">
              {t("settings.account.password_section_hint")}
            </p>
          </div>
          <FormField
            control={form.control}
            name="currentPassword"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("settings.account.password_current")}</FormLabel>
                <FormControl>
                  <PasswordInput autoComplete="current-password" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="newPassword"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("settings.account.password_new")}</FormLabel>
                <FormControl>
                  <PasswordInput autoComplete="new-password" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="confirmPassword"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("settings.account.password_confirm")}</FormLabel>
                <FormControl>
                  <Input type="password" autoComplete="new-password" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <Button type="submit" disabled={changePassword.isPending}>
          {changePassword.isPending ? t("common.saving") : t("common.save")}
        </Button>
      </form>
    </Form>
  );
}
