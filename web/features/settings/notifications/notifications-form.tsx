"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Check, ExternalLink, Loader2, Lock, MonitorSmartphone, Send } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { displayTimeZone } from "@/lib/timezone";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { groupIcon } from "@/components/notifications/group-icons";
import {
  notificationErrorText,
  useNotificationPrefs,
  useUpdateNotificationPrefs,
} from "@/hooks/use-notifications";
import { useMe } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import {
  checkTelegramLink,
  startTelegramLink,
  testNotificationChannel,
  type NotificationChannel,
  type NotificationPrefs,
  type NotificationPrefsUpdate,
} from "@/lib/api";
import {
  enableBrowserNotifications,
  setBrowserNotificationsEnabled,
  useBrowserNotifications,
} from "@/lib/browser-notifications";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";

const CHANNELS: { id: NotificationChannel; label: string; icon: string }[] = [
  { id: "email", label: "Email", icon: "ri-mail-line" },
  { id: "telegram", label: "Telegram", icon: "ri-telegram-line" },
  { id: "discord", label: "Discord", icon: "ri-discord-line" },
];

type Save = (payload: NotificationPrefsUpdate, success?: string) => void;

export function NotificationsForm() {
  const t = useT();
  const prefs = useNotificationPrefs();
  const update = useUpdateNotificationPrefs();

  if (prefs.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-[260px] rounded-2xl" />
        <Skeleton className="h-[300px] rounded-2xl" />
      </div>
    );
  }

  if (prefs.isError || !prefs.data) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 px-5 py-4 text-sm text-destructive">
        <span>{t("notifications.prefs.load_failed")}</span>
        <button type="button" onClick={() => void prefs.refetch()} className={cn(btnGhost, "h-8 px-3")}>
          {t("settings.common.retry")}
        </button>
      </div>
    );
  }

  const save: Save = (payload, success) =>
    update.mutate(payload, {
      onSuccess: () => {
        if (success) toast.success(success);
      },
      onError: (error) => toast.error(notificationErrorText(error)),
    });

  return (
    <div className="flex flex-col gap-4">
      <SettingsCard
        title={t("notifications.prefs.channels_title")}
        desc={t("notifications.prefs.channels_desc")}
      >
        <div className="flex flex-col divide-y divide-border">
          <EmailChannel prefs={prefs.data} save={save} />
          <TelegramChannel prefs={prefs.data} save={save} />
          <DiscordChannel prefs={prefs.data} save={save} />
        </div>
      </SettingsCard>

      <SettingsCard title={t("notifications.prefs.matrix_title")} desc={t("notifications.prefs.matrix_desc")}>
        <RoutesMatrix prefs={prefs.data} save={save} />
      </SettingsCard>

      <SettingsCard title={t("notifications.prefs.quiet_title")} desc={t("notifications.prefs.quiet_desc")}>
        <QuietHours prefs={prefs.data} save={save} pending={update.isPending} />
      </SettingsCard>

      <SettingsCard title={t("notifications.prefs.balance_title")} desc={t("notifications.prefs.balance_desc")}>
        <BalanceThreshold prefs={prefs.data} save={save} pending={update.isPending} />
      </SettingsCard>

      <SettingsCard title={t("notifications.prefs.device_title")} desc={t("notifications.prefs.device_desc")}>
        <DeviceNotifications />
      </SettingsCard>
    </div>
  );
}

function SettingsCard({ title, desc, children }: { title: string; desc: string; children: ReactNode }) {
  return (
    <section className="rounded-2xl border border-border bg-card p-5">
      <div className="mb-4 space-y-1">
        <h3 className="text-[15px] leading-none font-semibold">{title}</h3>
        <p className="text-[12.5px] text-muted-foreground">{desc}</p>
      </div>
      {children}
    </section>
  );
}

type ChannelStatus = "on" | "off" | "missing";

function StatusBadge({ status }: { status: ChannelStatus }) {
  const t = useT();
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-px text-[11px] font-medium",
        status === "on"
          ? "bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]"
          : "bg-muted text-muted-foreground"
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          status === "on" ? "bg-[var(--vx-ok)]" : "bg-muted-foreground/50"
        )}
      />
      {t(`notifications.channel_state.${status}`)}
    </span>
  );
}

function ChannelRow({
  icon,
  label,
  status,
  detail,
  actions,
  children,
}: {
  icon: string;
  label: string;
  status: ChannelStatus;
  detail: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 py-4 first:pt-0 last:pb-0">
      <div className="flex flex-wrap items-center gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-[18px] text-foreground/80">
          <i className={icon} />
        </span>
        <div className="min-w-[180px] flex-1">
          <div className="flex flex-wrap items-center gap-2 text-[14px] font-medium">
            {label}
            <StatusBadge status={status} />
          </div>
          <div className="mt-0.5 truncate text-[12.5px] text-muted-foreground">{detail}</div>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </div>
      {children}
    </div>
  );
}

function TestButton({ channel, disabled }: { channel: NotificationChannel; disabled?: boolean }) {
  const t = useT();
  const test = useMutation({
    mutationFn: () => testNotificationChannel(channel),
    onSuccess: () => toast.success(t("notifications.prefs.test_sent")),
    onError: (error) => toast.error(notificationErrorText(error)),
  });
  return (
    <button
      type="button"
      onClick={() => test.mutate()}
      disabled={disabled || test.isPending}
      className={cn(btnGhost, "h-8 px-3")}
    >
      {test.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Send className="size-3.5" />}
      {t("notifications.prefs.test")}
    </button>
  );
}

function EmailChannel({ prefs, save }: { prefs: NotificationPrefs; save: Save }) {
  const t = useT();
  const { data: me } = useMe();
  const enabled = prefs.channels.email;
  return (
    <ChannelRow
      icon="ri-mail-line"
      label="Email"
      status={enabled ? "on" : "off"}
      detail={me?.email ?? "—"}
      actions={
        <>
          <TestButton channel="email" disabled={!enabled} />
          <Switch
            checked={enabled}
            aria-label="Email"
            onCheckedChange={(on) => save({ email: on })}
          />
        </>
      }
    />
  );
}

type TelegramLinkState = { code: string; url: string; expiresAt: number };

function TelegramChannel({ prefs, save }: { prefs: NotificationPrefs; save: Save }) {
  const t = useT();
  const qc = useQueryClient();
  const { available, bot_username: bot } = prefs.telegram;
  const chatId = prefs.channels.telegram_chat_id;
  const connected = chatId !== "";
  const enabled = connected && prefs.channels.telegram;
  const [link, setLink] = useState<TelegramLinkState | null>(null);
  const [manual, setManual] = useState(false);
  const [draft, setDraft] = useState(chatId);

  const start = useMutation({
    mutationFn: startTelegramLink,
    onSuccess: (res) =>
      setLink({ code: res.code, url: res.url, expiresAt: new Date(res.expires_at).getTime() }),
    onError: (error) => toast.error(notificationErrorText(error)),
  });

  useEffect(() => {
    if (!link) return;
    let stopped = false;
    let timer = 0;
    const tick = async () => {
      if (stopped) return;
      if (Date.now() > link.expiresAt) {
        setLink(null);
        toast.error(t("notifications.prefs.telegram_expired"));
        return;
      }
      try {
        const res = await checkTelegramLink(link.code);
        if (stopped) return;
        if (res.status === "linked") {
          setLink(null);
          setManual(false);
          toast.success(
            res.chat
              ? t("notifications.prefs.telegram_linked_chat", { chat: res.chat })
              : t("notifications.prefs.telegram_linked")
          );
          void qc.invalidateQueries({ queryKey: queryKeys.notificationPrefs });
          return;
        }
        if (res.status === "expired") {
          setLink(null);
          toast.error(t("notifications.prefs.telegram_expired"));
          return;
        }
        if (res.status === "unavailable") {
          setLink(null);
          setManual(true);
          toast.error(t("notifications.prefs.telegram_webhook"));
          return;
        }
      } catch {}
      timer = window.setTimeout(tick, 3_000);
    };
    timer = window.setTimeout(tick, 2_000);
    return () => {
      stopped = true;
      window.clearTimeout(timer);
    };
  }, [link, qc, t]);

  const saveManual = (value: string) => {
    save({ telegram_chat_id: value, telegram: true }, t("notifications.prefs.saved"));
    setManual(false);
  };

  const detail = !available
    ? t("notifications.prefs.telegram_unavailable")
    : connected
      ? t("notifications.prefs.telegram_connected", { chat: chatId })
      : t("notifications.prefs.telegram_hint");

  return (
    <ChannelRow
      icon="ri-telegram-line"
      label="Telegram"
      status={enabled ? "on" : connected ? "off" : "missing"}
      detail={detail}
      actions={
        available ? (
          connected ? (
            <>
              <TestButton channel="telegram" disabled={!enabled} />
              <DisconnectButton
                channel="Telegram"
                onConfirm={() =>
                  save(
                    { telegram: false, telegram_chat_id: "" },
                    t("notifications.prefs.telegram_disconnected")
                  )
                }
              />
              <Switch
                checked={enabled}
                aria-label="Telegram"
                onCheckedChange={(on) => save({ telegram: on })}
              />
            </>
          ) : (
            !link && (
              <button
                type="button"
                onClick={() => start.mutate()}
                disabled={start.isPending || !bot}
                className={cn(btnPrimary, "h-8 px-3.5")}
              >
                {start.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Send className="size-3.5" />}
                {t("notifications.prefs.telegram_connect")}
              </button>
            )
          )
        ) : undefined
      }
    >
      {link && (
        <div className="flex flex-wrap items-center gap-3 rounded-xl border border-dashed border-input bg-muted/40 p-3.5">
          <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" />
          <div className="min-w-[200px] flex-1 text-[12.5px] leading-[1.5]">
            <div className="font-medium text-foreground">
              {t("notifications.prefs.telegram_wait_title", { bot: bot ? `@${bot}` : "" })}
            </div>
            <div className="text-muted-foreground">{t("notifications.prefs.telegram_wait_text")}</div>
          </div>
          <a href={link.url} target="_blank" rel="noopener noreferrer" className={cn(btnPrimary, "h-8 px-3.5")}>
            <ExternalLink className="size-3.5" />
            {t("notifications.prefs.telegram_open")}
          </a>
          <button
            type="button"
            onClick={() => setLink(null)}
            className="text-[12.5px] text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("common.cancel")}
          </button>
        </div>
      )}
      {available && !link && (
        <div>
          {!manual ? (
            <button
              type="button"
              onClick={() => {
                setDraft(chatId);
                setManual(true);
              }}
              className="text-[12.5px] text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
            >
              {connected ? t("notifications.prefs.telegram_change") : t("notifications.prefs.telegram_manual")}
            </button>
          ) : (
            <form
              className="flex flex-wrap items-center gap-2"
              onSubmit={(event) => {
                event.preventDefault();
                const value = draft.trim();
                if (!value) return;
                saveManual(value);
              }}
            >
              <input
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                placeholder="-1001234567890"
                aria-label="Chat ID"
                className={cn(fieldClass, "h-9 max-w-[260px] font-mono")}
              />
              <button type="submit" disabled={!draft.trim()} className={cn(btnGhost, "h-9")}>
                {t("common.save")}
              </button>
              <button
                type="button"
                onClick={() => setManual(false)}
                className="text-[12.5px] text-muted-foreground transition-colors hover:text-foreground"
              >
                {t("common.cancel")}
              </button>
              <p className="basis-full text-[12px] text-muted-foreground">
                {t("notifications.prefs.telegram_manual_hint", { bot: bot ? `@${bot}` : "" })}
              </p>
            </form>
          )}
        </div>
      )}
    </ChannelRow>
  );
}

function maskWebhook(url: string): string {
  const tail = url.slice(-4);
  return `discord.com/api/webhooks/…${tail}`;
}

function DiscordChannel({ prefs, save }: { prefs: NotificationPrefs; save: Save }) {
  const t = useT();
  const webhook = prefs.channels.discord_webhook;
  const connected = webhook !== "";
  const enabled = connected && prefs.channels.discord;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const showForm = !connected || editing;

  return (
    <ChannelRow
      icon="ri-discord-line"
      label="Discord"
      status={enabled ? "on" : connected ? "off" : "missing"}
      detail={connected ? maskWebhook(webhook) : t("notifications.prefs.discord_hint")}
      actions={
        connected ? (
          <>
            <TestButton channel="discord" disabled={!enabled} />
            <DisconnectButton
              channel="Discord"
              onConfirm={() => save({ discord: false, discord_webhook: "" }, t("notifications.prefs.discord_removed"))}
            />
            <Switch checked={enabled} aria-label="Discord" onCheckedChange={(on) => save({ discord: on })} />
          </>
        ) : undefined
      }
    >
      {showForm ? (
        <form
          className="flex flex-wrap items-center gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            const value = draft.trim();
            if (!value) return;
            save({ discord_webhook: value, discord: true }, t("notifications.prefs.saved"));
            setDraft("");
            setEditing(false);
          }}
        >
          <input
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="https://discord.com/api/webhooks/…"
            aria-label="Webhook URL"
            className={cn(fieldClass, "h-9 min-w-[220px] flex-1 font-mono")}
          />
          <button type="submit" disabled={!draft.trim()} className={cn(btnGhost, "h-9")}>
            {connected ? t("common.save") : t("notifications.prefs.discord_connect")}
          </button>
          {editing && (
            <button
              type="button"
              onClick={() => setEditing(false)}
              className="text-[12.5px] text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("common.cancel")}
            </button>
          )}
        </form>
      ) : (
        <button
          type="button"
          onClick={() => setEditing(true)}
          className="self-start text-[12.5px] text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
        >
          {t("notifications.prefs.discord_change")}
        </button>
      )}
    </ChannelRow>
  );
}

function RoutesMatrix({ prefs, save }: { prefs: NotificationPrefs; save: Save }) {
  const t = useT();
  const c = prefs.channels;
  const active: Record<NotificationChannel, boolean> = {
    email: c.email,
    telegram: c.telegram && c.telegram_chat_id !== "",
    discord: c.discord && c.discord_webhook !== "",
  };

  const cell = (groupId: string, channel: NotificationChannel, label: string) => {
    const group = prefs.groups.find((g) => g.id === groupId);
    if (!group || !group.channels.includes(channel)) {
      return <span className="text-muted-foreground/50">—</span>;
    }
    const locked = group.locked.includes(channel);
    const checked = locked || (active[channel] && group.routes[channel] !== false);
    const hint = locked
      ? t("notifications.prefs.locked")
      : !active[channel]
        ? t("notifications.prefs.channel_inactive")
        : undefined;
    return (
      <span title={hint} className="relative inline-flex items-center">
        {locked && (
          <Lock className="absolute top-1/2 -left-4 size-3 -translate-y-1/2 text-muted-foreground" />
        )}
        <Switch
          checked={checked}
          disabled={locked || !active[channel]}
          aria-label={`${group.label}: ${label}`}
          onCheckedChange={(on) => save({ routes: { [group.id]: { [channel]: on } } })}
        />
      </span>
    );
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="hidden overflow-hidden rounded-xl border border-border sm:block">
        <table className="w-full text-[13px]">
          <thead className="bg-muted/50 text-[12px] text-muted-foreground">
            <tr>
              <th className="px-4 py-2.5 text-left font-medium">{t("notifications.prefs.category")}</th>
              <th className="w-[84px] px-2 py-2.5 font-medium">{t("notifications.prefs.panel")}</th>
              {CHANNELS.map((channel) => (
                <th key={channel.id} className="w-[96px] px-2 py-2.5 font-medium">
                  <span className="inline-flex items-center gap-1.5">
                    <i className={channel.icon} />
                    {channel.label}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {prefs.groups.map((group) => {
              const Icon = groupIcon(group.id);
              return (
                <tr key={group.id}>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <Icon className="size-4" />
                      </span>
                      <div className="min-w-0">
                        <div className="font-medium">{group.label}</div>
                        <div className="text-[12px] text-muted-foreground">
                          {t(`notifications.prefs.group.${group.id}`)}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-2 py-3 text-center">
                    <Check className="mx-auto size-4 text-[var(--vx-ok)]" />
                  </td>
                  {CHANNELS.map((channel) => (
                    <td key={channel.id} className="px-2 py-3 text-center">
                      {cell(group.id, channel.id, channel.label)}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="flex flex-col gap-2 sm:hidden">
        {prefs.groups.map((group) => {
          const Icon = groupIcon(group.id);
          return (
            <div key={group.id} className="rounded-xl border border-border p-3.5">
              <div className="flex items-center gap-3">
                <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                  <Icon className="size-4" />
                </span>
                <div className="min-w-0">
                  <div className="text-[13.5px] font-medium">{group.label}</div>
                  <div className="text-[12px] text-muted-foreground">
                    {t(`notifications.prefs.group.${group.id}`)}
                  </div>
                </div>
              </div>
              <div className="mt-3 grid grid-cols-3 gap-2">
                {CHANNELS.map((channel) => (
                  <div
                    key={channel.id}
                    className="flex flex-col items-center gap-1.5 rounded-lg bg-muted/50 px-2 py-2 text-[11.5px] text-muted-foreground"
                  >
                    <span className="inline-flex items-center gap-1">
                      <i className={channel.icon} />
                      {channel.label}
                    </span>
                    {cell(group.id, channel.id, channel.label)}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <p className="text-[12px] leading-[1.5] text-muted-foreground">{t("notifications.prefs.matrix_hint")}</p>
    </div>
  );
}

function DeviceNotifications() {
  const t = useT();
  const state = useBrowserNotifications();

  const text = !state.supported
    ? t("notifications.browser_unsupported")
    : state.permission === "denied"
      ? t("notifications.browser_denied")
      : t("notifications.browser_text");

  return (
    <div className="flex items-center gap-3 rounded-xl border border-border px-4 py-3.5">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
        <MonitorSmartphone className="size-[18px]" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="text-[14px] font-medium">{t("notifications.browser_title")}</div>
        <div className="text-[12.5px] text-muted-foreground">{text}</div>
      </div>
      <Switch
        checked={state.enabled}
        disabled={!state.supported || state.permission === "denied"}
        aria-label={t("notifications.browser_title")}
        onCheckedChange={async (on) => {
          if (!on) {
            setBrowserNotificationsEnabled(false);
            return;
          }
          const permission = await enableBrowserNotifications();
          if (permission === "granted") toast.success(t("notifications.browser_enabled"));
          else if (permission === "denied") toast.error(t("notifications.browser_denied"));
        }}
      />
    </div>
  );
}

function DisconnectButton({ channel, onConfirm }: { channel: string; onConfirm: () => void }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)} className={cn(btnGhost, "h-8 px-3")}>
        {t("notifications.prefs.disconnect")}
      </button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={t("notifications.prefs.disconnect_title", { channel })}
        desc={t("notifications.prefs.disconnect_hint", { channel })}
        destructive
        cancelBtnText={t("common.cancel")}
        confirmText={t("notifications.prefs.disconnect")}
        handleConfirm={() => {
          setOpen(false);
          onConfirm();
        }}
      />
    </>
  );
}

function toTime(minutes: number): string {
  const m = ((minutes % 1440) + 1440) % 1440;
  return `${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`;
}

function fromTime(value: string): number {
  const [h, m] = value.split(":").map((v) => Number(v));
  if (!Number.isFinite(h) || !Number.isFinite(m)) return 0;
  return Math.min(1439, Math.max(0, h * 60 + m));
}

function QuietHours({ prefs, save, pending }: { prefs: NotificationPrefs; save: Save; pending: boolean }) {
  const t = useT();
  const initial = prefs.quiet ?? { enabled: false, from: 1380, to: 480, critical: true, tz: "" };
  const [form, setForm] = useState(initial);
  const key = JSON.stringify(initial);
  useEffect(() => {
    setForm(JSON.parse(key) as typeof initial);
  }, [key]);
  const zone = displayTimeZone();
  const dirty = JSON.stringify({ ...form, tz: "" }) !== JSON.stringify({ ...initial, tz: "" }) || (form.enabled && initial.tz !== zone);
  const same = form.from === form.to;

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (form.enabled && same) {
          toast.error(t("notifications.prefs.quiet_same"));
          return;
        }
        save({ quiet: { ...form, tz: zone } }, t("notifications.prefs.quiet_saved"));
      }}
    >
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-[14px] font-medium">{t("notifications.prefs.quiet_enable")}</div>
          <div className="text-[12.5px] text-muted-foreground">{t("notifications.prefs.quiet_zone", { zone: zone.replace(/_/g, " ") })}</div>
        </div>
        <Switch
          checked={form.enabled}
          aria-label={t("notifications.prefs.quiet_enable")}
          onCheckedChange={(on) => setForm({ ...form, enabled: on })}
        />
      </div>
      <div className={cn("grid gap-4 sm:grid-cols-2", !form.enabled && "opacity-60")}>
        <label className="space-y-1.5">
          <span className="text-[13px] font-medium">{t("notifications.prefs.quiet_from")}</span>
          <input
            type="time"
            step={900}
            disabled={!form.enabled}
            value={toTime(form.from)}
            onChange={(e) => setForm({ ...form, from: fromTime(e.target.value) })}
            className={cn(fieldClass, "h-9")}
          />
        </label>
        <label className="space-y-1.5">
          <span className="text-[13px] font-medium">{t("notifications.prefs.quiet_to")}</span>
          <input
            type="time"
            step={900}
            disabled={!form.enabled}
            value={toTime(form.to)}
            onChange={(e) => setForm({ ...form, to: fromTime(e.target.value) })}
            className={cn(fieldClass, "h-9")}
          />
        </label>
      </div>
      <div className={cn("flex items-center justify-between gap-3", !form.enabled && "opacity-60")}>
        <div>
          <div className="text-[13.5px] font-medium">{t("notifications.prefs.quiet_critical")}</div>
          <div className="text-[12.5px] text-muted-foreground">{t("notifications.prefs.quiet_critical_hint")}</div>
        </div>
        <Switch
          checked={form.critical}
          disabled={!form.enabled}
          aria-label={t("notifications.prefs.quiet_critical")}
          onCheckedChange={(on) => setForm({ ...form, critical: on })}
        />
      </div>
      <button type="submit" disabled={!dirty || pending} className={cn(btnPrimary, "h-9 px-4")}>
        {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Check className="size-3.5" />}
        {t("common.save")}
      </button>
    </form>
  );
}

function BalanceThreshold({ prefs, save, pending }: { prefs: NotificationPrefs; save: Save; pending: boolean }) {
  const t = useT();
  const stored = prefs.balance_threshold ?? null;
  const [enabled, setEnabled] = useState(stored !== null);
  const [amount, setAmount] = useState(stored !== null ? String(stored) : "");
  useEffect(() => {
    setEnabled(stored !== null);
    setAmount(stored !== null ? String(stored) : "");
  }, [stored]);
  const value = Number(amount.replace(",", "."));
  const valid = Number.isFinite(value) && value > 0 && value <= 10_000_000;
  const dirty = enabled ? !valid || value !== stored : stored !== null;

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (enabled && !valid) {
          toast.error(t("notifications.prefs.balance_invalid"));
          return;
        }
        save({ balance_threshold: enabled ? value : null }, t("notifications.prefs.balance_saved"));
      }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="text-[14px] font-medium">{t("notifications.prefs.balance_enable")}</div>
        <Switch
          checked={enabled}
          aria-label={t("notifications.prefs.balance_enable")}
          onCheckedChange={setEnabled}
        />
      </div>
      <label className={cn("block max-w-xs space-y-1.5", !enabled && "opacity-60")}>
        <span className="text-[13px] font-medium">{t("notifications.prefs.balance_amount")}</span>
        <input
          inputMode="decimal"
          disabled={!enabled}
          value={amount}
          placeholder="500"
          onChange={(e) => setAmount(e.target.value.replace(/[^\d.,]/g, ""))}
          className={cn(fieldClass, "h-9")}
        />
      </label>
      <button type="submit" disabled={!dirty || pending} className={cn(btnPrimary, "h-9 px-4")}>
        {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Check className="size-3.5" />}
        {t("common.save")}
      </button>
    </form>
  );
}
