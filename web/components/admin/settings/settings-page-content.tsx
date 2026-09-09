"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { ChevronDown, ImageIcon, Upload } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import {
  fetchAdminSettings,
  saveAdminSettings,
  brandingUploadUrl,
  testAdminSettingsFilesStorage,
  testAdminSettingsMail,
  testAdminSettingsTelegram,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import {
  FILES_STORAGE_DRIVERS,
  isSettingsTab,
  MAIL_MAILERS,
  OAUTH_PROVIDERS,
  PAYMENT_FEE_PROVIDERS,
  SETTINGS_KEY_MAP,
  SETTINGS_TABS,
} from "./constants";
import {
  FieldGrid,
  Segmented,
  SelectField,
  SettingsCard,
  TextAreaField,
  TextField,
  ToggleRow,
} from "./settings-ui";
import type { PaymentProvider, SettingsTab } from "./types";

type SettingsPageContentProps = {
  initialTab?: SettingsTab;
};

type FilesDriver = (typeof FILES_STORAGE_DRIVERS)[number]["id"];

export function SettingsPageContent({ initialTab }: SettingsPageContentProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [tab, setTab] = useState<SettingsTab>(() => {
    if (initialTab) return initialTab;
    const tabParam = searchParams.get("tab");
    if (isSettingsTab(tabParam)) return tabParam;
    if (pathname.endsWith("/files")) return "files";
    return "main";
  });

  const [values, setValues] = useState<Record<string, string>>({});
  const [paymentProviders, setPaymentProviders] = useState<PaymentProvider[]>(
    []
  );
  const [openProviders, setOpenProviders] = useState<Record<string, boolean>>(
    {}
  );
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [baseline, setBaseline] = useState<{
    values: Record<string, string>;
    providers: PaymentProvider[];
  }>({ values: {}, providers: [] });

  const logoInputRef = useRef<HTMLInputElement>(null);
  const iconInputRef = useRef<HTMLInputElement>(null);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminSettings,
    queryFn: fetchAdminSettings,
    staleTime: 60 * 1000,
  });

  useEffect(() => {
    if (!data) return;
    const nextValues = data.values || {};
    const rawProviders = data.payment_providers ?? data.paymentProviders ?? [];
    const providers = Array.isArray(rawProviders)
      ? rawProviders
      : Object.values(rawProviders || {});
    setValues(nextValues);
    setPaymentProviders(providers);
    setBaseline({ values: nextValues, providers });
    setLogoFile(null);
    setIconFile(null);
  }, [data]);

  useEffect(() => {
    if (initialTab) {
      setTab(initialTab);
      return;
    }
    const tabParam = searchParams.get("tab");
    const nextTab = isSettingsTab(tabParam)
      ? tabParam
      : pathname.endsWith("/files")
        ? "files"
        : "main";
    setTab((current) => (current === nextTab ? current : nextTab));
  }, [pathname, searchParams, initialTab]);

  const panelVersion =
    String(
      data?.panel_version ??
        data?.panelVersion ??
        data?.values?.["app.version"] ??
        ""
    ).trim() || "—";

  const saveMutation = useMutation({
    mutationFn: saveAdminSettings,
    onSuccess: () => {
      toast.success(t("admin.settings.saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminSettings });
    },
    onError: (err: Error) => toast.error(err.message || t("common.save_failed")),
  });

  const testMailMutation = useMutation({
    mutationFn: testAdminSettingsMail,
    onSuccess: (res) =>
      toast.success(res?.message || t("admin.settings.mail.test_sent")),
    onError: (err: Error) =>
      toast.error(err.message || t("admin.settings.send_failed")),
  });

  const testTelegramMutation = useMutation({
    mutationFn: testAdminSettingsTelegram,
    onSuccess: (res) =>
      toast.success(res?.message || t("admin.settings.telegram.test_sent")),
    onError: (err: Error) =>
      toast.error(err.message || t("admin.settings.send_failed")),
  });

  const testFilesStorageMutation = useMutation({
    mutationFn: testAdminSettingsFilesStorage,
    onSuccess: (res) =>
      toast.success(res?.message || t("admin.settings.storage.test_ok")),
    onError: (err: Error) =>
      toast.error(err.message || t("admin.settings.storage.test_failed")),
  });

  const updateValue = (key: string, val: string) =>
    setValues((prev) => ({ ...prev, [key]: val }));

  const val = (key: string) => values[key] ?? "";
  const flag = (key: string) => values[key] === "1";
  const setFlag = (key: string) => (checked: boolean) =>
    updateValue(key, checked ? "1" : "0");

  const changeTab = (nextTab: SettingsTab) => {
    setTab(nextTab);
    if (nextTab === "files") {
      router.replace("/admin/settings/files");
      return;
    }
    if (nextTab === "main") {
      router.replace("/admin/settings");
      return;
    }
    const nextParams = new URLSearchParams(searchParams.toString());
    nextParams.set("tab", nextTab);
    router.replace(`/admin/settings?${nextParams.toString()}`);
  };

  const filesDriver: FilesDriver =
    values["files.storage.driver"] === "s3"
      ? "s3"
      : values["files.storage.driver"] === "sftp"
        ? "sftp"
        : "ftp";

  const isDirty = useMemo(() => {
    if (logoFile || iconFile) return true;
    const keys = new Set([
      ...Object.keys(baseline.values),
      ...Object.keys(values),
    ]);
    for (const key of keys) {
      if ((baseline.values[key] ?? "") !== (values[key] ?? "")) return true;
    }
    return (
      JSON.stringify(baseline.providers) !== JSON.stringify(paymentProviders)
    );
  }, [baseline, values, paymentProviders, logoFile, iconFile]);

  const reset = () => {
    setValues(baseline.values);
    setPaymentProviders(baseline.providers);
    setLogoFile(null);
    setIconFile(null);
    if (logoInputRef.current) logoInputRef.current.value = "";
    if (iconInputRef.current) iconInputRef.current.value = "";
  };

  const save = () => {
    const fd = new FormData();
    const normalizedValues: Record<string, string> = {
      ...values,
      "files.storage.driver": filesDriver,
    };
    Object.entries(SETTINGS_KEY_MAP).forEach(([dotKey, formKey]) => {
      fd.append(formKey, normalizedValues[dotKey] ?? "");
    });
    if (logoFile) fd.append("logo", logoFile);
    if (iconFile) fd.append("icon", iconFile);
    paymentProviders.forEach((pp) => {
      fd.append(`payment_providers[${pp.key}][enabled]`, pp.enabled ? "1" : "0");
      Object.entries(pp.config).forEach(([fk, fv]) =>
        fd.append(`payment_providers[${pp.key}][${fk}]`, fv)
      );
    });
    saveMutation.mutate(fd);
  };

  const testFilesStorage = () => {
    testFilesStorageMutation.mutate({
      files_storage_driver: filesDriver,
      files_storage_url: val("files.storage.url"),
      files_storage_ftp_host: val("files.storage.ftp.host"),
      files_storage_ftp_port: val("files.storage.ftp.port"),
      files_storage_ftp_username: val("files.storage.ftp.username"),
      files_storage_ftp_password: val("files.storage.ftp.password"),
      files_storage_ftp_root: val("files.storage.ftp.root"),
      files_storage_ftp_passive: flag("files.storage.ftp.passive"),
      files_storage_ftp_ssl: flag("files.storage.ftp.ssl"),
      files_storage_ftp_timeout: val("files.storage.ftp.timeout"),
      files_storage_sftp_host: val("files.storage.sftp.host"),
      files_storage_sftp_port: val("files.storage.sftp.port"),
      files_storage_sftp_username: val("files.storage.sftp.username"),
      files_storage_sftp_password: val("files.storage.sftp.password"),
      files_storage_sftp_root: val("files.storage.sftp.root"),
      files_storage_sftp_timeout: val("files.storage.sftp.timeout"),
      files_storage_s3_key: val("files.storage.s3.key"),
      files_storage_s3_secret: val("files.storage.s3.secret"),
      files_storage_s3_region: val("files.storage.s3.region"),
      files_storage_s3_bucket: val("files.storage.s3.bucket"),
      files_storage_s3_endpoint: val("files.storage.s3.endpoint"),
      files_storage_s3_use_path_style_endpoint: flag(
        "files.storage.s3.use_path_style_endpoint"
      ),
    });
  };

  const updateProviderField = (provKey: string, field: string, next: string) =>
    setPaymentProviders((prev) =>
      prev.map((p) =>
        p.key === provKey ? { ...p, config: { ...p.config, [field]: next } } : p
      )
    );

  const toggleProvider = (provKey: string) =>
    setPaymentProviders((prev) =>
      prev.map((p) => (p.key === provKey ? { ...p, enabled: !p.enabled } : p))
    );

  const tabTitleKey = SETTINGS_TABS.find((item) => item.id === tab)?.labelKey;
  const tabTitle = tabTitleKey ? t(tabTitleKey) : t("admin.settings.title");

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-6">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-96 w-full" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6 pb-24">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.settings.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.settings.subtitle")}
            </p>
          </div>
          <div className="space-y-1 text-right">
            <div className="font-mono text-[11px] tracking-wider text-muted-foreground uppercase">
              {t("admin.settings.panel_version")}
            </div>
            <div className="font-mono text-sm">{panelVersion}</div>
          </div>
        </div>

        <Segmented
          items={SETTINGS_TABS.map((item) => ({
            id: item.id,
            label: t(item.labelKey),
          }))}
          value={tab}
          onChange={changeTab}
        />

        {tab === "main" && (
          <div className="space-y-4">
            <SettingsCard
              title={t("admin.settings.project.title")}
              description={t("admin.settings.project.description")}
            >
              <FieldGrid>
                <TextField
                  span="full"
                  label={t("admin.settings.project.name")}
                  value={val("app.name")}
                  onChange={(v) => updateValue("app.name", v)}
                  placeholder="Vortanix"
                />
                <TextAreaField
                  span="full"
                  label={t("admin.settings.project.site_description")}
                  value={val("app.site.description")}
                  onChange={(v) => updateValue("app.site.description", v)}
                  placeholder={t(
                    "admin.settings.project.site_description_placeholder"
                  )}
                />
                <TextField
                  mono
                  label={t("admin.settings.project.domain")}
                  value={val("app.site.domain")}
                  onChange={(v) => updateValue("app.site.domain", v)}
                  placeholder="example.com"
                />
                <TextField
                  mono
                  label={t("admin.settings.project.site_ip")}
                  value={val("app.site.ip")}
                  onChange={(v) => updateValue("app.site.ip", v)}
                  placeholder="1.2.3.4"
                />
                <TextField
                  mono
                  label={t("admin.settings.project.subnet")}
                  value={val("app.site.subnet")}
                  onChange={(v) => updateValue("app.site.subnet", v)}
                  placeholder="1.2.3.0/24"
                />
                <TextField
                  label={t("admin.settings.project.default_template")}
                  value={val("app.site.default_template")}
                  onChange={(v) => updateValue("app.site.default_template", v)}
                  placeholder="default"
                />
              </FieldGrid>
            </SettingsCard>

            <SettingsCard
              title={t("admin.settings.trial.title")}
              description={t("admin.settings.trial.description")}
            >
              <div className="space-y-3">
                <ToggleRow
                  label={t("admin.settings.trial.enabled")}
                  hint={t("admin.settings.trial.enabled_hint")}
                  checked={flag("trial.enabled")}
                  onCheckedChange={setFlag("trial.enabled")}
                />
                <FieldGrid cols={3}>
                  <TextField
                    label={t("admin.settings.trial.hours")}
                    value={val("trial.hours")}
                    onChange={(v) => updateValue("trial.hours", v)}
                    placeholder="2"
                  />
                  <TextField
                    label={t("admin.settings.trial.cooldown")}
                    value={val("trial.cooldown_days")}
                    onChange={(v) => updateValue("trial.cooldown_days", v)}
                    placeholder="30"
                  />
                  <TextField
                    label={t("admin.settings.trial.memory")}
                    value={val("trial.memory_mb")}
                    onChange={(v) => updateValue("trial.memory_mb", v)}
                    placeholder={t("admin.settings.trial.memory_placeholder")}
                  />
                </FieldGrid>
                <TextField
                  span="full"
                  label={t("admin.settings.trial.games")}
                  value={val("trial.games")}
                  onChange={(v) => updateValue("trial.games", v)}
                  placeholder={t("admin.settings.trial.games_placeholder")}
                />
              </div>
            </SettingsCard>

            <SettingsCard title={t("admin.settings.links.title")}>
              <FieldGrid cols={3}>
                <TextField
                  label="Telegram"
                  value={val("app.links.telegram")}
                  onChange={(v) => updateValue("app.links.telegram", v)}
                  placeholder="https://t.me/example"
                />
                <TextField
                  label="Discord"
                  value={val("app.links.discord")}
                  onChange={(v) => updateValue("app.links.discord", v)}
                  placeholder="https://discord.gg/example"
                />
                <TextField
                  label="Support"
                  value={val("app.links.support")}
                  onChange={(v) => updateValue("app.links.support", v)}
                  placeholder="https://example.com/support"
                />
              </FieldGrid>
            </SettingsCard>

            <SettingsCard title={t("admin.settings.branding.title")}>
              <div className="grid gap-5 sm:grid-cols-2">
                <BrandingPicker
                  label={t("admin.settings.branding.logo")}
                  inputRef={logoInputRef}
                  file={logoFile}
                  currentPath={val("app.branding.logo")}
                  onSelect={setLogoFile}
                />
                <BrandingPicker
                  label={t("admin.settings.branding.icon")}
                  inputRef={iconInputRef}
                  file={iconFile}
                  currentPath={val("app.branding.icon")}
                  onSelect={setIconFile}
                />
              </div>
            </SettingsCard>

            <SettingsCard
              title={t("admin.settings.security.title")}
              description={t("admin.settings.security.description")}
            >
              <FieldGrid className="mb-4">
                <TextField
                  mono
                  label="ReCaptcha Site Key"
                  value={val("services.recaptcha.site_key")}
                  onChange={(v) => updateValue("services.recaptcha.site_key", v)}
                  placeholder="6Lc..."
                />
                <TextField
                  mono
                  type="password"
                  label="ReCaptcha Secret Key"
                  value={val("services.recaptcha.secret_key")}
                  onChange={(v) =>
                    updateValue("services.recaptcha.secret_key", v)
                  }
                />
              </FieldGrid>
              <div className="space-y-2.5">
                <ToggleRow
                  label={t("admin.settings.security.verify_email")}
                  hint={t("admin.settings.security.verify_email_hint")}
                  checked={flag("auth.require_verified_email")}
                  onCheckedChange={setFlag("auth.require_verified_email")}
                />
                <ToggleRow
                  label={t("admin.settings.security.status_mail")}
                  hint={t("admin.settings.security.status_mail_hint")}
                  checked={flag("vtx_mail.server_status_notifications")}
                  onCheckedChange={setFlag(
                    "vtx_mail.server_status_notifications"
                  )}
                />
              </div>
            </SettingsCard>
          </div>
        )}

        {tab === "payments" && (
          <div className="space-y-4">
            <SettingsCard
              title={t("admin.settings.fees.title")}
              description={t("admin.settings.fees.description")}
              action={
                <span className="font-mono text-[11px] text-muted-foreground">
                  %
                </span>
              }
            >
              <FieldGrid cols={4} className="gap-y-4">
                {PAYMENT_FEE_PROVIDERS.map((p) => (
                  <TextField
                    key={p.key}
                    mono
                    label={p.label}
                    value={values[`payments.providers.${p.key}.fee_percent`] ?? "0"}
                    onChange={(v) =>
                      updateValue(`payments.providers.${p.key}.fee_percent`, v)
                    }
                    placeholder="0"
                  />
                ))}
                <TextField
                  mono
                  accent
                  label={t("admin.settings.fees.fx")}
                  value={values["payments.fx.fee_percent"] ?? "0"}
                  onChange={(v) => updateValue("payments.fx.fee_percent", v)}
                  placeholder="0"
                />
              </FieldGrid>
            </SettingsCard>

            <SettingsCard
              title={t("admin.settings.providers.title")}
              description={t("admin.settings.providers.description")}
            >
              <div className="space-y-2.5">
                {paymentProviders.length === 0 && (
                  <div className="rounded-xl border border-dashed px-4 py-6 text-center text-[13px] text-muted-foreground">
                    {t("admin.settings.providers.empty")}
                  </div>
                )}
                {paymentProviders.map((pp) => {
                  const open = !!openProviders[pp.key];
                  return (
                    <div
                      key={pp.key}
                      className="rounded-xl border bg-muted/30 px-5 py-4"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-4">
                        <button
                          type="button"
                          onClick={() =>
                            setOpenProviders((prev) => ({
                              ...prev,
                              [pp.key]: !prev[pp.key],
                            }))
                          }
                          className="flex min-w-0 flex-1 items-center gap-3 text-left"
                        >
                          <ChevronDown
                            className={cn(
                              "size-4 shrink-0 text-muted-foreground transition-transform",
                              open && "rotate-180"
                            )}
                          />
                          <span className="min-w-0">
                            <span className="block truncate text-sm font-semibold">
                              {pp.name}
                            </span>
                            <span className="block truncate font-mono text-[11px] text-muted-foreground">
                              {pp.key}
                            </span>
                          </span>
                        </button>
                        <label className="flex items-center gap-3.5">
                          <span className="text-xs text-muted-foreground">
                            {pp.enabled
                              ? t("admin.settings.providers.enabled")
                              : t("admin.settings.providers.disabled")}
                          </span>
                          <Switch
                            checked={pp.enabled}
                            onCheckedChange={() => toggleProvider(pp.key)}
                          />
                        </label>
                      </div>
                      {open && (
                        <FieldGrid cols={3} className="mt-4 border-t pt-4">
                          {Object.entries(pp.fields).map(
                            ([fieldKey, field]) => {
                              if (field.type === "checkbox") {
                                return (
                                  <div
                                    key={fieldKey}
                                    className="sm:col-span-2 lg:col-span-3"
                                  >
                                    <ToggleRow
                                      label={field.label}
                                      checked={
                                        (pp.config[fieldKey] || "0") === "1"
                                      }
                                      onCheckedChange={(checked) =>
                                        updateProviderField(
                                          pp.key,
                                          fieldKey,
                                          checked ? "1" : "0"
                                        )
                                      }
                                    />
                                  </div>
                                );
                              }
                              if (field.type === "select" && field.options) {
                                return (
                                  <SelectField
                                    key={fieldKey}
                                    label={field.label}
                                    value={
                                      pp.config[fieldKey] || field.default || ""
                                    }
                                    onChange={(v) =>
                                      updateProviderField(pp.key, fieldKey, v)
                                    }
                                    options={Object.entries(field.options).map(
                                      ([value, label]) => ({ value, label })
                                    )}
                                  />
                                );
                              }
                              if (field.type === "textarea") {
                                return (
                                  <TextAreaField
                                    key={fieldKey}
                                    span={3}
                                    label={field.label}
                                    value={pp.config[fieldKey] || ""}
                                    onChange={(v) =>
                                      updateProviderField(pp.key, fieldKey, v)
                                    }
                                  />
                                );
                              }
                              return (
                                <TextField
                                  key={fieldKey}
                                  mono
                                  type={
                                    field.type === "password"
                                      ? "password"
                                      : "text"
                                  }
                                  label={field.label}
                                  value={pp.config[fieldKey] || ""}
                                  onChange={(v) =>
                                    updateProviderField(pp.key, fieldKey, v)
                                  }
                                />
                              );
                            }
                          )}
                        </FieldGrid>
                      )}
                    </div>
                  );
                })}
              </div>
            </SettingsCard>
          </div>
        )}

        {tab === "mail" && (
          <div className="space-y-4">
            <SettingsCard
              title="SMTP"
              description={t("admin.settings.mail.description")}
            >
              <FieldGrid cols={4}>
                <SelectField
                  label="Mailer"
                  value={val("mail.default") || "smtp"}
                  onChange={(v) => updateValue("mail.default", v)}
                  options={MAIL_MAILERS}
                />
                <TextField
                  mono
                  label="Scheme"
                  value={val("mail.mailers.smtp.scheme")}
                  onChange={(v) => updateValue("mail.mailers.smtp.scheme", v)}
                  placeholder="tls"
                />
                <TextField
                  mono
                  span={2}
                  label="Host"
                  value={val("mail.mailers.smtp.host")}
                  onChange={(v) => updateValue("mail.mailers.smtp.host", v)}
                  placeholder="smtp.example.com"
                />
                <TextField
                  mono
                  label="Port"
                  value={val("mail.mailers.smtp.port")}
                  onChange={(v) => updateValue("mail.mailers.smtp.port", v)}
                  placeholder="587"
                />
                <TextField
                  label="Username"
                  value={val("mail.mailers.smtp.username")}
                  onChange={(v) => updateValue("mail.mailers.smtp.username", v)}
                />
                <TextField
                  type="password"
                  label="Password"
                  value={val("mail.mailers.smtp.password")}
                  onChange={(v) => updateValue("mail.mailers.smtp.password", v)}
                />
                <TextField
                  label="From name"
                  value={val("mail.from.name")}
                  onChange={(v) => updateValue("mail.from.name", v)}
                  placeholder="Vortanix"
                />
                <TextField
                  mono
                  span={2}
                  label="From address"
                  value={val("mail.from.address")}
                  onChange={(v) => updateValue("mail.from.address", v)}
                  placeholder="no-reply@example.com"
                />
              </FieldGrid>
            </SettingsCard>

            <SettingsCard>
              <div className="flex flex-wrap items-end gap-4">
                <div className="min-w-[280px] flex-1">
                  <TextField
                    mono
                    label={t("admin.settings.mail.test_to")}
                    value={val("test_mail_to")}
                    onChange={(v) => updateValue("test_mail_to", v)}
                    placeholder="test@example.com"
                  />
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="h-[38px] text-[13px]"
                  onClick={() => testMailMutation.mutate(val("test_mail_to"))}
                  disabled={testMailMutation.isPending}
                >
                  {testMailMutation.isPending
                    ? t("common.sending")
                    : t("admin.settings.mail.send_test")}
                </Button>
              </div>
            </SettingsCard>
          </div>
        )}

        {tab === "social" && (
          <SettingsCard
            title={t("admin.settings.oauth.title")}
            description={t("admin.settings.oauth.description")}
          >
            <div className="space-y-3">
              {OAUTH_PROVIDERS.map((provider) => (
                <div
                  key={provider.slug}
                  className="rounded-xl border bg-muted/30 px-5 py-5"
                >
                  <div className="mb-4 text-sm font-semibold">
                    {provider.name}
                  </div>
                  <FieldGrid>
                    <TextField
                      mono
                      label="Client ID"
                      value={val(`${provider.prefix}.client_id`)}
                      onChange={(v) =>
                        updateValue(`${provider.prefix}.client_id`, v)
                      }
                    />
                    <TextField
                      mono
                      type="password"
                      label="Client Secret"
                      value={val(`${provider.prefix}.client_secret`)}
                      onChange={(v) =>
                        updateValue(`${provider.prefix}.client_secret`, v)
                      }
                    />
                    <TextField
                      mono
                      span="full"
                      label="Redirect URI"
                      value={val(`${provider.prefix}.redirect`)}
                      onChange={(v) =>
                        updateValue(`${provider.prefix}.redirect`, v)
                      }
                      placeholder={`https://example.com/auth/${provider.slug}/callback`}
                    />
                  </FieldGrid>
                </div>
              ))}
            </div>
          </SettingsCard>
        )}

        {tab === "dockerhub" && (
          <SettingsCard
            title="Docker Hub"
            description={t("admin.settings.dockerhub.description")}
          >
            <FieldGrid cols={2}>
              <TextField
                label={t("admin.settings.dockerhub.username")}
                value={val("dockerhub.username")}
                onChange={(v) => updateValue("dockerhub.username", v)}
                placeholder="vortanix"
              />
              <TextField
                mono
                label="Access Token"
                value={val("dockerhub.token")}
                onChange={(v) => updateValue("dockerhub.token", v)}
                placeholder="dckr_pat_..."
              />
            </FieldGrid>
            <p className="mt-3 text-[12.5px] text-muted-foreground">
              {t("admin.settings.dockerhub.token_hint")}
            </p>
          </SettingsCard>
        )}

        {tab === "telegram" && (
          <SettingsCard
            title={t("admin.settings.telegram.title")}
            description={t("admin.settings.telegram.description")}
            action={
              <Switch
                checked={flag("telegram.notifications.enabled")}
                onCheckedChange={setFlag("telegram.notifications.enabled")}
              />
            }
          >
            <FieldGrid cols={3}>
              <TextField
                mono
                label="Bot Token"
                value={val("telegram.notifications.bot_token")}
                onChange={(v) =>
                  updateValue("telegram.notifications.bot_token", v)
                }
                placeholder="123456:ABC-DEF..."
              />
              <TextField
                mono
                label="Bot Username"
                value={val("telegram.notifications.bot_username")}
                onChange={(v) =>
                  updateValue("telegram.notifications.bot_username", v)
                }
                placeholder="VortanixBot"
              />
              <TextField
                mono
                label="Admin Chat ID"
                value={val("telegram.notifications.admin_chat_id")}
                onChange={(v) =>
                  updateValue("telegram.notifications.admin_chat_id", v)
                }
                placeholder="-1001234567890"
              />
            </FieldGrid>
            <div className="mt-5 flex flex-wrap items-center gap-4">
              <Button
                type="button"
                variant="outline"
                className="h-[38px] text-[13px]"
                onClick={() => testTelegramMutation.mutate()}
                disabled={testTelegramMutation.isPending}
              >
                {testTelegramMutation.isPending
                  ? t("common.sending")
                  : t("admin.settings.telegram.send_test")}
              </Button>
              <span className="text-xs text-muted-foreground">
                {t("admin.settings.telegram.test_hint")}
              </span>
            </div>
          </SettingsCard>
        )}

        {tab === "files" && (
          <div className="space-y-4">
            <SettingsCard
              title={t("admin.settings.paths.title")}
              description={t("admin.settings.paths.description")}
            >
              <FieldGrid cols={4}>
                <TextField
                  mono
                  label={t("admin.settings.paths.avatars")}
                  value={val("files.storage.path.avatars")}
                  onChange={(v) => updateValue("files.storage.path.avatars", v)}
                  placeholder="avatars"
                />
                <TextField
                  mono
                  label={t("admin.settings.paths.plugins")}
                  value={val("files.storage.path.plugins")}
                  onChange={(v) => updateValue("files.storage.path.plugins", v)}
                  placeholder="plugins"
                />
                <TextField
                  mono
                  label={t("admin.settings.paths.maps")}
                  value={val("files.storage.path.maps")}
                  onChange={(v) => updateValue("files.storage.path.maps", v)}
                  placeholder="maps"
                />
                <TextField
                  mono
                  label={t("admin.settings.paths.support")}
                  value={val("files.storage.path.support")}
                  onChange={(v) => updateValue("files.storage.path.support", v)}
                  placeholder="support"
                />
              </FieldGrid>
            </SettingsCard>

            <SettingsCard
              title={t("admin.settings.storage.title")}
              description={t("admin.settings.storage.description")}
              action={
                <Segmented
                  size="sm"
                  items={FILES_STORAGE_DRIVERS}
                  value={filesDriver}
                  onChange={(next) => updateValue("files.storage.driver", next)}
                />
              }
            >
              <div className="mb-5">
                <TextField
                  mono
                  label={t("admin.settings.storage.public_url")}
                  value={val("files.storage.url")}
                  onChange={(v) => updateValue("files.storage.url", v)}
                  placeholder="https://cdn.example.com/support"
                />
              </div>

              {filesDriver === "ftp" && (
                <div className="space-y-4">
                  <FieldGrid cols={3}>
                    <TextField
                      mono
                      label="FTP Host"
                      value={val("files.storage.ftp.host")}
                      onChange={(v) => updateValue("files.storage.ftp.host", v)}
                      placeholder="127.0.0.1"
                    />
                    <TextField
                      mono
                      label="Port"
                      value={val("files.storage.ftp.port")}
                      onChange={(v) => updateValue("files.storage.ftp.port", v)}
                      placeholder="21"
                    />
                    <TextField
                      mono
                      label={t("admin.settings.storage.timeout")}
                      value={val("files.storage.ftp.timeout")}
                      onChange={(v) =>
                        updateValue("files.storage.ftp.timeout", v)
                      }
                      placeholder="30"
                    />
                    <TextField
                      label="Username"
                      value={val("files.storage.ftp.username")}
                      onChange={(v) =>
                        updateValue("files.storage.ftp.username", v)
                      }
                    />
                    <TextField
                      type="password"
                      label="Password"
                      value={val("files.storage.ftp.password")}
                      onChange={(v) =>
                        updateValue("files.storage.ftp.password", v)
                      }
                    />
                    <TextField
                      mono
                      label="Root"
                      value={val("files.storage.ftp.root")}
                      onChange={(v) => updateValue("files.storage.ftp.root", v)}
                      placeholder="/uploads/support"
                    />
                  </FieldGrid>
                  <div className="space-y-2.5">
                    <ToggleRow
                      label="Passive mode"
                      hint={t("admin.settings.storage.passive_hint")}
                      checked={flag("files.storage.ftp.passive")}
                      onCheckedChange={setFlag("files.storage.ftp.passive")}
                    />
                    <ToggleRow
                      label="FTP over SSL"
                      hint={t("admin.settings.storage.ssl_hint")}
                      checked={flag("files.storage.ftp.ssl")}
                      onCheckedChange={setFlag("files.storage.ftp.ssl")}
                    />
                  </div>
                </div>
              )}

              {filesDriver === "sftp" && (
                <FieldGrid cols={3}>
                  <TextField
                    mono
                    label="SFTP Host"
                    value={val("files.storage.sftp.host")}
                    onChange={(v) => updateValue("files.storage.sftp.host", v)}
                    placeholder="127.0.0.1"
                  />
                  <TextField
                    mono
                    label="Port"
                    value={val("files.storage.sftp.port")}
                    onChange={(v) => updateValue("files.storage.sftp.port", v)}
                    placeholder="22"
                  />
                  <TextField
                    mono
                    label={t("admin.settings.storage.timeout")}
                    value={val("files.storage.sftp.timeout")}
                    onChange={(v) =>
                      updateValue("files.storage.sftp.timeout", v)
                    }
                    placeholder="30"
                  />
                  <TextField
                    label="Username"
                    value={val("files.storage.sftp.username")}
                    onChange={(v) =>
                      updateValue("files.storage.sftp.username", v)
                    }
                  />
                  <TextField
                    type="password"
                    label="Password"
                    value={val("files.storage.sftp.password")}
                    onChange={(v) =>
                      updateValue("files.storage.sftp.password", v)
                    }
                  />
                  <TextField
                    mono
                    label="Root"
                    value={val("files.storage.sftp.root")}
                    onChange={(v) => updateValue("files.storage.sftp.root", v)}
                    placeholder="/var/www/files"
                  />
                </FieldGrid>
              )}

              {filesDriver === "s3" && (
                <div className="space-y-4">
                  <FieldGrid cols={3}>
                    <TextField
                      mono
                      label="Access Key"
                      value={val("files.storage.s3.key")}
                      onChange={(v) => updateValue("files.storage.s3.key", v)}
                    />
                    <TextField
                      mono
                      type="password"
                      label="Secret Key"
                      value={val("files.storage.s3.secret")}
                      onChange={(v) => updateValue("files.storage.s3.secret", v)}
                    />
                    <TextField
                      mono
                      label="Region"
                      value={val("files.storage.s3.region")}
                      onChange={(v) => updateValue("files.storage.s3.region", v)}
                      placeholder="us-east-1"
                    />
                    <TextField
                      mono
                      label="Bucket"
                      value={val("files.storage.s3.bucket")}
                      onChange={(v) => updateValue("files.storage.s3.bucket", v)}
                      placeholder="vortanix-support"
                    />
                    <TextField
                      mono
                      span={2}
                      label={t("admin.settings.storage.endpoint")}
                      value={val("files.storage.s3.endpoint")}
                      onChange={(v) =>
                        updateValue("files.storage.s3.endpoint", v)
                      }
                      placeholder="https://s3.amazonaws.com"
                    />
                  </FieldGrid>
                  <ToggleRow
                    label="Path-style endpoint"
                    hint={t("admin.settings.storage.path_style_hint")}
                    checked={flag("files.storage.s3.use_path_style_endpoint")}
                    onCheckedChange={setFlag(
                      "files.storage.s3.use_path_style_endpoint"
                    )}
                  />
                </div>
              )}

              <div className="mt-5 flex flex-wrap items-center gap-4">
                <Button
                  type="button"
                  variant="outline"
                  className="h-[38px] text-[13px]"
                  onClick={testFilesStorage}
                  disabled={testFilesStorageMutation.isPending}
                >
                  {testFilesStorageMutation.isPending
                    ? t("admin.settings.storage.testing")
                    : t("admin.settings.storage.test")}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {t("admin.settings.storage.test_hint")}
                </span>
              </div>
            </SettingsCard>
          </div>
        )}
      </div>

      <div className="sticky bottom-0 -mx-4 -mb-6 px-4 pt-4 pb-5">
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-background via-background/90 to-transparent" />
        <div className="relative flex w-full flex-wrap items-center justify-between gap-4 rounded-2xl border bg-card px-5 py-3.5">
          <div className="flex items-center gap-2.5">
            <span
              className={cn(
                "size-1.5 rounded-full",
                isDirty ? "bg-amber-500" : "bg-muted-foreground/40"
              )}
            />
            <span className="text-[13px] text-muted-foreground">
              {isDirty
                ? t("admin.settings.dirty", { tab: tabTitle })
                : t("admin.settings.clean", { tab: tabTitle })}
            </span>
          </div>
          <div className="flex items-center gap-2.5">
            <Button
              type="button"
              variant="ghost"
              className="h-9 text-[13px] text-muted-foreground"
              onClick={reset}
              disabled={!isDirty || saveMutation.isPending}
            >
              {t("admin.settings.discard")}
            </Button>
            <Button
              type="button"
              className="h-9 px-5 text-[13px]"
              onClick={save}
              disabled={saveMutation.isPending}
            >
              {saveMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </div>
      </div>
    </PageShell>
  );
}

function BrandingPicker({
  label,
  file,
  currentPath,
  onSelect,
  inputRef,
}: {
  label: string;
  file: File | null;
  currentPath: string;
  onSelect: (file: File | null) => void;
  inputRef: React.RefObject<HTMLInputElement | null>;
}) {
  const t = useT();
  const previewUrl = currentPath ? brandingUploadUrl(currentPath) : "";
  const name =
    file?.name || currentPath || t("admin.settings.branding.no_file");

  return (
    <div className="flex flex-col gap-2.5">
      <Label className="text-xs font-normal text-muted-foreground">
        {label}
      </Label>
      <div className="flex items-center gap-3.5 rounded-xl border border-dashed bg-muted/30 p-3.5">
        <div className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted">
          {previewUrl && !file ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={previewUrl}
              alt={label}
              className="size-full object-contain"
            />
          ) : (
            <ImageIcon className="size-4 text-muted-foreground" />
          )}
        </div>
        <div className="flex min-w-0 flex-col gap-1.5">
          <span
            className={cn(
              "truncate font-mono text-xs",
              file || currentPath ? "text-foreground" : "text-muted-foreground"
            )}
          >
            {name}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-[26px] w-fit gap-1.5 px-2.5 text-xs"
            onClick={() => inputRef.current?.click()}
          >
            <Upload className="size-3" />
            {file || currentPath
              ? t("admin.settings.branding.replace")
              : t("admin.settings.branding.choose")}
          </Button>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(e) => onSelect(e.target.files?.[0] || null)}
        />
      </div>
    </div>
  );
}
