"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useMutation } from "@tanstack/react-query";
import { Check, ChevronsUpDown, Copy, Loader2, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useLocale } from "@/context/locale-provider";
import { useAccountQuery, useSetAccount } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import {
  deleteAccountAvatar,
  updateAccount,
  uploadAccountAvatar,
  type AccountUser,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import {
  deviceTimeZone,
  displayTimeZone,
  listTimeZones,
  setDisplayTimeZone,
  timeZoneOffsetLabel,
  timeZoneOffsetMinutes,
} from "@/lib/timezone";
import { setAccountPreferences } from "@/lib/user-preferences";
import { cn } from "@/lib/utils";
import {
  errorText,
  Field,
  FormActions,
  formatDate,
  formatDateTime,
  QueryState,
  SettingsSection,
  useCopy,
  useUnsavedGuard,
} from "./ui";

const AVATAR_MAX = 4 * 1024 * 1024;
const AVATAR_TYPES = ["image/jpeg", "image/png", "image/webp"];

type NameForm = { first_name: string; last_name: string; display_name: string };

function namesOf(user: AccountUser): NameForm {
  return {
    first_name: user.first_name ?? "",
    last_name: user.last_name ?? "",
    display_name: user.display_name ?? "",
  };
}

export function ProfileTab() {
  const q = useAccountQuery();
  return (
    <QueryState isLoading={q.isLoading} isError={q.isError} error={q.error} onRetry={() => void q.refetch()} rows={3}>
      {q.data && (
        <>
          <ProfileCard user={q.data.user} />
          <RegionCard user={q.data.user} />
          <AccountCard user={q.data.user} />
        </>
      )}
    </QueryState>
  );
}

function ProfileCard({ user }: { user: AccountUser }) {
  const t = useT();
  const setAccount = useSetAccount();
  const [form, setForm] = useState<NameForm>(() => namesOf(user));
  const [base, setBase] = useState<NameForm>(() => namesOf(user));
  const [dragging, setDragging] = useState(false);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const baseRef = useRef(base);
  baseRef.current = base;
  const dirty = JSON.stringify(form) !== JSON.stringify(base);
  const userKey = JSON.stringify(namesOf(user));
  useUnsavedGuard(dirty);

  useEffect(() => {
    const next = JSON.parse(userKey) as NameForm;
    setForm((current) => (JSON.stringify(current) === JSON.stringify(baseRef.current) ? next : current));
    setBase(next);
  }, [userKey]);

  const save = useMutation({
    mutationFn: () =>
      updateAccount({
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        display_name: form.display_name.trim(),
      }),
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.profile.saved"));
    },
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  const upload = useMutation({
    mutationFn: uploadAccountAvatar,
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.profile.avatar_updated"));
    },
    onError: (err) => toast.error(errorText(err, t("settings.profile.avatar_failed"))),
    onSettled: () => {
      if (fileRef.current) fileRef.current.value = "";
    },
  });

  const remove = useMutation({
    mutationFn: deleteAccountAvatar,
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.profile.avatar_removed"));
    },
    onError: (err) => toast.error(errorText(err, t("common.delete_failed"))),
  });

  const pick = (file: File | undefined) => {
    if (!file) return;
    if (!AVATAR_TYPES.includes(file.type)) {
      toast.error(t("settings.profile.avatar_type"));
      return;
    }
    if (file.size > AVATAR_MAX) {
      toast.error(t("settings.profile.avatar_too_big"));
      return;
    }
    upload.mutate(file);
  };

  const shownName =
    form.display_name.trim() ||
    [form.first_name.trim(), form.last_name.trim()].filter(Boolean).join(" ") ||
    user.email.split("@")[0];
  const initial = shownName.charAt(0).toUpperCase() || "U";
  const busy = upload.isPending || remove.isPending;

  return (
    <SettingsSection title={t("settings.profile.title")} description={t("settings.profile.hint")}>
      <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
        <button
          type="button"
          onClick={() => fileRef.current?.click()}
          onDragOver={(e) => {
            e.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragging(false);
            pick(e.dataTransfer.files?.[0]);
          }}
          aria-label={t("settings.profile.avatar_upload")}
          className={cn(
            "group relative size-20 shrink-0 overflow-hidden rounded-full border-2 border-dashed bg-muted transition-colors",
            dragging ? "border-primary" : "border-transparent hover:border-border"
          )}
        >
          {user.avatar_url ? (
            <img src={user.avatar_url} alt={shownName} className="size-full object-cover" />
          ) : (
            <span className="flex size-full items-center justify-center text-2xl font-semibold">{initial}</span>
          )}
          <span className="absolute inset-0 flex items-center justify-center bg-black/45 text-white opacity-0 transition-opacity group-hover:opacity-100">
            {busy ? <Loader2 className="size-5 animate-spin" /> : <Upload className="size-5" />}
          </span>
        </button>
        <div className="min-w-0 space-y-2">
          <div className="truncate text-[15px] font-semibold">{shownName}</div>
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" variant="outline" disabled={busy} onClick={() => fileRef.current?.click()}>
              <Upload className="size-3.5" />
              {user.avatar_url ? t("settings.profile.avatar_replace") : t("settings.profile.avatar_upload")}
            </Button>
            {user.avatar_url && (
              <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => remove.mutate()}>
                <Trash2 className="size-3.5" />
                {t("settings.profile.avatar_remove")}
              </Button>
            )}
          </div>
          <p className="text-[12px] text-muted-foreground">{t("settings.profile.avatar_hint")}</p>
        </div>
        <input
          ref={fileRef}
          type="file"
          accept={AVATAR_TYPES.join(",")}
          className="sr-only"
          onChange={(e) => pick(e.target.files?.[0])}
        />
      </div>

      <form
        className="mt-6"
        onSubmit={(e) => {
          e.preventDefault();
          if (dirty) save.mutate();
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("settings.profile.first_name")} htmlFor="first-name">
            <Input
              id="first-name"
              autoComplete="given-name"
              maxLength={64}
              value={form.first_name}
              onChange={(e) => setForm({ ...form, first_name: e.target.value })}
            />
          </Field>
          <Field label={t("settings.profile.last_name")} htmlFor="last-name">
            <Input
              id="last-name"
              autoComplete="family-name"
              maxLength={64}
              value={form.last_name}
              onChange={(e) => setForm({ ...form, last_name: e.target.value })}
            />
          </Field>
          <Field
            className="sm:col-span-2"
            label={t("settings.profile.display_name")}
            htmlFor="display-name"
            hint={t("settings.profile.display_name_hint")}
          >
            <Input
              id="display-name"
              autoComplete="nickname"
              maxLength={64}
              placeholder={[form.first_name, form.last_name].filter(Boolean).join(" ") || user.email.split("@")[0]}
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
            />
          </Field>
        </div>
        <FormActions dirty={dirty} pending={save.isPending} onReset={() => setForm(base)} />
      </form>
    </SettingsSection>
  );
}

function RegionCard({ user }: { user: AccountUser }) {
  const t = useT();
  const setAccount = useSetAccount();
  const { locale, languages } = useLocale();
  const [open, setOpen] = useState(false);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 30_000);
    return () => window.clearInterval(timer);
  }, []);

  const zones = useMemo(() => {
    const at = new Date();
    return listTimeZones()
      .map((zone) => ({ zone, offset: timeZoneOffsetMinutes(zone, at), label: timeZoneOffsetLabel(zone, at) }))
      .sort((a, b) => a.offset - b.offset || a.zone.localeCompare(b.zone));
  }, []);

  const saveLocale = useMutation({
    mutationFn: (next: string) => updateAccount({ locale: next }),
    onSuccess: (res) => setAccount(res.user),
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  const saveZone = useMutation({
    mutationFn: (next: string) => updateAccount({ timezone: next }),
    onSuccess: (res) => {
      setAccount(res.user);
      setDisplayTimeZone(res.user.timezone ?? "");
      setNow(new Date());
      toast.success(t("settings.region.zone_saved"));
    },
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  const zone = user.timezone ?? "";
  const effective = zone || deviceTimeZone();
  const preview = now.toLocaleString(localeTag(), {
    timeZone: displayTimeZone(),
    weekday: "short",
    day: "numeric",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <SettingsSection title={t("settings.region.title")} description={t("settings.region.hint")}>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label={t("settings.region.language")} htmlFor="locale-select" hint={t("settings.region.language_hint")}>
          <select
            id="locale-select"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
            value={locale}
            disabled={saveLocale.isPending}
            onChange={(e) => {
              setAccountPreferences({ language: e.target.value });
              saveLocale.mutate(e.target.value);
            }}
          >
            {languages.map((language) => (
              <option key={language.code} value={language.code}>
                {language.name}
              </option>
            ))}
          </select>
        </Field>
        <Field
          label={t("settings.region.timezone")}
          hint={t("settings.region.now", { time: preview, zone: effective.replace(/_/g, " ") })}
        >
          <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
              <Button
                type="button"
                variant="outline"
                role="combobox"
                aria-expanded={open}
                disabled={saveZone.isPending}
                className="h-9 w-full justify-between px-3 font-normal"
              >
                <span className="truncate">
                  {zone
                    ? `${zone.replace(/_/g, " ")} (${timeZoneOffsetLabel(zone)})`
                    : t("settings.region.zone_auto", { zone: deviceTimeZone().replace(/_/g, " ") })}
                </span>
                {saveZone.isPending ? (
                  <Loader2 className="size-4 shrink-0 animate-spin opacity-60" />
                ) : (
                  <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-[min(calc(100vw-2rem),360px)] p-0" align="start">
              <Command>
                <CommandInput placeholder={t("settings.region.zone_search")} />
                <CommandList className="max-h-72">
                  <CommandEmpty>{t("settings.region.zone_empty")}</CommandEmpty>
                  <CommandGroup>
                    <CommandItem
                      value={`auto ${t("settings.region.zone_auto_short")}`}
                      onSelect={() => {
                        setOpen(false);
                        if (zone) saveZone.mutate("");
                      }}
                    >
                      <Check className={cn("size-4", zone ? "opacity-0" : "opacity-100")} />
                      {t("settings.region.zone_auto", { zone: deviceTimeZone().replace(/_/g, " ") })}
                    </CommandItem>
                    {zones.map((item) => (
                      <CommandItem
                        key={item.zone}
                        value={`${item.zone} ${item.label}`}
                        onSelect={() => {
                          setOpen(false);
                          if (item.zone !== zone) saveZone.mutate(item.zone);
                        }}
                      >
                        <Check className={cn("size-4", item.zone === zone ? "opacity-100" : "opacity-0")} />
                        <span className="truncate">{item.zone.replace(/_/g, " ")}</span>
                        <span className="ml-auto font-mono text-[11.5px] text-muted-foreground">{item.label}</span>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        </Field>
      </div>
    </SettingsSection>
  );
}

function AccountCard({ user }: { user: AccountUser }) {
  const t = useT();
  const { copied, copy } = useCopy();
  const rows: { label: string; value: React.ReactNode }[] = [
    {
      label: t("settings.account_card.id"),
      value: (
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate font-mono text-[12.5px]">{user.id}</span>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-7 shrink-0"
            aria-label={t("settings.common.copy")}
            onClick={() => void copy(user.id, "id")}
          >
            {copied === "id" ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          </Button>
        </span>
      ),
    },
    {
      label: t("settings.account_card.email"),
      value: (
        <span className="flex min-w-0 flex-wrap items-center gap-x-2">
          <span className="truncate">{user.email}</span>
          <Link href="/settings?tab=contacts" className="text-[12.5px] text-primary hover:underline">
            {t("settings.account_card.change")}
          </Link>
        </span>
      ),
    },
    { label: t("settings.account_card.created"), value: formatDate(user.created_at) },
    { label: t("settings.account_card.last_login"), value: formatDateTime(user.last_login_at) },
  ];
  if (user.staff) {
    rows.push({ label: t("settings.account_card.role"), value: t(`settings.account_card.role_${user.role}`) });
  }
  return (
    <SettingsSection title={t("settings.account_card.title")}>
      <dl className="divide-y divide-border rounded-xl border border-border">
        {rows.map((row) => (
          <div key={row.label} className="grid gap-1 px-4 py-3 sm:grid-cols-[200px_minmax(0,1fr)] sm:items-center sm:gap-4">
            <dt className="text-[12.5px] text-muted-foreground">{row.label}</dt>
            <dd className="min-w-0 text-[13.5px]">{row.value}</dd>
          </div>
        ))}
      </dl>
    </SettingsSection>
  );
}
