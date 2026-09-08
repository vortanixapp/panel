"use client";

import { useEffect } from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchNotificationChannels,
  updateNotificationChannels,
  type NotificationChannels,
} from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

// Схема собирается внутри компонента: сообщения об ошибках должны меняться
// вместе с языком, а модульная константа заморозила бы их на первом рендере.
const buildSchema = (t: TranslateFn) =>
  z
    .object({
      email: z.boolean(),
      telegram: z.boolean(),
      discord: z.boolean(),
      telegram_chat_id: z.string().max(64, t("settings.notifications.chat_id_max")),
      discord_webhook: z.string().max(300, t("settings.notifications.webhook_max")),
    })
    .refine((v) => !v.telegram || v.telegram_chat_id.trim() !== "", {
      path: ["telegram_chat_id"],
      message: t("settings.notifications.chat_id_required"),
    })
    .refine((v) => !v.discord || v.discord_webhook.trim() !== "", {
      path: ["discord_webhook"],
      message: t("settings.notifications.webhook_required"),
    });

type NotificationsFormValues = z.infer<
  ReturnType<typeof buildSchema>
>;

const EMPTY: NotificationsFormValues = {
  email: true,
  telegram: false,
  discord: false,
  telegram_chat_id: "",
  discord_webhook: "",
};

export function NotificationsForm() {
  const t = useT();
  const qc = useQueryClient();

  const query = useQuery({
    queryKey: ["notification-channels"],
    queryFn: fetchNotificationChannels,
  });

  const form = useForm<NotificationsFormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: EMPTY,
  });

  const channels = query.data?.channels;

  useEffect(() => {
    if (!channels) return;
    form.reset({
      email: channels.email,
      telegram: channels.telegram,
      discord: channels.discord,
      telegram_chat_id: channels.telegram_chat_id ?? "",
      discord_webhook: channels.discord_webhook ?? "",
    });
  }, [form, channels]);

  const save = useMutation({
    mutationFn: (data: NotificationsFormValues) =>
      updateNotificationChannels(data as Partial<NotificationChannels>),
    onSuccess: () => {
      toast.success(t("settings.notifications.saved"));
      qc.invalidateQueries({ queryKey: ["notification-channels"] });
      qc.invalidateQueries({ queryKey: ["notifications"] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  if (query.isLoading) {
    return <Skeleton className="h-64 w-full rounded-2xl" />;
  }

  if (query.isError) {
    return (
      <p className="text-sm text-destructive">
        {t("settings.notifications.load_failed")}
      </p>
    );
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit((data) => save.mutate(data))}
        className="flex max-w-[720px] flex-col gap-4"
      >
        <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
          <div className="space-y-1">
            <div className="text-[15px] leading-none font-semibold">
              {t("settings.notifications.channels_title")}
            </div>
            <div className="text-xs text-muted-foreground">
              {t("settings.notifications.channels_hint")}
            </div>
          </div>

          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem className="flex flex-row items-center justify-between gap-4 rounded-xl border bg-muted/30 px-4 py-3.5">
                <div className="space-y-0.5">
                  <FormLabel className="text-[13px] font-medium">Email</FormLabel>
                  <FormDescription className="text-xs">
                    {t("settings.notifications.email_desc")}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                </FormControl>
              </FormItem>
            )}
          />

          <div className="flex flex-col gap-3 rounded-xl border bg-muted/30 p-4">
            <FormField
              control={form.control}
              name="telegram"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between gap-4">
                  <div className="space-y-0.5">
                    <FormLabel className="text-[13px] font-medium">Telegram</FormLabel>
                    <FormDescription className="text-xs">
                      {t("settings.notifications.telegram_desc")}
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
            {form.watch("telegram") && (
              <FormField
                control={form.control}
                name="telegram_chat_id"
                render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-xs font-normal text-muted-foreground">
                      Chat ID
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder="-1001234567890"
                        className="h-9 rounded-lg font-mono text-[13px] md:text-[13px]"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage className="text-xs" />
                  </FormItem>
                )}
              />
            )}
          </div>

          <div className="flex flex-col gap-3 rounded-xl border bg-muted/30 p-4">
            <FormField
              control={form.control}
              name="discord"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between gap-4">
                  <div className="space-y-0.5">
                    <FormLabel className="text-[13px] font-medium">Discord</FormLabel>
                    <FormDescription className="text-xs">
                      {t("settings.notifications.discord_desc")}
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
            {form.watch("discord") && (
              <FormField
                control={form.control}
                name="discord_webhook"
                render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-xs font-normal text-muted-foreground">
                      Webhook URL
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder="https://discord.com/api/webhooks/…"
                        className="h-9 rounded-lg font-mono text-[13px] md:text-[13px]"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage className="text-xs" />
                  </FormItem>
                )}
              />
            )}
          </div>
        </section>

        <Button
          type="submit"
          className="h-[38px] w-fit px-5 text-[13px]"
          disabled={save.isPending}
        >
          {save.isPending ? t("common.saving") : t("common.save")}
        </Button>
      </form>
    </Form>
  );
}
