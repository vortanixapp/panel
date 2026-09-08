"use client";

import { useEffect } from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { cn } from "@/lib/utils";
import type { TranslateFn } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import {
  DISPLAY_NAV_ITEMS,
  getDisplayPreferences,
  setDisplayPreferences,
} from "@/lib/user-preferences";

// Схема собирается внутри компонента: сообщение об ошибке должно меняться
// вместе с языком, а модульная константа заморозила бы его на первом рендере.
const buildSchema = (t: TranslateFn) =>
  z.object({
    items: z.array(z.string()).refine((value) => value.some((item) => item), {
      message: t("settings.display.pick_one"),
    }),
  });

type DisplayFormValues = z.infer<ReturnType<typeof buildSchema>>;

export function DisplayForm() {
  const t = useT();
  const form = useForm<DisplayFormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: getDisplayPreferences(),
  });

  useEffect(() => {
    form.reset(getDisplayPreferences());
  }, [form]);

  function onSubmit(data: DisplayFormValues) {
    setDisplayPreferences(data);
    toast.success(t("settings.display.saved"));
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="flex max-w-[720px] flex-col gap-4"
      >
        <section className="flex flex-col gap-4.5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
          <div className="space-y-1">
            <div className="text-[15px] leading-none font-semibold">
              {t("settings.display.sidebar_title")}
            </div>
            <div className="text-xs leading-relaxed text-muted-foreground">
              {t("settings.display.sidebar_hint")}
            </div>
          </div>

          <FormField
            control={form.control}
            name="items"
            render={({ field }) => (
              <FormItem className="space-y-2.5">
                {DISPLAY_NAV_ITEMS.map((item) => {
                  const checked = field.value?.includes(item.id);
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() =>
                        field.onChange(
                          checked
                            ? field.value.filter((v) => v !== item.id)
                            : [...field.value, item.id]
                        )
                      }
                      className="flex w-full items-center gap-3 rounded-xl border bg-muted/30 px-3.5 py-3 text-left transition-colors hover:bg-muted/50"
                    >
                      <span
                        className={cn(
                          "flex size-[18px] shrink-0 items-center justify-center rounded-[5px] border",
                          checked
                            ? "border-transparent bg-primary text-primary-foreground"
                            : "border-input"
                        )}
                      >
                        {checked && <Check className="size-3" />}
                      </span>
                      <span className="text-[13px]">{t(item.labelKey)}</span>
                    </button>
                  );
                })}
                <FormMessage className="text-xs" />
              </FormItem>
            )}
          />
        </section>

        <Button type="submit" className="h-[38px] w-fit px-5 text-[13px]">
          {t("common.save")}
        </Button>
      </form>
    </Form>
  );
}
